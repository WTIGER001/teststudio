package sources

import (
	"context"

	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func (s *Service) StartRepositorySynchronization(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker := domain.NewID()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			if err := s.RecoverRepositorySyncs(ctx); err != nil && ctx.Err() == nil {
				slog.Error("repository sync recovery failed")
			}
			if _, err := s.ProcessRepositorySync(ctx, worker); err != nil && ctx.Err() == nil {
				slog.Error("repository synchronization could not persist its outcome")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}
func (s *Service) checkSync(ctx context.Context, sync RepositorySync) error {
	if err := s.checkRepository(ctx, sync.Actor, sync.Project, sync.Source, "manage"); err != nil {
		return err
	}
	row, err := s.Store.Get(ctx, repositoryBucket(sync.Project), sync.Source)
	if err != nil {
		return err
	}
	source, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return err
	}
	if !source.Configuration.Enabled || source.Generation != sync.SourceGeneration {
		return errors.New("repository source is disabled or its configuration changed")
	}
	return s.validateRepositoryBindings(ctx, sync.Actor, sync.Project, sync.Configuration)
}
func (s *Service) ProcessRepositorySync(ctx context.Context, worker string) (bool, error) {
	if len(s.Providers) == 0 {
		return false, nil
	}
	page, err := s.Store.List(ctx, syncQueue(s.Location), "", "", 25)
	if err != nil {
		return false, err
	}
	for _, index := range page.Items {
		var claimed RepositorySync
		err := s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
			item, err := storage.Decode[syncIndex](index)
			if err != nil {
				return err
			}
			bound := &Service{Store: auth.Directory.Store, Auth: auth, BundleRunners: s.BundleRunners}
			row, err := bound.Store.Get(ctx, syncBucket(index.Project, item.Source), item.Sync)
			if err != nil {
				return err
			}
			sync, err := storage.Decode[RepositorySync](row)
			if err != nil {
				return err
			}
			if sync.State != "received" {
				return bound.Store.Commit(ctx, storage.Change{Record: index, Delete: true})
			}
			now := time.Now().UTC()
			sync.State, sync.Worker = "fetching", worker
			lease := now.Add(30 * time.Second)
			sync.LeaseUntil = &lease
			if err := bound.checkSync(ctx, sync); err != nil {
				sync.State, sync.Error = "failed", "Current source settings or access no longer permit synchronization."
				sync.FinishedAt = &now
				sync.LeaseUntil = nil
			}
			saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, sync)
			if err != nil {
				return err
			}
			changes := []storage.Change{saved, {Record: index, Delete: true}}
			if sync.State == "fetching" {
				lease, err := storage.Put(syncLeases(s.Location), sync.ID, sync.Project, 0, syncIndex{Source: sync.Source, Sync: sync.ID})
				if err != nil {
					return err
				}
				changes = append(changes, lease)
				claimed = sync
			}
			return bound.Store.Commit(ctx, changes...)
		})
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		if err != nil {
			return false, err
		}
		if claimed.ID == "" {
			continue
		}
		work, cancel := context.WithTimeout(ctx, 3*time.Minute)
		heartbeat := make(chan struct{})
		go func() { defer close(heartbeat); s.syncHeartbeat(work, cancel, claimed, worker) }()
		prepared, fetchErr := s.fetchRepository(work, claimed)
		cancel()
		<-heartbeat
		finish, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		err = s.finishRepositorySync(finish, claimed, worker, prepared, fetchErr)
		stop()
		return true, err
	}
	return false, nil
}
func preparationKey(sync RepositorySync, binding RepositoryBinding) string {
	key := fmt.Sprintf("repository/%s/%d/%s/%s/%s/%s", sync.Source, sync.SourceGeneration, sync.Revision.Commit, binding.Manifest, binding.Runner, binding.Profile)
	if sync.Force {
		key += "/retry/" + sync.ID
	}
	return digest(key)[7:]
}
func (s *Service) fetchRepository(ctx context.Context, sync RepositorySync) (map[string]string, error) {
	jobs := map[string]string{}
	provider := s.Providers[sync.Configuration.Provider]
	if provider == nil {
		return jobs, sourceprovider.ErrUnavailable
	}
	access := sourceprovider.Access{Project: sync.Project, Repository: sync.Configuration.Repository, RepositoryID: sync.RepositoryID, Credential: sync.Configuration.Credential}
	revision, err := provider.Resolve(ctx, access, sync.Configuration.Reference)
	if err != nil {
		return jobs, err
	}
	if !sourceprovider.ValidCommit(revision.Commit) || revision.RepositoryID == "" || revision.Reference != sync.Configuration.Reference {
		return jobs, errors.New("source provider returned an invalid resolved revision")
	}
	if !strings.EqualFold(revision.Repository, sync.Configuration.Repository) {
		return jobs, errors.New("resolved repository identity does not match its registration")
	}
	if err := suitespec.ValidateOrigin(catalog.Provenance{Repository: revision.URL, Reference: revision.Reference, Commit: revision.Commit}); err != nil {
		return jobs, err
	}
	sync.Revision = &revision
	if err := s.pinRepositoryRevision(ctx, sync); err != nil {
		return jobs, err
	}
	missing := false
	for _, binding := range sync.Configuration.Bindings {
		id := bundleJobID(sync.Actor, sync.Project, binding.Suite, preparationKey(sync, binding))
		row, err := s.Store.Get(ctx, JobBucket(sync.Project, binding.Suite), id)
		if errors.Is(err, storage.ErrNotFound) {
			missing = true
			continue
		}
		if err != nil {
			return jobs, err
		}
		job, err := storage.Decode[Job](row)
		if err != nil {
			return jobs, err
		}
		if job.RepositorySource != sync.Source || job.SourceGeneration != sync.SourceGeneration || job.Origin == nil || job.Origin.Commit != revision.Commit {
			return jobs, errors.New("existing preparation does not match the resolved source")
		}
		if err := s.adoptRepositoryPreparation(ctx, sync, job); err != nil {
			return jobs, err
		}
		jobs[binding.Suite] = id
	}
	if !missing {
		return jobs, nil
	}
	archive, err := provider.Archive(ctx, access, revision)
	if err != nil {
		return jobs, err
	}
	for _, binding := range sync.Configuration.Bindings {
		snapshot, err := sourcebundle.ValidateRoot(archive.Content, binding.Manifest, binding.Suite, archive.Root)
		if err != nil {
			return jobs, err
		}
		if snapshot.Manifest.Runner != binding.Runner || snapshot.Manifest.BuildProfile != binding.Profile {
			return jobs, errors.New("repository manifest does not match its registered build profile")
		}
	}
	sourceDigest := digest(string(archive.Content))
	origin := catalog.Provenance{Repository: revision.URL, Reference: revision.Reference, Commit: revision.Commit, SnapshotDigest: sourceDigest}
	if err := s.storeRepositorySnapshot(ctx, sync, archive, sourceDigest); err != nil {
		return jobs, err
	}
	for _, binding := range sync.Configuration.Bindings {
		if _, ok := jobs[binding.Suite]; ok {
			continue
		}
		_, name, _ := strings.Cut(revision.Repository, "/")
		row, err := s.enqueueBundle(ctx, sync.Actor, sync.Project, binding.Suite, preparationKey(sync, binding), name+"-"+revision.Commit[:12]+".zip", binding.Manifest, archive.Content, &repositoryPreparation{Sync: sync.ID, Worker: sync.Worker, Source: sync.Source, Generation: sync.SourceGeneration, Sequence: sync.Sequence, Snapshot: sync.ID, Origin: origin, Root: archive.Root})
		if err != nil {
			return jobs, err
		}
		jobs[binding.Suite] = row.ID
	}
	return jobs, nil
}
func (s *Service) storeRepositorySnapshot(ctx context.Context, sync RepositorySync, archive sourceprovider.Archive, digest string) error {
	return s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		store := auth.Directory.Store
		if err := (&Service{Store: store, Auth: auth, BundleRunners: s.BundleRunners}).checkSync(ctx, sync); err != nil {
			return err
		}
		if err := (&Service{Store: store}).checkSyncLease(ctx, sync); err != nil {
			return err
		}
		snapshot := RepositorySnapshot{ID: sync.ID, Project: sync.Project, Source: sync.Source, Origin: *sync.Revision, Root: archive.Root, Digest: digest, ExpiresAt: time.Now().UTC().Add(StagingLifetime)}
		meta, err := storage.Put(snapshotBucket(sync.Project), sync.ID, sync.Project, 0, snapshot)
		if err != nil {
			return err
		}
		content, err := storage.Put(repositoryContentBucket(sync.Project), sync.ID, sync.Project, 0, repositoryContent{Content: archive.Content})
		if err != nil {
			return err
		}
		index, err := storage.Put("repository-snapshot-staging", sync.ID, sync.Project, 0, map[string]string{"sourceId": sync.Source})
		if err != nil {
			return err
		}
		syncRow, err := store.Get(ctx, syncBucket(sync.Project, sync.Source), sync.ID)
		if err != nil {
			return err
		}
		current, err := storage.Decode[RepositorySync](syncRow)
		if err != nil {
			return err
		}
		current.Snapshot = sync.ID
		updated, err := storage.Put(syncRow.Bucket, syncRow.ID, sync.Project, syncRow.Revision, current)
		if err != nil {
			return err
		}
		return store.Commit(ctx, meta, content, index, updated)
	})
}
