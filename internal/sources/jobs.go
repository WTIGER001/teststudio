package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

var immutableImage = regexp.MustCompile(`^(sha256:[a-f0-9]{64}|[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64})$`)

func (s *Service) EnqueuePrebuilt(ctx context.Context, user, project, suite, key string, input PrebuiltInput) (result storage.Record, err error) {
	if err := s.check(ctx, user, project, suite, "manage"); err != nil {
		return result, err
	}
	if s.Inspectors[input.Runner] == nil || !domain.ValidID(s.Location) {
		return result, errors.New("prebuilt preparation is not enabled for this runner and location")
	}
	if !immutableImage.MatchString(input.Image) || len(key) < 1 || len(key) > 128 {
		return result, errors.New("an immutable image and idempotency key are required")
	}
	hash := sha256.Sum256([]byte(user + "\x00" + project + "\x00" + suite + "\x00" + key))
	id := hex.EncodeToString(hash[:])
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.check(ctx, user, project, suite, "manage"); err != nil {
			return err
		}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
			return err
		}
		if _, err := (&catalog.Service{Store: bound.Store, Auth: auth}).Tests(ctx, user, project, suite); err != nil {
			return err
		}
		existing, err := bound.Store.Get(ctx, JobBucket(project, suite), id)
		if err == nil {
			job, err := storage.Decode[Job](existing)
			if err != nil {
				return err
			}
			if job.Image != input.Image || job.Runner != input.Runner {
				return errors.New("idempotency key reused for another preparation")
			}
			result = existing
			return nil
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		job := Job{ID: id, Project: project, Suite: suite, Actor: user, Image: input.Image, Runner: input.Runner, Location: s.Location, State: "received", Phase: "validating", CreatedAt: time.Now().UTC()}
		row, err := storage.Put(JobBucket(project, suite), id, project, 0, job)
		if err != nil {
			return err
		}
		queued, err := storage.Put(queueBucket(s.Location), queueID(job), project, 0, jobIndex{Job: id, Suite: suite, Location: s.Location})
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.prepare", suite+"/"+id, nil, input)
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, row, queued, audit); err != nil {
			return err
		}
		result, err = bound.Store.Get(ctx, row.Bucket, row.ID)
		return err
	})
	if errors.Is(err, storage.ErrConflict) {
		if row, readErr := s.ReadJob(ctx, user, project, suite, id); readErr == nil {
			job, _ := storage.Decode[Job](row)
			if job.Actor == user && job.Image == input.Image && job.Runner == input.Runner {
				return row, nil
			}
		}
	}
	return result, err
}

func (s *Service) ReadJob(ctx context.Context, user, project, suite, id string) (storage.Record, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, JobBucket(project, suite), id)
}
func (s *Service) ListJobs(ctx context.Context, user, project, suite, cursor string, limit int) (storage.Page, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return storage.Page{}, err
	}
	return s.Store.ListNewest(ctx, JobBucket(project, suite), project, cursor, limit)
}

func (s *Service) CancelJob(ctx context.Context, user, project, suite, id string, revision int64) error {
	if revision < 1 {
		return errors.New("a positive preparation revision is required")
	}
	if err := s.check(ctx, user, project, suite, "manage"); err != nil {
		return err
	}
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.check(ctx, user, project, suite, "manage"); err != nil {
			return err
		}
		row, err := bound.Store.Get(ctx, JobBucket(project, suite), id)
		if err != nil {
			return err
		}
		job, err := storage.Decode[Job](row)
		if err != nil {
			return err
		}
		if job.State == "cancelled" {
			return nil
		}
		if row.Revision != revision {
			return storage.ErrConflict
		}
		if job.State == "published" {
			return errors.New("published preparation cannot be cancelled")
		}
		if job.State == "failed" {
			return errors.New("preparation is already finished")
		}
		changes := []storage.Change{}
		if job.State == "ready" {
			uploadRow, err := bound.Store.Get(ctx, Bucket(project, suite), job.Upload)
			if err != nil {
				return err
			}
			upload, err := storage.Decode[Upload](uploadRow)
			if err != nil {
				return err
			}
			if upload.State == "published" {
				return errors.New("published preparation cannot be cancelled")
			}
			upload.State = "cancelled"
			saved, err := storage.Put(uploadRow.Bucket, uploadRow.ID, project, uploadRow.Revision, upload)
			if err != nil {
				return err
			}
			changes = append(changes, saved)
		}
		job.CancelRequested = true
		if job.State != "discovering" {
			job.State, job.Phase = "cancelled", "finished"
			now := time.Now().UTC()
			job.FinishedAt = &now
		}
		if queued, err := bound.Store.Get(ctx, queueBucket(job.Location), queueID(job)); err == nil {
			changes = append(changes, storage.Change{Record: queued, Delete: true})
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		saved, err := storage.Put(row.Bucket, row.ID, project, row.Revision, job)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.prepare.cancel", suite+"/"+id, nil, job.State)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, append(changes, saved, audit)...)
	})
}

type PreparationOptions struct {
	Runners  []string `json:"runners"`
	Location string   `json:"executionLocation"`
}

func (s *Service) Options(ctx context.Context, user, project, suite string) (PreparationOptions, error) {
	options := PreparationOptions{Runners: []string{}, Location: s.Location}
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return options, err
	}
	for _, runner := range []string{"go", "playwright"} {
		if s.Inspectors[runner] != nil {
			options.Runners = append(options.Runners, runner)
		}
	}
	return options, nil
}
