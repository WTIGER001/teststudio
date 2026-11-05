package sources

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/domain"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Bundle struct {
	Snapshot  string              `json:"repositorySnapshotId,omitempty"`
	Root      string              `json:"archiveRoot,omitempty"`
	Origin    *catalog.Provenance `json:"sourceOrigin,omitempty"`
	ID        string              `json:"id"`
	Project   string              `json:"projectId"`
	Suite     string              `json:"suiteId"`
	Name      string              `json:"name"`
	Manifest  string              `json:"manifestPath"`
	Digest    string              `json:"digest"`
	Content   []byte              `json:"content"`
	ExpiresAt time.Time           `json:"expiresAt"`
}

type bundleIndex struct {
	Job       string    `json:"jobId"`
	Suite     string    `json:"suiteId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func bundleBucket(project string) string      { return "source-bundles/" + project }
func buildQueueBucket(location string) string { return "preparation-build-queue/" + location }
func jobQueue(job Job) string {
	if job.Bundle != "" && !job.AssemblyComplete {
		return buildQueueBucket(job.Location)
	}
	return queueBucket(job.Location)
}
func activeJob(state string) bool { return state == "preparing" || state == "discovering" }

func (s *Service) EnqueueBundle(ctx context.Context, user, project, suite, key, name, manifest string, data []byte) (storage.Record, error) {
	return s.enqueueBundle(ctx, user, project, suite, key, name, manifest, data, nil, nil)
}

func (s *Service) enqueueBundle(ctx context.Context, user, project, suite, key, name, manifest string, data []byte, repository *repositoryPreparation, draft *storage.Record) (result storage.Record, err error) {
	if err := s.check(ctx, user, project, suite, "manage"); err != nil {
		return result, err
	}
	if !utf8.ValidString(name) || len(name) < 1 || len(name) > 200 || strings.TrimSpace(name) != name || strings.ContainsAny(name, "/\\") || strings.ContainsFunc(name, unicode.IsControl) {
		return result, errors.New("bundle requires a bounded file name")
	}
	if len(key) < 1 || len(key) > 128 || !domain.ValidID(s.Location) {
		return result, errors.New("bundle upload requires an idempotency key")
	}
	root := ""
	if repository != nil {
		root = repository.Root
	}
	snapshot, err := sourcebundle.ValidateRoot(data, manifest, suite, root)
	if err != nil {
		return result, err
	}
	enabled := false
	for _, runner := range s.BundleRunners {
		enabled = enabled || runner == snapshot.Manifest.Runner
	}
	if !enabled || !preparation.LocalSourceProfile(snapshot.Manifest.Runner, snapshot.Manifest.BuildProfile) {
		return result, errors.New("bundle preparation is not enabled for this runner")
	}
	id := bundleJobID(user, project, suite, key)
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
		if draft != nil {
			if err := bound.check(ctx, user, project, suite, "read"); err != nil {
				return err
			}
			current, err := bound.Store.Get(ctx, draft.Bucket, draft.ID)
			if err != nil {
				return err
			}
			if current.Revision != draft.Revision {
				return storage.ErrConflict
			}
			if err := bound.draftSuite(ctx, project, suite, snapshot.Manifest.Browser == nil); err != nil {
				return err
			}
		}
		if repository != nil {
			if err := bound.checkSyncLease(ctx, RepositorySync{Project: project, Source: repository.Source, ID: repository.Sync, Worker: repository.Worker}); err != nil {
				return err
			}
		}
		if row, err := bound.Store.Get(ctx, JobBucket(project, suite), id); err == nil {
			job, err := storage.Decode[Job](row)
			if err != nil {
				return err
			}
			if (repository == nil && job.SourceDigest != snapshot.Digest) || job.Manifest != manifest || job.SourceName != name || (draft != nil && (job.Draft != draft.ID || job.DraftRevision != draft.Revision)) || (repository != nil && (job.RepositorySource != repository.Source || job.SourceGeneration != repository.Generation || job.Origin == nil || job.Origin.Commit != repository.Origin.Commit)) {
				return errors.New("idempotency key reused for another source bundle")
			}
			if repository != nil && job.State != "published" && job.SourceSequence < repository.Sequence {
				job.SourceSequence = repository.Sequence
				if err := bound.checkRepositoryJob(ctx, job, false); err != nil {
					return err
				}
				changed, err := storage.Put(row.Bucket, row.ID, project, row.Revision, job)
				if err != nil {
					return err
				}
				if err := bound.Store.Commit(ctx, changed); err != nil {
					return err
				}
				row, err = bound.Store.Get(ctx, row.Bucket, row.ID)
				if err != nil {
					return err
				}
			}
			result = row
			return nil
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		now := time.Now().UTC()
		expires := now.Add(StagingLifetime)
		job := Job{Kind: "bundle", Bundle: id, SourceDigest: snapshot.Digest, SourceName: name, Manifest: manifest, ID: id, Project: project, Suite: suite, Actor: user, Runner: snapshot.Manifest.Runner, Location: s.Location, State: "received", Phase: "validating", CreatedAt: now}
		if draft != nil {
			job.Draft, job.DraftRevision = draft.ID, draft.Revision
		}
		bundle := Bundle{ID: id, Project: project, Suite: suite, Name: name, Manifest: manifest, Digest: snapshot.Digest, Content: data, ExpiresAt: expires}
		if repository != nil {
			job.Kind = "repository"
			job.RepositorySync = repository.Sync
			job.RepositorySource = repository.Source
			job.SourceGeneration = repository.Generation
			job.SourceSequence = repository.Sequence
			job.Origin = &repository.Origin
			if err := bound.checkRepositoryJob(ctx, job, false); err != nil {
				return err
			}
			bundle.Content = nil
			bundle.Snapshot = repository.Snapshot
			bundle.Root = repository.Root
			bundle.Origin = &repository.Origin
		}
		changes := []storage.Change{}
		if draft != nil {
			changes = append(changes, storage.Change{Record: *draft, Assert: true})
		}
		for _, item := range []struct {
			bucket, id string
			value      any
		}{{JobBucket(project, suite), id, job}, {bundleBucket(project), id, bundle}, {buildQueueBucket(s.Location), queueID(job), jobIndex{Job: id, Suite: suite, Location: s.Location}}, {"source-bundle-staging", id, bundleIndex{Job: id, Suite: suite, ExpiresAt: expires}}} {
			change, err := storage.Put(item.bucket, item.id, project, 0, item.value)
			if err != nil {
				return err
			}
			changes = append(changes, change)
		}
		if repository != nil {
			references, err := bound.referenceRepositorySnapshot(ctx, project, repository.Snapshot, 1, false, expires)
			if err != nil {
				return err
			}
			changes = append(changes, references...)
		}
		audit, err := domain.Audit(user, project, "sources.bundle.upload", suite+"/"+id, nil, map[string]any{"digest": snapshot.Digest, "bytes": len(data)})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, append(changes, audit)...); err != nil {
			return err
		}
		result, err = bound.Store.Get(ctx, JobBucket(project, suite), id)
		return err
	})
	if errors.Is(err, storage.ErrConflict) {
		if row, readErr := s.ReadJob(ctx, user, project, suite, id); readErr == nil {
			job, _ := storage.Decode[Job](row)
			if draft != nil {
				current, readErr := s.Draft(ctx, user, project, suite, draft.ID)
				if readErr != nil || current.Revision != draft.Revision || job.Draft != draft.ID || job.DraftRevision != draft.Revision {
					return result, err
				}
			}
			if job.Actor == user && job.SourceDigest == snapshot.Digest && job.Manifest == manifest && job.SourceName == name {
				return row, nil
			}
		}
	}
	return result, err
}

func (s *Service) ReadBundle(ctx context.Context, user, project, suite, id string) (Bundle, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return Bundle{}, err
	}
	row, err := s.Store.Get(ctx, JobBucket(project, suite), id)
	if err != nil {
		return Bundle{}, err
	}
	job, err := storage.Decode[Job](row)
	if err != nil {
		return Bundle{}, err
	}
	if job.RepositorySource != "" {
		if err := s.checkRepository(ctx, user, project, job.RepositorySource, "read"); err != nil {
			return Bundle{}, err
		}
	}
	if job.Bundle == "" {
		return Bundle{}, storage.ErrNotFound
	}
	return s.loadBundle(ctx, job)
}
func (s *Service) loadBundle(ctx context.Context, job Job) (Bundle, error) {
	row, err := s.Store.Get(ctx, bundleBucket(job.Project), job.Bundle)
	if err != nil {
		return Bundle{}, err
	}
	bundle, err := storage.Decode[Bundle](row)
	if err != nil {
		return bundle, err
	}
	if bundle.Snapshot != "" {
		snapshot, err := s.loadRepositorySnapshot(ctx, job.Project, bundle.Snapshot, job.RepositorySource)
		if err != nil {
			return Bundle{}, err
		}
		bundle.Content = snapshot.Content
	}
	if bundle.ID != job.Bundle || bundle.Project != job.Project || bundle.Suite != job.Suite || bundle.Digest != job.SourceDigest || digest(string(bundle.Content)) != bundle.Digest {
		return Bundle{}, errors.New("stored source bundle does not match its preparation")
	}
	if job.State != "published" && time.Now().After(bundle.ExpiresAt) {
		return Bundle{}, errors.New("source bundle staging has expired")
	}
	return bundle, nil
}

func bundleJobID(user, project, suite, key string) string {
	return digest("bundle\x00" + user + "\x00" + project + "\x00" + suite + "\x00" + key)[7:]
}
