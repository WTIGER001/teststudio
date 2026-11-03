package notifications

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store        *storage.Store
	Auth         *authorization.Service
	Secrets      secrets.Provider
	Sender       Sender
	Origin       string
	Now          func() time.Time
	PollInterval time.Duration
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
func Channels(project string) string   { return "notification-channels/" + project }
func Deliveries(project string) string { return "notification-deliveries/" + project }
func Attempts(id string) string        { return "notification-attempts/" + id }
func (s *Service) channel(ctx context.Context, project, id string) (Channel, storage.Record, error) {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return Channel{}, storage.Record{}, errors.New("valid project and notification identifiers are required")
	}
	row, err := s.Store.Get(ctx, Channels(project), id)
	if err != nil {
		return Channel{}, row, err
	}
	v, err := storage.Decode[Channel](row)
	if err == nil && (v.Project != project || row.Project != project) {
		err = storage.ErrNotFound
	}
	return v, row, err
}
func (s *Service) ReadChannel(ctx context.Context, user, project, id string) (Channel, storage.Record, error) {
	if err := s.Auth.Check(ctx, user, project, "notification/"+id, "notifications.read"); err != nil {
		return Channel{}, storage.Record{}, err
	}
	return s.channel(ctx, project, id)
}
func (s *Service) ListChannels(ctx context.Context, user, project, cursor string, limit int, archived bool) (storage.Page, error) {
	if err := s.Auth.Check(ctx, user, project, "", "notifications.read"); err != nil {
		return storage.Page{}, err
	}
	if limit < 1 || limit > 100 {
		return storage.Page{}, errors.New("notification page limit must be between 1 and 100")
	}
	page, err := s.Store.List(ctx, Channels(project), project, cursor, limit)
	if err != nil {
		return page, err
	}
	allowed := []storage.Record{}
	for _, row := range page.Items {
		v, err := storage.Decode[Channel](row)
		if err != nil {
			return page, err
		}
		if v.Archived && !archived {
			continue
		}
		err = s.Auth.Check(ctx, user, project, "notification/"+v.ID, "notifications.read")
		if errors.Is(err, authorization.ErrForbidden) {
			continue
		}
		if err != nil {
			return page, err
		}
		allowed = append(allowed, row)
	}
	page.Items = allowed
	return page, nil
}
func normalize(d Definition) Definition {
	d.Name = strings.TrimSpace(d.Name)
	if d.MaxAttempts == 0 {
		d.MaxAttempts = 6
	}
	if d.Outcomes == nil {
		d.Outcomes = []string{"failed", "timed-out", "flaky", "unknown"}
	}
	return d
}
func validate(d Definition) error {
	if !domain.ValidID(d.ID) || d.Name == "" || len(d.Name) > 256 || !domain.ValidID(d.Reference) || (d.Kind != "webhook" && d.Kind != "discord") || d.MaxAttempts < 1 || d.MaxAttempts > 10 {
		return errors.New("channel requires valid identity, transport, secret reference, and 1 through 10 attempts")
	}
	if len(d.Outcomes) == 0 && !d.EvidenceGaps {
		return errors.New("select outcomes or evidence gaps for notification delivery")
	}
	for _, group := range []struct {
		values, allowed []string
		ids             bool
	}{{d.Suites, nil, true}, {d.Environments, nil, true}, {d.Stages, nil, false}, {d.Triggers, []string{"manual", "schedule", "rerun"}, false}, {d.Outcomes, []string{"passed", "failed", "timed-out", "flaky", "unknown"}, false}} {
		if len(group.values) > 100 {
			return errors.New("notification filter exceeds 100 values")
		}
		seen := map[string]bool{}
		for _, value := range group.values {
			if value == "" || len(value) > 128 || seen[value] || (group.ids && !domain.ValidID(value)) || (group.allowed != nil && !slices.Contains(group.allowed, value)) {
				return errors.New("notification filters require valid unique values")
			}
			seen[value] = true
		}
	}
	return nil
}
func (s *Service) References(ctx context.Context, user, project string) ([]secrets.Reference, error) {
	if err := s.Auth.Check(ctx, user, project, "", "notifications.manage"); err != nil {
		return nil, err
	}
	if s.Secrets == nil {
		return []secrets.Reference{}, nil
	}
	return s.Secrets.References(ctx, project, secrets.NotificationWebhook)
}
func (s *Service) SaveChannel(ctx context.Context, user, project string, d Definition, revision int64) error {
	if err := s.Auth.Check(ctx, user, project, "notification/"+d.ID, "notifications.manage"); err != nil {
		return err
	}
	d = normalize(d)
	if err := ValidateOrigin(s.Origin); err != nil {
		return err
	}
	if err := validate(d); err != nil {
		return err
	}
	if s.Secrets == nil {
		return errors.New("no notification secret provider is configured")
	}
	refs, err := s.Secrets.References(ctx, project, secrets.NotificationWebhook)
	if err != nil {
		return errors.New("notification secret references are unavailable")
	}
	if !slices.ContainsFunc(refs, func(ref secrets.Reference) bool { return ref.ID == d.Reference }) {
		return errors.New("notification secret reference is not registered for this project")
	}
	pRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return err
	}
	p, err := storage.Decode[projects.Project](pRow)
	if err != nil {
		return err
	}
	if p.Archived {
		return errors.New("project is archived")
	}
	old, row, err := s.channel(ctx, project, d.ID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if revision > 0 && errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if row.Revision != revision {
		return storage.ErrConflict
	}
	if old.Archived {
		return errors.New("restore the archived channel before editing")
	}
	v := Channel{Definition: d, Project: project, Owner: user}
	v.Epoch = max(1, old.Epoch)
	if revision > 0 && ((old.Enabled && !d.Enabled) || old.Kind != d.Kind || old.Reference != d.Reference) {
		v.Epoch++
	}
	c, err := storage.Put(Channels(project), d.ID, project, revision, v)
	if err != nil {
		return err
	}
	index, err := s.registryChange(ctx, project, d.ID, false)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "notification.channel.save", d.ID, old, v)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, c, index, audit, storage.Change{Record: pRow, Assert: true})
}

func ValidateOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || len(origin) > 512 {
		return errors.New("notification UI origin must be a valid HTTP or HTTPS origin within 512 bytes")
	}
	return nil
}
func (s *Service) registryChange(ctx context.Context, project, id string, archive bool) (storage.Change, error) {
	row, err := s.Store.Get(ctx, "notification-registry", project)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.Change{}, err
	}
	v := registry{IDs: []string{}}
	if err == nil {
		v, err = storage.Decode[registry](row)
		if err != nil {
			return storage.Change{}, err
		}
	}
	if archive {
		v.IDs = slices.DeleteFunc(v.IDs, func(value string) bool { return value == id })
	} else if !slices.Contains(v.IDs, id) {
		if len(v.IDs) >= 16 {
			return storage.Change{}, errors.New("project supports at most 16 unarchived notification channels")
		}
		v.IDs = append(v.IDs, id)
		slices.Sort(v.IDs)
	}
	return storage.Put("notification-registry", project, project, row.Revision, v)
}
func (s *Service) Archive(ctx context.Context, user, project, id string, revision int64, archived bool) error {
	if err := s.Auth.Check(ctx, user, project, "notification/"+id, "notifications.manage"); err != nil {
		return err
	}
	v, row, err := s.channel(ctx, project, id)
	if err != nil {
		return err
	}
	if row.Revision != revision {
		return storage.ErrConflict
	}
	old := v
	v.Archived = archived
	v.Enabled = false
	v.Owner = user
	v.Epoch++
	c, err := storage.Put(row.Bucket, row.ID, project, revision, v)
	if err != nil {
		return err
	}
	index, err := s.registryChange(ctx, project, id, archived)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "notification.channel.archive", id, old, v)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, c, index, audit)
}
