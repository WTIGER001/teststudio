package catalog

import (
	"context"
	"errors"
	"reflect"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type SavedTestRecord struct {
	storage.Record
	SavedVersion string `json:"savedVersionId,omitempty"`
}
type savedVersion struct {
	Revision int64  `json:"revision"`
	Version  string `json:"versionId"`
}

func savedVersionsBucket(project, suite string) string {
	return "catalog-saved-versions/" + project + "/" + suite
}
func (s *Service) ReadSavedTest(ctx context.Context, user, project, suite, id string) (SavedTestRecord, error) {
	if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+id, "catalog.read"); err != nil {
		return SavedTestRecord{}, err
	}
	row, err := s.Store.Get(ctx, Bucket("tests", project, suite), id)
	if err != nil {
		return SavedTestRecord{}, err
	}
	result := SavedTestRecord{Record: row}
	link, err := s.Store.Get(ctx, savedVersionsBucket(project, suite), id)
	if errors.Is(err, storage.ErrNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	version, err := storage.Decode[savedVersion](link)
	if err == nil && version.Revision == row.Revision {
		result.SavedVersion = version.Version
	}
	return result, err
}
func (s *Service) SaveCurrentTest(ctx context.Context, user, project, suite string, test Test, revision int64) error {
	_, err := s.SaveCurrentTestRecord(ctx, user, project, suite, test, revision)
	return err
}
func (s *Service) SaveCurrentTestRecord(ctx context.Context, user, project, suite string, test Test, revision int64) (result SavedTestRecord, resultErr error) {
	resultErr = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			return err
		}
		if test.State == "draft" {
			test.State = "enabled"
		}
		if revision > 0 {
			row, err := bound.Store.Get(ctx, Bucket("tests", project, suite), test.ID)
			if err != nil {
				return err
			}
			if row.Revision != revision {
				return storage.ErrConflict
			}
			before, err := storage.Decode[Test](row)
			if err != nil {
				return err
			}
			if before.State == "draft" && test.State == "enabled" {
				if err := bound.ChangeState(ctx, user, project, suite, test.ID, "enabled", revision); err != nil {
					return err
				}
				revision++
			}
		}
		if err := bound.SaveTest(ctx, user, project, suite, test, revision); err != nil {
			return err
		}
		if err := bound.useCurrent(ctx, user, project, suite); err != nil {
			return err
		}
		row, err := bound.Store.Get(ctx, Bucket("tests", project, suite), test.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[Test](row)
		if err != nil {
			return err
		}
		definition, _, err := bound.suite(ctx, project, suite)
		if err != nil {
			return err
		}
		version := ""
		if definition.ActiveRelease != "" {
			releaseRow, err := bound.Store.Get(ctx, Bucket("releases", project, suite), definition.ActiveRelease)
			if err != nil {
				return err
			}
			release, err := storage.Decode[Release](releaseRow)
			if err != nil {
				return err
			}
			for _, frozen := range release.Draft.Tests {
				if reflect.DeepEqual(frozen, current) {
					version = release.ID
					break
				}
			}
		}
		before, err := bound.Store.Get(ctx, savedVersionsBucket(project, suite), test.ID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		change, err := storage.Put(savedVersionsBucket(project, suite), test.ID, project, before.Revision, savedVersion{Revision: row.Revision, Version: version})
		if err != nil {
			return err
		}
		if err = bound.Store.Commit(ctx, change); err != nil {
			return err
		}
		result = SavedTestRecord{Record: row, SavedVersion: version}
		return nil
	})
	return result, resultErr
}
func (s *Service) useCurrent(ctx context.Context, user, project, suite string) error {
	definition, _, err := s.suite(ctx, project, suite)
	if err != nil {
		return err
	}
	artifact := Artifact{}
	var before Draft
	if definition.ActiveRelease != "" {
		row, err := s.Store.Get(ctx, Bucket("releases", project, suite), definition.ActiveRelease)
		if err != nil {
			return err
		}
		release, err := storage.Decode[Release](row)
		if err != nil {
			return err
		}
		before = release.Draft
		artifact = release.Draft.Artifact
	}
	rows, err := s.Tests(ctx, user, project, suite)
	if err != nil {
		return err
	}
	policy, _, err := s.captureSettings(ctx, project, suite)
	if err != nil {
		return err
	}
	timing, err := s.timingSettings(ctx, project, suite)
	if err != nil {
		return err
	}
	current := Draft{ID: domain.NewID(), Artifact: artifact, Timing: timing, Capture: policy, Tests: []Test{}}
	for _, row := range rows {
		test, e := storage.Decode[Test](row)
		if e != nil {
			return e
		}
		if test.State == "archived" {
			continue
		}
		if test.Runner != "http" && artifact.Discovered[test.ID] != test.Selector {
			continue
		}
		current.Tests = append(current.Tests, test)
	}
	if reflect.DeepEqual(before.Artifact, current.Artifact) && reflect.DeepEqual(before.Timing, current.Timing) && reflect.DeepEqual(before.Capture, current.Capture) && reflect.DeepEqual(before.Tests, current.Tests) {
		return nil
	}
	if len(Validate(current)) > 0 {
		return nil
	}
	if err := s.SaveDraft(ctx, user, project, suite, current, 0); err != nil {
		return err
	}
	_, err = s.Publish(ctx, user, project, suite, current.ID, "save-test:"+current.ID, 1, true)
	return err
}

func (s *Service) UseCurrent(ctx context.Context, user, project, suite string) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			return err
		}
		return (&Service{Store: auth.Directory.Store, Auth: auth}).useCurrent(ctx, user, project, suite)
	})
}

func (s *Service) SaveImport(ctx context.Context, user, project, suite, id string, revision int64) (result ImportPlan, err error) {
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			return err
		}
		var err error
		result, err = bound.ApplyImport(ctx, user, project, suite, id, revision)
		if err != nil {
			return err
		}
		return bound.useCurrent(ctx, user, project, suite)
	})
	return result, err
}
func (s *Service) SaveLifecycle(ctx context.Context, user, project, suite, id, state string, revision int64) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			return err
		}
		if err := bound.ChangeState(ctx, user, project, suite, id, state, revision); err != nil {
			return err
		}
		return bound.useCurrent(ctx, user, project, suite)
	})
}
