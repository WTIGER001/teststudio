package schedules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store        *storage.Store
	Auth         *authorization.Service
	Runs         *runs.Service
	Now          func() time.Time
	PollInterval time.Duration
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
func Bucket(project string) string    { return "schedules/" + project }
func Fires(project, id string) string { return "schedule-fires/" + project + "/" + id }
func key(value string) string         { sum := sha256.Sum256([]byte(value)); return hex.EncodeToString(sum[:]) }
func (s *Service) get(ctx context.Context, project, id string) (Schedule, storage.Record, error) {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return Schedule{}, storage.Record{}, errors.New("valid project and schedule identifiers are required")
	}
	row, err := s.Store.Get(ctx, Bucket(project), id)
	if err != nil {
		return Schedule{}, row, err
	}
	v, err := storage.Decode[Schedule](row)
	if err == nil && (row.Project != project || v.Project != project) {
		err = storage.ErrNotFound
	}
	return v, row, err
}
func (s *Service) Read(ctx context.Context, user, project, id string) (Schedule, storage.Record, error) {
	if err := s.Auth.Check(ctx, user, project, "schedule/"+id, "schedules.read"); err != nil {
		return Schedule{}, storage.Record{}, err
	}
	v, row, err := s.get(ctx, project, id)
	row.Revision = v.Generation
	return v, row, err
}
func (s *Service) List(ctx context.Context, user, project, cursor string, limit int, archived bool) (storage.Page, error) {
	if err := s.Auth.Check(ctx, user, project, "", "schedules.read"); err != nil {
		return storage.Page{}, err
	}
	if limit < 1 || limit > 100 {
		return storage.Page{}, errors.New("schedule page limit must be between 1 and 100")
	}
	page, err := s.Store.List(ctx, Bucket(project), project, cursor, limit)
	if err != nil {
		return page, err
	}
	allowed := []storage.Record{}
	for _, row := range page.Items {
		v, err := storage.Decode[Schedule](row)
		if err != nil {
			return page, err
		}
		if v.Archived && !archived {
			continue
		}
		err = s.Auth.Check(ctx, user, project, "schedule/"+v.ID, "schedules.read")
		if errors.Is(err, authorization.ErrForbidden) {
			continue
		}
		if err != nil {
			return page, err
		}
		row.Revision = v.Generation
		allowed = append(allowed, row)
	}
	page.Items = allowed
	return page, nil
}
func (s *Service) History(ctx context.Context, user, project, id, cursor string, limit int) (storage.Page, error) {
	if _, _, err := s.Read(ctx, user, project, id); err != nil {
		return storage.Page{}, err
	}
	if limit < 1 || limit > 100 {
		return storage.Page{}, errors.New("firing page limit must be between 1 and 100")
	}
	return s.Store.ListNewest(ctx, Fires(project, id), project, cursor, limit)
}
func (s *Service) Preview(ctx context.Context, user, project string, d Definition) (Preview, error) {
	d = normalize(d, s.Runs.Location)
	permission, object := "schedules.create", ""
	if _, _, err := s.get(ctx, project, d.ID); err == nil {
		permission, object = "schedules.update", "schedule/"+d.ID
	} else if !errors.Is(err, storage.ErrNotFound) {
		return Preview{}, err
	}
	if err := s.Auth.Check(ctx, user, project, object, permission); err != nil {
		return Preview{}, err
	}
	if err := validate(d); err != nil {
		return Preview{}, err
	}
	preview := Preview{Next: []time.Time{}}
	at := s.now()
	for range 5 {
		next, err := Next(d, at)
		if err != nil {
			return preview, err
		}
		preview.Next = append(preview.Next, next)
		at = next
	}
	estimate, err := s.Runs.Estimate(ctx, user, project, d.Request)
	preview.Timing = estimate
	return preview, err
}
func (s *Service) Save(ctx context.Context, user, project string, d Definition, revision int64) error {
	return s.retryConfiguration(ctx, project, d.ID, revision, func() error { return s.save(ctx, user, project, d, revision) })
}
func (s *Service) retryConfiguration(ctx context.Context, project, id string, revision int64, change func() error) error {
	for range 8 {
		err := change()
		if !errors.Is(err, storage.ErrConflict) {
			return err
		}
		v, _, readErr := s.get(ctx, project, id)
		if readErr != nil || v.Generation != revision {
			return err
		}
	}
	return storage.ErrConflict
}
func (s *Service) save(ctx context.Context, user, project string, d Definition, revision int64) error {
	permission, object := "schedules.create", ""
	if revision > 0 {
		permission, object = "schedules.update", "schedule/"+d.ID
	}
	if err := s.Auth.Check(ctx, user, project, object, permission); err != nil {
		return err
	}
	d = normalize(d, s.Runs.Location)
	if err := validate(d); err != nil {
		return err
	}
	projectRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return err
	}
	p, err := storage.Decode[projects.Project](projectRow)
	if err != nil {
		return err
	}
	if p.Archived {
		return errors.New("project is archived")
	}
	if _, err := s.Store.Get(ctx, projects.Bucket("suites", project), d.Request.Suite); err != nil {
		return err
	}

	old, row, err := s.get(ctx, project, d.ID)
	if revision == 0 && err == nil {
		return storage.ErrConflict
	}
	if revision > 0 && err != nil {
		return err
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if old.Generation != revision {
		return storage.ErrConflict
	}
	if old.Archived {
		return errors.New("restore the archived schedule before editing")
	}
	if d.Enabled {
		if _, err := s.Runs.Estimate(ctx, user, project, d.Request); err != nil {
			return err
		}
	}
	v := old
	v.Definition = d
	v.Project = project
	v.Owner = user
	v.Generation++
	v.Claim = ""
	v.ClaimUntil = nil
	if !d.Enabled {
		v.NextAt = nil
	} else if old.NextAt == nil || old.Kind != d.Kind || old.IntervalSeconds != d.IntervalSeconds || old.Cron != d.Cron || old.Timezone != d.Timezone {
		next, err := Next(d, s.now())
		if err != nil {
			return err
		}
		v.NextAt = &next
	}
	c, err := storage.Put(Bucket(project), d.ID, project, row.Revision, v)
	if err != nil {
		return err
	}
	index, err := s.indexChange(ctx, v)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "schedule.save", d.ID, old, v)
	if err != nil {
		return err
	}
	changes := []storage.Change{c, audit, {Record: projectRow, Assert: true}}
	changes = append(changes, index...)
	return s.Store.Commit(ctx, changes...)
}
func (s *Service) indexChange(ctx context.Context, v Schedule) ([]storage.Change, error) {
	id := key(v.Project + "/" + v.ID)
	row, err := s.Store.Get(ctx, "schedule-work", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if !v.Enabled || v.Archived {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil
		}
		return []storage.Change{{Record: row, Delete: true}}, nil
	}
	c, err := storage.Put("schedule-work", id, v.Project, row.Revision, work{Project: v.Project, Schedule: v.ID})
	return []storage.Change{c}, err
}
func (s *Service) Archive(ctx context.Context, user, project, id string, revision int64, archived bool) error {
	return s.retryConfiguration(ctx, project, id, revision, func() error { return s.archive(ctx, user, project, id, revision, archived) })
}
func (s *Service) archive(ctx context.Context, user, project, id string, revision int64, archived bool) error {
	permission := "schedules.delete"
	if !archived {
		permission = "schedules.update"
	}
	if err := s.Auth.Check(ctx, user, project, "schedule/"+id, permission); err != nil {
		return err
	}
	v, row, err := s.get(ctx, project, id)
	if err != nil {
		return err
	}
	if v.Generation != revision {
		return storage.ErrConflict
	}
	old := v
	v.Archived = archived
	v.Enabled = false
	v.NextAt = nil
	v.Claim = ""
	v.ClaimUntil = nil
	v.Generation++
	c, err := storage.Put(row.Bucket, row.ID, project, row.Revision, v)
	if err != nil {
		return err
	}
	index, err := s.indexChange(ctx, v)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "schedule.archive", id, old, v)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, append([]storage.Change{c, audit}, index...)...)
}
