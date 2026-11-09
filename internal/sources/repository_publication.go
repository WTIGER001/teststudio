package sources

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/storage"
)

func publicationQueue(location string) string { return "repository-publication-queue/" + location }
func (s *Service) queueRepositoryPublication(ctx context.Context, job Job) ([]storage.Change, error) {
	if job.AutoSave && job.State == "ready" {
		if err := s.checkSavedTestJob(ctx, job); err != nil {
			return nil, err
		}
		change, err := storage.Put(savedPublicationQueue(job.Location), job.ID, job.Project, 0, jobIndex{Job: job.ID, Suite: job.Suite, Location: job.Location})
		return []storage.Change{change}, err
	}
	if job.RepositorySource == "" || job.State != "ready" {
		return nil, nil
	}
	row, err := s.Store.Get(ctx, repositoryBucket(job.Project), job.RepositorySource)
	if err != nil {
		return nil, err
	}
	source, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return nil, err
	}
	if !source.Configuration.AutoActivate || !source.Configuration.Enabled || source.Generation != job.SourceGeneration || source.Sequence != job.SourceSequence {
		return nil, nil
	}
	revision := int64(0)
	if existing, err := s.Store.Get(ctx, publicationQueue(job.Location), job.ID); err == nil {
		revision = existing.Revision
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}

	change, err := storage.Put(publicationQueue(job.Location), job.ID, job.Project, revision, jobIndex{Job: job.ID, Suite: job.Suite, Location: job.Location})
	return []storage.Change{change}, err
}
func (s *Service) ProcessRepositoryPublication(ctx context.Context) (bool, error) {
	return s.processPublication(ctx, publicationQueue(s.Location))
}
func savedPublicationQueue(location string) string { return "saved-test-publication-queue/" + location }
func (s *Service) ProcessSavedTestPublication(ctx context.Context) (bool, error) {
	return s.processPublication(ctx, savedPublicationQueue(s.Location))
}
func (s *Service) processPublication(ctx context.Context, bucket string) (bool, error) {
	page, err := s.Store.List(ctx, bucket, "", "", 25)
	if err != nil {
		return false, err
	}
	for _, index := range page.Items {
		item, err := storage.Decode[jobIndex](index)
		if err != nil {
			return false, err
		}
		row, err := s.Store.Get(ctx, JobBucket(index.Project, item.Suite), item.Job)
		if err != nil {
			return false, err
		}
		job, err := storage.Decode[Job](row)
		if err != nil {
			return false, err
		}
		if job.State == "ready" {
			upload, err := s.Store.Get(ctx, Bucket(job.Project, job.Suite), job.Upload)
			if err != nil {
				return false, err
			}
			_, publishErr := s.Publish(ctx, job.Actor, job.Project, job.Suite, job.Upload, upload.Revision, true)
			if publishErr != nil {
				latest, err := s.Store.Get(ctx, row.Bucket, row.ID)
				if err != nil {
					return false, err
				}
				current, err := storage.Decode[Job](latest)
				if err != nil {
					return false, err
				}
				if current.State == "ready" && current.Upload == job.Upload && current.SourceSequence == job.SourceSequence {
					current.Error = "Automatic source setup did not complete. Review source settings, access, and the catalog before retrying."
					updated, err := storage.Put(latest.Bucket, latest.ID, latest.Project, latest.Revision, current)
					if err != nil {
						return false, err
					}
					if err := s.Store.Commit(ctx, updated); err != nil && !errors.Is(err, storage.ErrConflict) {
						return false, err
					}
				}
			}
		}
		if err := s.Store.Commit(ctx, storage.Change{Record: index, Delete: true}); err != nil && !errors.Is(err, storage.ErrConflict) {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
