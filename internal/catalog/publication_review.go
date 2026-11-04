package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ReviewPublication(ctx context.Context, user, project, suite, draft string, revision int64) (PublicationReview, error) {
	result := PublicationReview{}
	err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
			return err
		}
		if err := bound.lockPublication(ctx, project, suite, draft); err != nil {
			return err
		}
		current, row, err := bound.suite(ctx, project, suite)
		if err != nil {
			return err
		}
		draftRow, err := bound.Store.Get(ctx, Bucket("drafts", project, suite), draft)
		if err != nil {
			return err
		}
		if draftRow.Revision != revision {
			return storage.ErrConflict
		}
		next, err := storage.Decode[Draft](draftRow)
		if err != nil {
			return err
		}
		if next.PublishedRelease != "" {
			return errors.New("the draft is already published")
		}
		if err := bound.validateSourceSnapshot(ctx, user, project, suite, next.Artifact); err != nil {
			return err
		}
		previous := Draft{}
		if current.ActiveRelease != "" {
			active, err := bound.Store.Get(ctx, Bucket("releases", project, suite), current.ActiveRelease)
			if err != nil {
				return err
			}
			release, err := storage.Decode[Release](active)
			if err != nil {
				return err
			}
			previous = release.Draft
		}
		ids := []string{}
		seen := map[string]bool{}
		for _, tests := range [][]Test{previous.Tests, next.Tests} {
			for _, test := range tests {
				if !seen[test.ID] {
					ids = append(ids, test.ID)
					seen[test.ID] = true
				}
			}
		}
		if err := bound.authorizePublicationTests(ctx, user, project, suite, ids); err != nil {
			return err
		}
		head, err := bound.head(ctx, project, suite)
		if err != nil {
			return err
		}
		schedules, digest, err := bound.publicationSchedules(ctx, user, project, suite)
		if err != nil {
			return err
		}
		changes, unchanged := publicationDiff(previous, next)
		issues := Validate(next)
		workingRows, err := bound.Tests(ctx, user, project, suite)
		if err != nil {
			return err
		}
		working := map[string]Test{}
		for _, row := range workingRows {
			test, err := storage.Decode[Test](row)
			if err != nil {
				return err
			}
			if test.State != "archived" {
				working[test.ID] = test
			}
		}
		same := len(working) == len(next.Tests)
		for _, test := range next.Tests {
			same = same && reflect.DeepEqual(test, working[test.ID])
		}
		if !same {
			issues = append(issues, Issue{Field: "catalog", Message: "The working catalog changed. Create a new draft and review it."})
		}
		projectRow, err := bound.Store.Get(ctx, "projects", project)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		result = PublicationReview{ID: domain.NewID(), Project: project, Suite: suite, Draft: draft, DraftRevision: revision, Active: current.ActiveRelease, Changes: changes, Unchanged: unchanged, Total: len(next.Tests), ArtifactChanged: !reflect.DeepEqual(previous.Artifact, next.Artifact), TimingChanged: !reflect.DeepEqual(previous.Timing, next.Timing), CaptureChanged: !reflect.DeepEqual(previous.Capture, next.Capture), Issues: issues, Schedules: schedules, CreatedAt: now, ExpiresAt: now.Add(30 * time.Minute)}
		result.AfterSettings = PublicationSettings{Artifact: next.Artifact, Timing: next.Timing, Capture: next.Capture}
		if current.ActiveRelease != "" {
			result.BeforeSettings = &PublicationSettings{Artifact: previous.Artifact, Timing: previous.Timing, Capture: previous.Capture}
		}
		for _, test := range next.Tests {
			if test.State == "enabled" {
				result.Enabled++
			}
		}
		review := publicationReviewRecord{Review: result, Actor: user, SuiteRevision: row.Revision, ProjectRevision: projectRow.Revision, Artifact: next.Artifact, HeadRevision: head.Revision, ScheduleDigest: digest, TestIDs: ids}
		change, err := storage.Put(Bucket("publication-reviews", project, suite), result.ID, project, 0, review)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "catalog.publication.review", suite+"/"+draft, nil, result.ID)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, audit)
	})
	return result, err
}

func (s *Service) authorizePublicationTests(ctx context.Context, user, project, suite string, ids []string) error {
	for _, id := range ids {
		if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+id, "catalog.read"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) publicationSchedules(ctx context.Context, user, project, suite string) ([]PublicationSchedule, string, error) {
	rows, err := s.Store.All(ctx, "schedules/"+project, project)
	if err != nil {
		return nil, "", err
	}
	visible := []PublicationSchedule{}
	fingerprints := []any{}
	for _, row := range rows {
		value, err := storage.Decode[struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Enabled    bool   `json:"enabled"`
			Archived   bool   `json:"archived"`
			Generation int64  `json:"generation"`
			Request    struct {
				Suite   string `json:"suiteId"`
				Release string `json:"releaseId"`
			} `json:"request"`
		}](row)
		if err != nil {
			return nil, "", err
		}
		if value.Request.Suite != suite || value.Archived {
			continue
		}
		fingerprints = append(fingerprints, value)
		if err := s.Auth.Check(ctx, user, project, "schedule/"+value.ID, "schedules.read"); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return nil, "", err
		}
		visible = append(visible, PublicationSchedule{ID: value.ID, Name: value.Name, Enabled: value.Enabled, Release: value.Request.Release, FollowsActive: value.Request.Release == ""})
	}
	data, err := json.Marshal(fingerprints)
	if err != nil {
		return nil, "", err
	}
	hash := sha256.Sum256(data)
	return visible, hex.EncodeToString(hash[:]), nil
}
