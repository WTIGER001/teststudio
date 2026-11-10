package catalog

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/authorization"
	"reflect"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/timing"
)

func (s *Service) CreateDraft(ctx context.Context, user, project, suite string, artifact Artifact) (Draft, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.update"); err != nil {
		return Draft{}, err
	}
	rows, err := s.Tests(ctx, user, project, suite)
	if err != nil {
		return Draft{}, err
	}
	policy, _, err := s.captureSettings(ctx, project, suite)
	if err != nil {
		return Draft{}, err
	}
	timingPolicy, err := s.timingSettings(ctx, project, suite)
	if err != nil {
		return Draft{}, err
	}
	d := Draft{Timing: timingPolicy, ID: domain.NewID(), Tests: []Test{}, Artifact: artifact, Capture: policy}
	for _, row := range rows {
		t, err := storage.Decode[Test](row)
		if err != nil {
			return Draft{}, err
		}
		if t.State != "archived" {
			d.Tests = append(d.Tests, t)
		}
	}
	return d, s.SaveDraft(ctx, user, project, suite, d, 0)
}

func (s *Service) SaveDraft(ctx context.Context, user, project, suite string, d Draft, revision int64) error {
	ctx, authorityErr := s.Auth.BeginMutation(ctx)
	if authorityErr != nil {
		return authorityErr
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.update"); err != nil {
		return err
	}
	owner, _, err := s.suite(ctx, project, suite)
	if err != nil {
		return err
	}
	if len(owner.Applications) != 1 {
		return errors.New("assign this suite to one application before saving tests")
	}
	for _, test := range d.Tests {
		if test.Application != "" && test.Application != owner.Applications[0] {
			return errors.New("test application is outside the suite")
		}
		if err := s.validateRequirements(ctx, project, owner.Applications[0], test); err != nil {
			return err
		}
	}
	if err := s.validateSourceSnapshot(ctx, user, project, suite, d.Artifact); err != nil {
		return err
	}
	if !domain.ValidID(d.ID) || len(d.Tests) > 1000 {
		return errors.New("invalid draft identifier or catalog size")
	}
	if d.PublishedRelease != "" {
		return errors.New("published draft is immutable")
	}
	_, limits, err := s.captureSettings(ctx, project, suite)
	if err != nil {
		return err
	}
	d.Timing = timing.Normalize(d.Timing)
	if err := timing.Validate(d.Timing); err != nil {
		return err
	}
	for _, t := range d.Tests {
		if t.Timing != nil {
			if err := timing.Validate(*t.Timing); err != nil {
				return err
			}
		}
	}
	d.Capture = capture.Normalize(d.Capture)
	if err := capture.Validate(d.Capture, limits); err != nil {
		return err
	}
	for _, t := range d.Tests {
		if t.Capture != nil {
			if err := capture.Validate(*t.Capture, limits); err != nil {
				return err
			}
		}
	}
	if revision > 0 {
		r, err := s.Store.Get(ctx, Bucket("drafts", project, suite), d.ID)
		if err != nil {
			return err
		}
		old, err := storage.Decode[Draft](r)
		if err != nil {
			return err
		}
		if old.PublishedRelease != "" {
			return errors.New("published draft is immutable")
		}
	}
	for _, t := range d.Tests {
		r, err := s.Store.Get(ctx, Bucket("tests", project, suite), t.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[Test](r)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(current, t) {
			return errors.New("save test edits before updating the catalog draft")
		}
	}
	c, err := storage.Put(Bucket("drafts", project, suite), d.ID, project, revision, d)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "catalog.draft.save", suite+"/"+d.ID, nil, d)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, append([]storage.Change{c, audit}, authorization.AuthorityAssertions(ctx)...)...)
}

func (s *Service) ValidateDraft(ctx context.Context, user, project, suite, id string) ([]Issue, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
		return nil, err
	}
	r, err := s.Store.Get(ctx, Bucket("drafts", project, suite), id)
	if err != nil {
		return nil, err
	}
	d, err := storage.Decode[Draft](r)
	if err != nil {
		return nil, err
	}
	return Validate(d), nil
}
