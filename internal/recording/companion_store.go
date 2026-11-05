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

func (s *Service) companionAuthority(ctx context.Context, value Session) error {
	if !s.CompanionEnabled || !time.Now().Before(value.DeadlineAt) {
		return authorization.ErrForbidden
	}
	if err := s.check(ctx, value.Actor, value.Project, value.Suite, true); err != nil {
		return authorization.ErrForbidden
	}
	row, err := s.Store.Get(ctx, projects.Bucket("suites", value.Project), value.Suite)
	if err != nil {
		return authorization.ErrForbidden
	}
	suite, err := storage.Decode[projects.Suite](row)
	if err != nil || suite.Archived {
		return authorization.ErrForbidden
	}
	row, err = s.Store.Get(ctx, "projects", value.Project)
	if err != nil {
		return authorization.ErrForbidden
	}
	project, err := storage.Decode[projects.Project](row)
	if err != nil || project.Archived {
		return authorization.ErrForbidden
	}
	return nil
}

func (s *Service) companionUpdate(ctx context.Context, id string, verify func(Session, companionRecord) error, change func(*Session, *companionRecord) error) (Session, companionRecord, storage.Record, error) {
	var value Session
	var connection companionRecord
	var result storage.Record
	if !domain.ValidID(id) || !s.CompanionEnabled {
		return value, connection, result, authorization.ErrForbidden
	}
	for range 8 {
		err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
			bound := &Service{Store: auth.Directory.Store, Auth: auth, CompanionEnabled: s.CompanionEnabled}
			row, err := bound.Store.Get(ctx, Bucket(), id)
			if err != nil {
				return authorization.ErrForbidden
			}
			value, err = storage.Decode[Session](row)
			if err != nil || value.Mode != "companion" || value.ID != id || value.Project != row.Project {
				return authorization.ErrForbidden
			}
			private, err := bound.Store.Get(ctx, companionBucket, id)
			if err != nil {
				return authorization.ErrForbidden
			}
			connection, err = storage.Decode[companionRecord](private)
			if err != nil || private.Project != value.Project || connection.Binding != companionBinding(value) {
				return authorization.ErrForbidden
			}
			if err := verify(value, connection); err != nil {
				return err
			}
			if err := bound.companionAuthority(ctx, value); err != nil {
				return err
			}
			original := companionHash(value)
			if err := change(&value, &connection); err != nil {
				return err
			}
			changes := []storage.Change{{Record: row, Assert: true}}
			if companionHash(value) != original {
				saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
				if err != nil {
					return err
				}
				changes = []storage.Change{saved}
			}
			saved, err := storage.Put(private.Bucket, private.ID, private.Project, private.Revision, connection)
			if err != nil {
				return err
			}
			if err := bound.Store.Commit(ctx, append(changes, saved)...); err != nil {
				return err
			}
			result, err = bound.Store.Get(ctx, row.Bucket, row.ID)
			return err
		})
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		return value, connection, result, err
	}
	return value, connection, result, storage.ErrConflict
}

func verifyCompanionDevice(instance, token string) func(Session, companionRecord) error {
	return func(_ Session, connection companionRecord) error {
		if !domain.ValidID(instance) || connection.Instance != instance || !matchesCompanionToken(token, connection.TokenHash) {
			return authorization.ErrForbidden
		}
		return nil
	}
}
