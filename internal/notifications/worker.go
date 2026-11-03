package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() { s.loop(ctx) })
	}
	go func() { workers.Wait(); close(done) }()
	return done
}
func (s *Service) loop(ctx context.Context) {
	interval := s.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	cursor := ""
	for {
		if ctx.Err() != nil {
			return
		}
		cursor, _ = s.Tick(ctx, cursor)
		if cursor != "" {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Service) Tick(ctx context.Context, cursor string) (string, error) {
	page, err := s.Store.List(ctx, "notification-work", "", cursor, 50)
	if err != nil {
		return "", err
	}
	var problems []error
	for _, row := range page.Items {
		if ctx.Err() != nil {
			return cursor, ctx.Err()
		}
		j, err := storage.Decode[job](row)
		if err == nil {
			err = s.process(ctx, j, row)
		}
		if err != nil && !errors.Is(err, storage.ErrConflict) {
			problems = append(problems, err)
		}
	}
	return page.Cursor, errors.Join(problems...)
}
func (s *Service) delivery(ctx context.Context, project, id string) (Delivery, storage.Record, error) {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return Delivery{}, storage.Record{}, errors.New("valid notification identifiers are required")
	}
	row, err := s.Store.Get(ctx, Deliveries(project), id)
	if err != nil {
		return Delivery{}, row, err
	}
	v, err := storage.Decode[Delivery](row)
	if err == nil && (v.Project != project || row.Project != project || v.ID != id) {
		err = storage.ErrNotFound
	}
	return v, row, err
}
func cooldownKey(v Delivery) string {
	sum := sha256.Sum256([]byte(v.Destination.Channel.Kind + "/" + v.Destination.Channel.Reference))
	return hex.EncodeToString(sum[:])
}
func (s *Service) process(parent context.Context, j job, index storage.Record) error {
	v, row, err := s.delivery(parent, j.Project, j.Delivery)
	if err != nil {
		return err
	}
	if index.Project != v.Project {
		return errors.New("notification work ownership does not match its project")
	}
	if v.State == "delivered" || v.State == "failed" || v.State == "blocked" || v.State == "cancelled" {
		return s.Store.Commit(parent, storage.Change{Record: index, Delete: true}, storage.Change{Record: row, Assert: true})
	}
	now := s.now()
	if now.Before(v.NextAt) || (v.ClaimUntil != nil && now.Before(*v.ClaimUntil)) {
		return nil
	}
	changes := []storage.Change{}
	if v.Claim != "" {
		attempt, aRow, err := s.attempt(parent, v, v.Attempts)
		if err != nil {
			return err
		}
		attempt.State = "unknown"
		attempt.FinishedAt = &now
		attempt.AcceptanceUnknown = true
		attempt.Error = "Delivery worker stopped before its checkpoint. Remote acceptance is unknown."
		c, err := storage.Put(aRow.Bucket, aRow.ID, aRow.Project, aRow.Revision, attempt)
		if err != nil {
			return err
		}
		changes = append(changes, c)
		v.AcceptanceUnknown = true
	}
	if v.Attempts >= v.AttemptLimit {
		v.State = "failed"
		v.Claim = ""
		v.ClaimUntil = nil
		v.Error = "Delivery attempt budget is exhausted. Remote acceptance may be unknown."
		v.UpdatedAt = now
		c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
		if err != nil {
			return err
		}
		return s.Store.Commit(parent, append(changes, c, storage.Change{Record: index, Delete: true})...)
	}
	at, err := s.cooldown(parent, "notification-cooldowns/"+v.Project, cooldownKey(v))
	if err != nil {
		return err
	}
	if v.Destination.Channel.Kind == "discord" {
		global, err := s.cooldown(parent, "notification-provider-cooldowns", "discord")
		if err != nil {
			return err
		}
		at = maxTime(at, global)
	}
	if now.Before(at) {
		v.Claim = ""
		v.ClaimUntil = nil
		v.State = "retrying"
		v.NextAt = at
		v.UpdatedAt = now
		v.Error = "Waiting for the notification destination rate limit."
		c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
		if err != nil {
			return err
		}
		return s.Store.Commit(parent, append(changes, c, storage.Change{Record: index, Assert: true})...)
	}
	v.Attempts++
	v.Claim = domain.NewID()
	until := now.Add(30 * time.Second)
	v.ClaimUntil = &until
	v.State = "sending"
	v.UpdatedAt = now
	c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
	if err != nil {
		return err
	}
	a, err := storage.Put(Attempts(v.ID), fmt.Sprintf("%04d", v.Attempts), v.Project, 0, Attempt{Number: v.Attempts, State: "sending", StartedAt: now})
	if err != nil {
		return err
	}
	if err = s.Store.Commit(parent, append(changes, c, a, storage.Change{Record: index, Assert: true})...); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	state, reason, checkErr := s.check(ctx, v)
	result := Outcome{Error: reason}
	if checkErr != nil {
		result = Outcome{Error: "Notification checks are temporarily unavailable.", Retryable: true}
	} else if state == "" {
		if s.Secrets == nil || s.Sender == nil {
			result = Outcome{Error: "Notification delivery dependencies are unavailable.", Retryable: true}
		} else {
			value, err := s.Secrets.Resolve(ctx, v.Project, v.Destination.Channel.Reference, secrets.NotificationWebhook)
			if err != nil {
				result = Outcome{Error: "Notification secret reference is unavailable.", Retryable: true}
			} else {
				result = s.Sender.Send(ctx, v.Destination.Channel.Kind, value, v.Message)
			}
		}
	}
	if parent.Err() != nil {
		return parent.Err()
	}
	if state == "" && result.Error == "" && (result.HTTPStatus < 200 || result.HTTPStatus >= 300) {
		result.Error = "Notification transport did not confirm acceptance."
	}
	return s.finish(parent, v, index, state, result)
}
func (s *Service) check(ctx context.Context, v Delivery) (string, string, error) {
	channel, _, err := s.channel(ctx, v.Project, v.Channel)
	if errors.Is(err, storage.ErrNotFound) {
		return "blocked", "Notification channel is unavailable.", nil
	}
	if err != nil {
		return "", "", err
	}
	if channel.Archived || !channel.Enabled {
		return "cancelled", "Notification channel is paused or archived.", nil
	}
	if channel.Kind != v.Destination.Channel.Kind || channel.Reference != v.Destination.Channel.Reference {
		return "cancelled", "Notification destination changed after this run was queued.", nil
	}
	if channel.Epoch != v.DispatchEpoch {
		return "cancelled", "Channel was paused or its destination changed after this run was queued.", nil
	}
	row, err := s.Store.Get(ctx, "projects", v.Project)
	if errors.Is(err, storage.ErrNotFound) {
		return "blocked", "Notification project is unavailable.", nil
	}
	if err != nil {
		return "", "", err
	}
	project, err := storage.Decode[projects.Project](row)
	if err != nil {
		return "", "", err
	}
	if project.Archived {
		return "cancelled", "Notification project is archived.", nil
	}
	for _, permission := range []struct{ object, name, reason string }{{"notification/" + channel.ID, "notifications.manage", "Current channel owner cannot manage notifications."}, {"run/" + v.Run, "runs.read", "Current channel owner cannot read this run."}} {
		err := s.Auth.Check(ctx, channel.Owner, v.Project, permission.object, permission.name)
		if errors.Is(err, authorization.ErrForbidden) || errors.Is(err, storage.ErrNotFound) {
			return "blocked", permission.reason, nil
		}
		if err != nil {
			return "", "", err
		}
	}
	return "", "", nil
}
func (s *Service) attempt(ctx context.Context, v Delivery, number int) (Attempt, storage.Record, error) {
	row, err := s.Store.Get(ctx, Attempts(v.ID), fmt.Sprintf("%04d", number))
	if err != nil {
		return Attempt{}, row, err
	}
	a, err := storage.Decode[Attempt](row)
	if row.Project != v.Project {
		return a, row, storage.ErrNotFound
	}
	return a, row, err
}
func delay(id string, attempt int, after time.Duration) time.Duration {
	base := min(15*time.Minute, 5*time.Second*time.Duration(1<<min(max(0, attempt-1), 8)))
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s/%d", id, attempt)))
	jitter := time.Duration(sum[0]) * time.Millisecond * 4
	return max(base+jitter, min(time.Hour, after))
}
func (s *Service) finish(ctx context.Context, claimed Delivery, index storage.Record, state string, result Outcome) error {
	for range 8 {
		v, row, err := s.delivery(ctx, claimed.Project, claimed.ID)
		if err != nil {
			return err
		}
		now := s.now()
		if v.Claim != claimed.Claim || v.ClaimUntil == nil || !now.Before(*v.ClaimUntil) {
			return storage.ErrConflict
		}
		v.Claim = ""
		v.ClaimUntil = nil
		v.HTTPStatus = result.HTTPStatus
		v.Error = result.Error
		v.AcceptanceUnknown = v.AcceptanceUnknown || result.AcceptanceUnknown
		v.UpdatedAt = now
		if state != "" {
			v.State = state
		} else if result.Error == "" && result.HTTPStatus >= 200 && result.HTTPStatus < 300 {
			v.State = "delivered"
			v.DeliveredAt = &now
		} else if result.Retryable && v.Attempts < v.AttemptLimit {
			v.State = "retrying"
			v.NextAt = now.Add(delay(v.ID, v.Attempts, result.RetryAfter))
		} else {
			v.State = "failed"
		}
		a, aRow, err := s.attempt(ctx, v, v.Attempts)
		if err != nil {
			return err
		}
		a.State = v.State
		a.FinishedAt = &now
		a.HTTPStatus = result.HTTPStatus
		a.Error = result.Error
		a.AcceptanceUnknown = result.AcceptanceUnknown
		c, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, v)
		if err != nil {
			return err
		}
		ac, err := storage.Put(aRow.Bucket, aRow.ID, aRow.Project, aRow.Revision, a)
		if err != nil {
			return err
		}
		changes := []storage.Change{c, ac}
		if v.State != "retrying" {
			changes = append(changes, storage.Change{Record: index, Delete: true})
		} else {
			changes = append(changes, storage.Change{Record: index, Assert: true})
		}
		if result.Retryable && (result.HTTPStatus == 429 || result.RetryAfter > 0) {
			at := now.Add(delay(v.ID, v.Attempts, result.RetryAfter))
			cc, err := s.cooldownChange(ctx, "notification-cooldowns/"+v.Project, cooldownKey(v), v.Project, at)
			if err != nil {
				return err
			}
			changes = append(changes, cc)
			if result.GlobalRateLimit && v.Destination.Channel.Kind == "discord" {
				cc, err := s.cooldownChange(ctx, "notification-provider-cooldowns", "discord", "", at)
				if err != nil {
					return err
				}
				changes = append(changes, cc)
			}
		}

		err = s.Store.Commit(ctx, changes...)
		if !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return storage.ErrConflict
}
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func (s *Service) cooldown(ctx context.Context, bucket, id string) (time.Time, error) {
	row, err := s.Store.Get(ctx, bucket, id)
	if errors.Is(err, storage.ErrNotFound) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return storage.Decode[time.Time](row)
}
func (s *Service) cooldownChange(ctx context.Context, bucket, id, project string, at time.Time) (storage.Change, error) {
	row, err := s.Store.Get(ctx, bucket, id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.Change{}, err
	}
	if err == nil {
		old, err := storage.Decode[time.Time](row)
		if err != nil {
			return storage.Change{}, err
		}
		at = maxTime(at, old)
	}
	return storage.Put(bucket, id, project, row.Revision, at)
}
