package projects

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type SuiteSettings struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Applications   []string `json:"applications"`
	RequiresTarget bool     `json:"requiresTarget"`
}
type SuiteSchedule struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Release string `json:"releaseId,omitempty"`
}
type SuiteLifecycle struct {
	UpdatedAt time.Time       `json:"updatedAt"`
	Suite     Suite           `json:"suite"`
	Revision  int64           `json:"revision"`
	CanManage bool            `json:"canManage"`
	Schedules []SuiteSchedule `json:"schedules"`
}

func (s *Service) SuiteLifecycle(ctx context.Context, user, project, id string) (SuiteLifecycle, error) {
	result := SuiteLifecycle{Schedules: []SuiteSchedule{}}
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return result, errors.New("valid project and suite identifiers are required")
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+id, "suites.read"); err != nil {
		return result, err
	}
	row, err := s.Store.Get(ctx, Bucket("suites", project), id)
	if err != nil {
		return result, err
	}
	result.Suite, err = storage.Decode[Suite](row)
	if err != nil {
		return result, err
	}
	result.Revision = row.Revision
	result.UpdatedAt = row.Updated
	if err := s.Auth.Check(ctx, user, project, "suite/"+id, "suites.manage"); err == nil {
		result.CanManage = true
	} else if !errors.Is(err, authorization.ErrForbidden) {
		return result, err
	}
	rows, err := s.Store.All(ctx, "schedules/"+project, project)
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		schedule, err := storage.Decode[suiteScheduleRecord](row)
		if err != nil {
			return result, err
		}
		if schedule.Request.Suite != id || schedule.Archived {
			continue
		}
		if err := s.Auth.Check(ctx, user, project, "schedule/"+schedule.ID, "schedules.read"); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return result, err
		}
		result.Schedules = append(result.Schedules, SuiteSchedule{ID: schedule.ID, Name: schedule.Name, Enabled: schedule.Enabled, Release: schedule.Request.Release})
	}
	return result, nil
}
func (s *Service) UpdateSuiteSettings(ctx context.Context, user, project, id string, input SuiteSettings, revision int64) (SuiteLifecycle, error) {
	return s.withSuiteUpdate(ctx, user, project, id, func(bound *Service) error {
		row, err := bound.Store.Get(ctx, Bucket("suites", project), id)
		if err != nil {
			return err
		}
		current, err := storage.Decode[Suite](row)
		if err != nil {
			return err
		}
		if current.Archived {
			return errors.New("restore the suite before editing its settings")
		}
		current.Name, current.Description, current.Applications, current.RequiresTarget = input.Name, input.Description, input.Applications, input.RequiresTarget
		return bound.saveSuite(ctx, user, project, current, revision)
	})
}
func (s *Service) ChangeSuiteState(ctx context.Context, user, project, id, action string, revision int64) (SuiteLifecycle, error) {
	if action != "archive" && action != "restore" {
		return SuiteLifecycle{}, errors.New("select an archive or restore action")
	}
	return s.withSuiteUpdate(ctx, user, project, id, func(bound *Service) error {
		store := bound.Store
		row, err := store.Get(ctx, Bucket("suites", project), id)
		if err != nil {
			return err
		}
		before, err := storage.Decode[Suite](row)
		if err != nil {
			return err
		}
		after := before
		after.Archived = action == "archive"
		change, err := storage.Put(row.Bucket, row.ID, project, revision, after)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "suite."+action, id, before, after)
		if err != nil {
			return err
		}
		return store.Commit(ctx, change, audit)
	})
}
func (s *Service) withSuiteUpdate(ctx context.Context, user, project, id string, update func(*Service) error) (SuiteLifecycle, error) {
	result := SuiteLifecycle{}
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return result, errors.New("valid project and suite identifiers are required")
	}
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		for _, permission := range []string{"suites.read", "suites.manage"} {
			if err := auth.Check(ctx, user, project, "suite/"+id, permission); err != nil {
				return err
			}
		}
		if err := update(bound); err != nil {
			return err
		}
		var err error
		result, err = bound.SuiteLifecycle(ctx, user, project, id)
		return err
	})
	return result, err
}
func validSuiteSettings(value Suite) error {
	if !domain.ValidID(value.ID) || strings.TrimSpace(value.Name) == "" || len(value.Name) > 200 || len(value.Description) > 4096 || len(value.Applications) != 1 {
		return errors.New("suite requires a permanent identifier, bounded name and description, and exactly one application")
	}
	seen := map[string]bool{}
	for _, id := range value.Applications {
		if !domain.ValidID(id) || seen[id] {
			return errors.New("suite applications require distinct registered identifiers")
		}
		seen[id] = true
	}
	return nil
}

type suiteScheduleRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Archived bool   `json:"archived"`
	Request  struct {
		Suite   string `json:"suiteId"`
		Release string `json:"releaseId"`
	} `json:"request"`
}
