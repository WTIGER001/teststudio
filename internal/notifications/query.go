package notifications

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Page struct {
	Items   []storage.Record `json:"items"`
	Cursor  string           `json:"cursor,omitempty"`
	Scanned int              `json:"scanned"`
	Partial bool             `json:"partial"`
}

func (s *Service) authorizeRead(ctx context.Context, user string, v Delivery) error {
	if err := s.Auth.Check(ctx, user, v.Project, "notification/"+v.Channel, "notifications.read"); err != nil {
		return err
	}
	return s.Auth.Check(ctx, user, v.Project, "run/"+v.Run, "runs.read")
}
func (s *Service) ReadDelivery(ctx context.Context, user, project, id string) (Delivery, storage.Record, error) {
	v, row, err := s.delivery(ctx, project, id)
	if err != nil {
		return v, row, err
	}
	if err = s.authorizeRead(ctx, user, v); err != nil {
		return Delivery{}, storage.Record{}, err
	}
	return v, row, nil
}
func (s *Service) ListDeliveries(ctx context.Context, user, project, run, cursor string, limit int) (Page, error) {
	if err := s.Auth.Check(ctx, user, project, "", "notifications.read"); err != nil {
		return Page{}, err
	}
	if limit < 1 || limit > 100 || (run != "" && !domain.ValidID(run)) {
		return Page{}, errors.New("notification query requires a valid run and a limit from 1 through 100")
	}
	if run != "" {
		if err := s.Auth.Check(ctx, user, project, "run/"+run, "runs.read"); err != nil {
			return Page{}, err
		}
	}
	position := ""
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return Page{}, errors.New("notification cursor is invalid")
		}
		var value struct{ Project, Run, Position string }
		if json.Unmarshal(data, &value) != nil || value.Project != project || value.Run != run || value.Position == "" {
			return Page{}, errors.New("notification cursor does not match the query")
		}
		position = value.Position
	}
	result := Page{Items: []storage.Record{}}
	for result.Scanned < 500 && len(result.Items) < limit {
		page, err := s.Store.ListNewest(ctx, Deliveries(project), project, position, min(50, limit-len(result.Items), 500-result.Scanned))
		if err != nil {
			return result, err
		}
		for _, row := range page.Items {
			result.Scanned++
			v, err := storage.Decode[Delivery](row)
			if err != nil {
				return result, err
			}
			if run != "" && v.Run != run {
				continue
			}
			err = s.authorizeRead(ctx, user, v)
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			if err != nil {
				return result, err
			}
			result.Items = append(result.Items, row)
		}
		position = page.Cursor
		if position == "" {
			break
		}
	}
	if position != "" {
		data, _ := json.Marshal(struct{ Project, Run, Position string }{project, run, position})
		result.Cursor = base64.RawURLEncoding.EncodeToString(data)
	}
	result.Partial = result.Scanned >= 500 && position != ""
	return result, nil
}
func (s *Service) History(ctx context.Context, user, project, id, cursor string, limit int) (storage.Page, error) {
	if _, _, err := s.ReadDelivery(ctx, user, project, id); err != nil {
		return storage.Page{}, err
	}
	if limit < 1 || limit > 100 {
		return storage.Page{}, errors.New("notification attempt page limit must be between 1 and 100")
	}
	return s.Store.ListNewest(ctx, Attempts(id), project, cursor, limit)
}
func (s *Service) Retry(ctx context.Context, user, project, id string, revision int64) error {
	v, row, err := s.ReadDelivery(ctx, user, project, id)
	if err != nil {
		return err
	}
	if err = s.Auth.Check(ctx, user, project, "notification/"+v.Channel, "notifications.manage"); err != nil {
		return err
	}
	if row.Revision != revision {
		return storage.ErrConflict
	}
	if v.State != "failed" && v.State != "blocked" && v.State != "cancelled" && v.State != "retrying" {
		return errors.New("only stopped or waiting deliveries can retry")
	}
	current, _, err := s.channel(ctx, project, v.Channel)
	if err != nil {
		return err
	}
	candidate := v
	candidate.DispatchEpoch = current.Epoch
	if state, reason, err := s.check(ctx, candidate); err != nil {
		return errors.New("notification checks are unavailable")
	} else if state != "" {
		return errors.New(reason)
	}
	if v.Attempts+v.Destination.Channel.MaxAttempts > 1000 {
		return errors.New("delivery exceeds its lifetime attempt limit; create a new run")
	}
	old := v
	v.DispatchEpoch = current.Epoch
	v.AttemptLimit = v.Attempts + v.Destination.Channel.MaxAttempts
	v.State = "pending"
	v.NextAt = s.now()
	v.Error = ""
	v.Claim = ""
	v.ClaimUntil = nil
	v.UpdatedAt = s.now()
	c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
	if err != nil {
		return err
	}
	jRow, err := s.Store.Get(ctx, "notification-work", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	j, err := storage.Put("notification-work", id, project, jRow.Revision, job{Project: project, Delivery: id})
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "notification.delivery.retry", id, old, v)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, c, j, audit)
}
func (s *Service) Cancel(ctx context.Context, user, project, id string, revision int64) error {
	v, row, err := s.ReadDelivery(ctx, user, project, id)
	if err != nil {
		return err
	}
	if err = s.Auth.Check(ctx, user, project, "notification/"+v.Channel, "notifications.manage"); err != nil {
		return err
	}
	if row.Revision != revision {
		return storage.ErrConflict
	}
	if v.State != "pending" && v.State != "retrying" {
		return errors.New("only pending or waiting deliveries can stop")
	}
	old := v
	v.State = "cancelled"
	v.Error = "Delivery retries stopped by a project administrator."
	v.Claim = ""
	v.ClaimUntil = nil
	v.UpdatedAt = s.now()
	c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
	if err != nil {
		return err
	}
	jRow, err := s.Store.Get(ctx, "notification-work", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	changes := []storage.Change{c}
	if err == nil {
		changes = append(changes, storage.Change{Record: jRow, Delete: true})
	}
	audit, err := domain.Audit(user, project, "notification.delivery.cancel", id, old, v)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, append(changes, audit)...)
}
