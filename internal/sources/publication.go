package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Publish(ctx context.Context, user, project, suite, id string, revision int64, activate bool) (release catalog.Release, resultErr error) {
	defer func() {
		if !errors.Is(resultErr, storage.ErrConflict) {
			return
		}
		if err := s.check(ctx, user, project, suite, "manage"); err != nil {
			resultErr = err
			return
		}
		if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			resultErr = err
			return
		}
		row, err := s.Store.Get(ctx, Bucket(project, suite), id)
		if err != nil {
			return
		}
		upload, err := storage.Decode[Upload](row)
		if err != nil || upload.State != "published" || upload.Actor != user || upload.Activated != activate {
			return
		}
		record, err := s.Store.Get(ctx, catalog.Bucket("releases", project, suite), upload.ReleaseID)
		if err != nil {
			return
		}
		release, err = storage.Decode[catalog.Release](record)
		if err == nil {
			resultErr = nil
		}
	}()
	if revision < 1 {
		return release, errors.New("a positive upload revision is required")
	}
	if err := s.check(ctx, user, project, suite, "manage"); err != nil {
		return release, err
	}
	resultErr = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.check(ctx, user, project, suite, "manage"); err != nil {
			return err
		}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
			return err
		}
		row, err := bound.Store.Get(ctx, Bucket(project, suite), id)
		if err != nil {
			return err
		}
		upload, err := storage.Decode[Upload](row)
		if err != nil {
			return err
		}
		if upload.Actor != user {
			return authorization.ErrForbidden
		}
		if upload.State == "published" {
			if upload.Activated != activate {
				return errors.New("upload publication cannot change its original activation choice")
			}
			record, err := bound.Store.Get(ctx, catalog.Bucket("releases", project, suite), upload.ReleaseID)
			if err != nil {
				return err
			}
			release, err = storage.Decode[catalog.Release](record)
			return err
		}
		if row.Revision != revision {
			return storage.ErrConflict
		}
		if upload.State != "staged" || !time.Now().Before(upload.ExpiresAt) {
			return errors.New("upload expired or contains validation errors")
		}
		if _, err := bound.Content(ctx, user, project, suite, id); err != nil {
			return err
		}
		catalogs := &catalog.Service{Store: bound.Store, Auth: auth}
		if _, err := catalogs.ApplySourceImport(ctx, user, project, suite, upload.PlanID, 1, upload.ID); err != nil {
			return err
		}
		artifact := catalog.Artifact{SourceRevision: upload.Digest, SourceSnapshot: id}
		if upload.Artifact != nil {
			artifact = *upload.Artifact
		}
		draft, err := catalogs.CreateDraft(ctx, user, project, suite, artifact)
		if err != nil {
			return err
		}
		release, err = catalogs.Publish(ctx, user, project, suite, draft.ID, "source-upload:"+id, 1, activate)
		if err != nil {
			return err
		}
		upload.State, upload.ReleaseID, upload.Activated = "published", release.ID, activate
		saved, err := storage.Put(row.Bucket, id, project, row.Revision, upload)
		if err != nil {
			return err
		}
		staging, err := bound.Store.Get(ctx, "source-staging", id)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.publish", suite+"/"+id, nil, map[string]any{"releaseId": release.ID, "digest": upload.Digest, "activate": activate})
		if err != nil {
			return err
		}
		changes := []storage.Change{saved, {Record: staging, Delete: true}, audit}
		if (upload.Kind == "prebuilt" || upload.Kind == "bundle") && upload.Artifact != nil {
			approved, err := storage.Put("prepared-artifacts/"+project, id, project, 0, *upload.Artifact)
			if err != nil {
				return err
			}
			changes = append(changes, approved)
			jobRow, err := bound.Store.Get(ctx, JobBucket(project, suite), upload.Job)
			if err != nil {
				return err
			}
			job, err := storage.Decode[Job](jobRow)
			if err != nil {
				return err
			}
			if job.State != "ready" || job.Upload != id {
				return errors.New("preparation is no longer ready for publication")
			}
			if job.Bundle != "" {
				if _, err := bound.loadBundle(ctx, job); err != nil {
					return err
				}
				staged, err := bound.Store.Get(ctx, "source-bundle-staging", job.Bundle)
				if err != nil {
					return err
				}
				changes = append(changes, storage.Change{Record: staged, Delete: true})
			}
			job.State, job.Phase, job.Release = "published", "finished", release.ID
			updated, err := storage.Put(jobRow.Bucket, jobRow.ID, project, jobRow.Revision, job)
			if err != nil {
				return err
			}
			changes = append(changes, updated)
		}
		return bound.Store.Commit(ctx, changes...)
	})
	return release, resultErr
}
