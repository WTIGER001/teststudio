package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Publish(ctx context.Context, user, project, suite, id, key string, revision int64, activate bool) (Release, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
		return Release{}, err
	}
	if len(key) == 0 || len(key) > 128 {
		return Release{}, errors.New("an idempotency key of up to 128 characters is required")
	}
	digest := sha256.Sum256([]byte(user + "\x00" + project + "\x00" + suite + "\x00" + key))
	releaseID := hex.EncodeToString(digest[:])
	if existing, err := s.Store.Get(ctx, Bucket("releases", project, suite), releaseID); err == nil {
		v, err := storage.Decode[Release](existing)
		if err != nil {
			return Release{}, err
		}
		if v.Draft.ID != id || v.DraftRevision != revision || v.ActivatedOnPublication != activate {
			return Release{}, errors.New("idempotency key reused for another draft revision")
		}
		return v, nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return Release{}, err
	}
	v, suiteRecord, err := s.suite(ctx, project, suite)
	if err != nil {
		return Release{}, err
	}
	r, err := s.Store.Get(ctx, Bucket("drafts", project, suite), id)
	if err != nil {
		return Release{}, err
	}
	if r.Revision != revision {
		return Release{}, storage.ErrConflict
	}
	d, err := storage.Decode[Draft](r)
	if err != nil {
		return Release{}, err
	}
	if err := s.validateSourceSnapshot(ctx, user, project, suite, d.Artifact); err != nil {
		return Release{}, err
	}
	if d.PublishedRelease != "" {
		return Release{}, errors.New("draft is already published")
	}
	_, limits, err := s.captureSettings(ctx, project, suite)
	if err != nil {
		return Release{}, err
	}
	if err := capture.Validate(d.Capture, limits); err != nil {
		return Release{}, err
	}
	for _, test := range d.Tests {
		if test.Capture != nil {
			if err := capture.Validate(*test.Capture, limits); err != nil {
				return Release{}, err
			}
		}
	}
	issues := Validate(d)
	if len(issues) > 0 {
		return Release{}, fmt.Errorf("catalog validation failed: %s: %s", issues[0].Field, issues[0].Message)
	}
	release := Release{ActivatedOnPublication: activate, ID: releaseID, Suite: suite, Project: project, DraftRevision: revision, Draft: d, Actor: user, CreatedAt: time.Now().UTC()}
	c, err := storage.Put(Bucket("releases", project, suite), release.ID, project, 0, release)
	if err != nil {
		return Release{}, err
	}
	d.PublishedRelease = release.ID
	draft, err := storage.Put(r.Bucket, r.ID, project, r.Revision, d)
	if err != nil {
		return Release{}, err
	}
	audit, err := domain.Audit(user, project, "catalog.publish", suite+"/"+id, nil, release.ID)
	if err != nil {
		return Release{}, err
	}
	changes := []storage.Change{c, draft, audit}
	if activate {
		v.ActiveRelease = release.ID
		active, err := storage.Put(suiteRecord.Bucket, suite, project, suiteRecord.Revision, v)
		if err != nil {
			return Release{}, err
		}
		changes = append(changes, active)
	}
	err = s.Store.Commit(ctx, changes...)
	if errors.Is(err, storage.ErrConflict) {
		existing, readErr := s.Store.Get(ctx, Bucket("releases", project, suite), release.ID)
		if readErr == nil {
			v, decodeErr := storage.Decode[Release](existing)
			if decodeErr != nil {
				return Release{}, decodeErr
			}
			if v.Draft.ID != id || v.DraftRevision != revision || v.ActivatedOnPublication != activate {
				return Release{}, errors.New("idempotency key reused for another publication")
			}
			return v, nil
		}
	}
	return release, err
}

func (s *Service) Activate(ctx context.Context, user, project, suite, releaseID string, revision int64) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			return err
		}
		record, err := bound.Store.Get(ctx, Bucket("releases", project, suite), releaseID)
		if err != nil {
			return err
		}
		release, err := storage.Decode[Release](record)
		if err != nil {
			return err
		}
		if release.Project != project || release.Suite != suite || release.ID != releaseID {
			return storage.ErrNotFound
		}
		current, row, err := bound.suite(ctx, project, suite)
		if err != nil {
			return err
		}
		previous := current.ActiveRelease
		current.ActiveRelease = releaseID
		change, err := storage.Put(row.Bucket, row.ID, project, revision, current)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "catalog.activate", suite, previous, releaseID)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, audit)
	})
}

func (s *Service) ActiveRelease(ctx context.Context, project, suite string) (Release, error) {
	r, err := s.Store.Get(ctx, projects.Bucket("suites", project), suite)
	if err != nil {
		return Release{}, err
	}
	v, err := storage.Decode[projects.Suite](r)
	if err != nil {
		return Release{}, err
	}
	if v.Archived || v.ActiveRelease == "" {
		return Release{}, errors.New("suite has no executable active release")
	}
	record, err := s.Store.Get(ctx, Bucket("releases", project, suite), v.ActiveRelease)
	if err != nil {
		return Release{}, err
	}
	return storage.Decode[Release](record)
}
