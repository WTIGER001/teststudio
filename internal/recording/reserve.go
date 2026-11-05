package recording

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) reserve(ctx context.Context, user, project, suite string, input Input, mode string) (storage.Record, error) {
	var row storage.Record
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.check(ctx, user, project, suite, true); err != nil {
			return err
		}
		if existing, err := bound.Store.Get(ctx, Bucket(), input.ID); err == nil {
			value, err := storage.Decode[Session](existing)
			if err != nil {
				return err
			}
			if value.Actor != user || value.Project != project || value.Suite != suite {
				return authorization.ErrForbidden
			}
			if value.Input != input || value.Mode != mode {
				return storage.ErrConflict
			}
			row = existing
			return nil
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		suiteRow, err := bound.Store.Get(ctx, projects.Bucket("suites", project), suite)
		if err != nil {
			return err
		}
		definition, err := storage.Decode[projects.Suite](suiteRow)
		if err != nil {
			return err
		}
		if definition.Archived {
			return errors.New("restore the suite before recording browser tests")
		}
		now := time.Now().UTC()
		value := Session{Mode: mode, Input: input, Project: project, Suite: suite, Actor: user, State: "starting", CreatedAt: now, DeadlineAt: now.Add(Lifetime)}
		if mode == "companion" {
			value.State = "pairing"
		}
		change, err := storage.Put(Bucket(), input.ID, project, 0, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.recording.start", suite+"/"+input.ID, nil, map[string]string{"testId": input.TestID})
		if err != nil {
			return err
		}
		changes := []storage.Change{change, audit}
		if mode == "companion" {
			connection, err := storage.Put(companionBucket, input.ID, project, 0, companionRecord{Binding: companionBinding(value)})
			if err != nil {
				return err
			}
			changes = append(changes, connection)
		}
		if err := bound.Store.Commit(ctx, changes...); err != nil {
			return err
		}
		row, err = bound.Store.Get(ctx, Bucket(), input.ID)
		return err
	})
	return row, err
}
