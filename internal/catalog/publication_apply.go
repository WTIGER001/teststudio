package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) lockPublication(ctx context.Context, project, suite, draft string) error {
	for _, target := range [][2]string{{Bucket("drafts", project, suite), draft}, {projects.Bucket("suites", project), suite}, {"projects", project}, {"catalog-heads/" + project, suite}} {
		row, err := s.Store.Get(ctx, target[0], target[1])
		if errors.Is(err, storage.ErrNotFound) && target[0] == "catalog-heads/"+project {
			continue
		}
		if err != nil {
			return err
		}
		if err := s.Store.Commit(ctx, storage.Change{Record: row, Assert: true}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) readPublicationReview(ctx context.Context, user, project, suite, id string) (publicationReviewRecord, storage.Record, error) {
	value := publicationReviewRecord{}
	if !domain.ValidID(id) {
		return value, storage.Record{}, storage.ErrNotFound
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
		return value, storage.Record{}, err
	}
	row, err := s.Store.Get(ctx, Bucket("publication-reviews", project, suite), id)
	if err != nil {
		return value, row, err
	}
	value, err = storage.Decode[publicationReviewRecord](row)
	if err != nil {
		return value, row, err
	}
	if value.Actor != user || value.Review.Project != project || value.Review.Suite != suite || value.Review.ID != id {
		return publicationReviewRecord{}, row, storage.ErrNotFound
	}
	if err := s.authorizePublicationTests(ctx, user, project, suite, value.TestIDs); err != nil {
		return publicationReviewRecord{}, row, err
	}
	if err := s.validateSourceSnapshot(ctx, user, project, suite, value.Artifact); err != nil {
		return publicationReviewRecord{}, row, err
	}
	return value, row, nil
}

func (s *Service) ReadPublicationReview(ctx context.Context, user, project, suite, id string) (PublicationReview, error) {
	value, _, err := s.readPublicationReview(ctx, user, project, suite, id)
	if err != nil {
		return PublicationReview{}, err
	}
	if !time.Now().Before(value.Review.ExpiresAt) {
		return PublicationReview{}, storage.ErrConflict
	}
	schedules, _, err := s.publicationSchedules(ctx, user, project, suite)
	if err != nil {
		return PublicationReview{}, err
	}
	value.Review.Schedules = schedules
	return value.Review, nil
}

func (s *Service) PublishReviewed(ctx context.Context, user, project, suite, draft, id, key string, revision int64, activate bool) (Release, error) {
	result := Release{}
	apply := func(auth *authorization.Service) error {
		result = Release{}
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			return err
		}
		value, reviewRow, err := bound.readPublicationReview(ctx, user, project, suite, id)
		if err != nil {
			return err
		}
		if value.Review.Draft != draft || value.Review.DraftRevision != revision {
			return storage.ErrConflict
		}
		if value.Release != "" {
			if value.Key != key || value.Activate != activate {
				return storage.ErrConflict
			}
			row, err := bound.Store.Get(ctx, Bucket("releases", project, suite), value.Release)
			if err != nil {
				return err
			}
			result, err = storage.Decode[Release](row)
			return err
		}
		if !time.Now().Before(value.Review.ExpiresAt) {
			return storage.ErrConflict
		}
		if len(value.Review.Issues) > 0 {
			return errors.New("correct publication review issues before publishing")
		}
		if err := bound.lockPublication(ctx, project, suite, draft); err != nil {
			return err
		}
		current, suiteRow, err := bound.suite(ctx, project, suite)
		if err != nil {
			return err
		}
		draftRow, err := bound.Store.Get(ctx, Bucket("drafts", project, suite), draft)
		if err != nil {
			return err
		}
		head, err := bound.head(ctx, project, suite)
		if err != nil {
			return err
		}
		projectRow, err := bound.Store.Get(ctx, "projects", project)
		if err != nil {
			return err
		}
		_, digest, err := bound.publicationSchedules(ctx, user, project, suite)
		if err != nil {
			return err
		}
		if current.ActiveRelease != value.Review.Active || suiteRow.Revision != value.SuiteRevision || head.Revision != value.HeadRevision || draftRow.Revision != revision || projectRow.Revision != value.ProjectRevision || digest != value.ScheduleDigest {
			return storage.ErrConflict
		}
		result, err = bound.Publish(ctx, user, project, suite, draft, key, revision, activate)
		if err != nil {
			return err
		}
		value.Release, value.Key, value.Activate = result.ID, key, activate
		change, err := storage.Put(reviewRow.Bucket, reviewRow.ID, project, reviewRow.Revision, value)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change)
	}
	err := s.Auth.WithAssignmentTransaction(ctx, apply)
	if errors.Is(err, storage.ErrConflict) {
		row, readErr := s.Store.Get(ctx, Bucket("publication-reviews", project, suite), id)
		if readErr == nil {
			value, decodeErr := storage.Decode[publicationReviewRecord](row)
			if decodeErr == nil && value.Actor == user && value.Release != "" && value.Key == key && value.Activate == activate {
				err = s.Auth.WithAssignmentTransaction(ctx, apply)
			}
		}
	}
	return result, err
}
