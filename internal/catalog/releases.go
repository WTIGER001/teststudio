package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

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
	if d.PublishedRelease != "" {
		return Release{}, errors.New("draft is already published")
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
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
		return err
	}
	if _, err := s.Store.Get(ctx, Bucket("releases", project, suite), releaseID); err != nil {
		return err
	}
	v, r, err := s.suite(ctx, project, suite)
	if err != nil {
		return err
	}
	v.ActiveRelease = releaseID
	c, err := storage.Put(r.Bucket, r.ID, project, revision, v)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "catalog.activate", suite, nil, releaseID)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, c, audit)
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
