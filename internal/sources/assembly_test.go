package sources_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

type assemblyBuilder struct {
	*hostedBuilder
	completed      int
	references     []execution.Reference
	beforeComplete func()
}

func (b *assemblyBuilder) CompleteAssembly(context.Context, execution.Reference) error {
	if b.beforeComplete != nil {
		b.beforeComplete()
	}
	b.completed++
	return nil
}
func (b *assemblyBuilder) Build(ctx context.Context, input preparation.BuildRequest) (preparation.Build, error) {
	b.references = append(b.references, input.Reference)
	return b.hostedBuilder.Build(ctx, input)
}
func assemblySetup(t *testing.T) (*sources.Service, *sources.Service, *assemblyBuilder, *discoveryProvider) {
	t.Helper()
	api, provider := hostedPreparation(t, "")
	builder := &assemblyBuilder{hostedBuilder: provider.builder}
	writer := *api
	writer.AssemblyOnly = true
	writer.Builders = map[string]preparation.ArtifactBuilder{"go": builder}
	writer.RecoveryBuilders = writer.Builders
	writer.Inspectors, writer.RecoveryInspectors = nil, nil
	api.Builders, api.RecoveryBuilders = nil, nil
	return api, &writer, builder, provider
}
func storedAssemblyJob(t *testing.T, s *sources.Service, id string) (storage.Record, sources.Job) {
	t.Helper()
	row, err := s.Store.Get(t.Context(), sources.JobBucket("alpha", "health"), id)
	if err != nil {
		t.Fatal(err)
	}
	job, err := storage.Decode[sources.Job](row)
	if err != nil {
		t.Fatal(err)
	}
	return row, job
}
func expireAssemblyLease(t *testing.T, s *sources.Service, id string) {
	t.Helper()
	row, job := storedAssemblyJob(t, s, id)
	past := time.Now().UTC().Add(-time.Second)
	job.LeaseUntil = &past
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Store.CommitAll(t.Context(), change); err != nil {
		t.Fatal(err)
	}
}
func assertPendingAssembly(t *testing.T, s *sources.Service, id string) sources.Job {
	t.Helper()
	_, job := storedAssemblyJob(t, s, id)
	if !job.AssemblyComplete || job.AssemblyReference == nil || job.Reference != nil || job.State != "received" || job.Phase != "awaiting-discovery" || job.Worker != "" || job.LeaseUntil != nil || job.Upload != "" || job.Build == nil || job.Build.Image != job.Image {
		t.Fatalf("invalid discovery handoff: %+v", job)
	}
	if _, err := s.Store.Get(t.Context(), "preparation-leases/"+s.Location, id); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("assembly lease survived its atomic handoff", err)
	}
	return job
}

func TestAssemblyOnlyQueuesDiscoveryWithoutApprovalAndUsesAnIndependentOwner(t *testing.T) {
	api, writer, builder, provider := assemblySetup(t)
	queued, _ := enqueueZIP(t, api, "assembly-handoff")
	if processed, err := writer.ProcessBuildNext(t.Context(), "registry-writer"); !processed || err != nil {
		t.Fatal(processed, err)
	}
	waiting := assertPendingAssembly(t, api, queued.ID)
	if provider.created != 0 || provider.discovered != 0 || builder.calls != 1 || builder.completed != 1 || *waiting.AssemblyReference != builder.references[0] {
		t.Fatal("writer dispatched discovery or lost its assembly owner")
	}
	if _, err := api.AuthorizeDiscovery(t.Context(), *waiting.AssemblyReference); err == nil {
		t.Fatal("assembly reference authorized discovery")
	}
	if err := api.ApproveArtifact(t.Context(), catalog.Artifact{SourceProject: "alpha", SourceSnapshot: queued.ID}); err == nil {
		t.Fatal("uninspected assembly became an approved artifact")
	}
	if processed, err := writer.ProcessNext(t.Context(), "registry-writer"); processed || err != nil {
		t.Fatal("assembly process consumed discovery", processed, err)
	}
	if processed, err := writer.ProcessBuildNext(t.Context(), "registry-writer"); processed || err != nil || builder.calls != 1 {
		t.Fatal("completed assembly was rebuilt", processed, err)
	}
	if processed, err := api.ProcessNext(t.Context(), "api-discovery"); !processed || err != nil {
		t.Fatal("API without writer credentials could not discover the handed image", processed, err)
	}
	_, ready := storedAssemblyJob(t, api, queued.ID)
	if ready.State != "ready" || ready.Reference == nil || ready.Reference.ID == waiting.AssemblyReference.ID || ready.Reference.Worker != "api-discovery" || ready.Image != waiting.Image || ready.SourceDigest != waiting.SourceDigest || builder.calls != 1 || provider.created != 1 || provider.discovered != 1 {
		t.Fatal("discovery lost its independent claim or rebuilt source", ready.State, ready.Error)
	}
	published, err := api.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, 1, true)
	if err != nil || api.ApproveArtifact(t.Context(), published.Draft.Artifact) != nil {
		t.Fatal("normal discovery review and approval did not complete", err)
	}
	expireAssemblyLease(t, api, queued.ID)
	if err := api.RecoverPreparations(t.Context(), "api-cleanup"); err != nil {
		t.Fatal(err)
	}
	if provider.removed != 1 || builder.cleanups != 0 || builder.completed != 1 {
		t.Fatal("discovery cleanup required writer authority or used the assembly owner")
	}
}

func TestAssemblyOnlyRecoversConfirmedCheckpointWithoutRebuildOrDiscovery(t *testing.T) {
	api, writer, builder, provider := assemblySetup(t)
	queued, _ := enqueueZIP(t, api, "assembly-checkpoint")
	crash := errors.New("owned assembly interruption")
	builder.beforeComplete = func() { panic(crash) }
	func() {
		defer func() {
			if recovered := recover(); recovered != crash {
				t.Fatal("expected checkpoint interruption", recovered)
			}
		}()
		writer.ProcessBuildNext(t.Context(), "interrupted-writer")
	}()
	_, checkpoint := storedAssemblyJob(t, api, queued.ID)
	if checkpoint.AssemblyComplete || checkpoint.Build == nil || checkpoint.State != "discovering" {
		t.Fatal("interruption did not preserve the image checkpoint")
	}
	builder.beforeComplete = nil
	expireAssemblyLease(t, api, queued.ID)
	restarted := *writer
	restarted.Builders = nil
	restarted.RecoveryBuilders = map[string]preparation.ArtifactBuilder{"go": builder}
	if err := restarted.RecoverPreparations(t.Context(), "restarted-writer"); err != nil {
		t.Fatal(err)
	}
	assertPendingAssembly(t, api, queued.ID)
	if err := restarted.RecoverPreparations(t.Context(), "restarted-writer-again"); err != nil {
		t.Fatal(err)
	}
	if builder.calls != 1 || builder.completed != 1 || provider.created != 0 {
		t.Fatal("checkpoint recovery repeated publication or dispatched discovery")
	}
	page, err := api.Store.All(t.Context(), "preparation-queue/"+api.Location, "alpha")
	if err != nil || len(page) != 1 {
		t.Fatal("recovery duplicated or lost the discovery queue", len(page), err)
	}
}

func TestAssemblyHandoffCancellationExpiryAndRevocationDoNotDispatch(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "expire-waiting", "expired-before-sweep", "revoke-waiting", "cancel-during-handoff"} {
		t.Run(mode, func(t *testing.T) {
			api, writer, builder, provider := assemblySetup(t)
			queued, _ := enqueueZIP(t, api, "assembly-"+mode)
			if mode == "cancel-during-handoff" {
				builder.beforeComplete = func() {
					row, _ := storedAssemblyJob(t, api, queued.ID)
					if err := api.CancelJob(t.Context(), "admin", "alpha", "health", queued.ID, row.Revision); err != nil {
						t.Fatal(err)
					}
				}
			}
			if processed, err := writer.ProcessBuildNext(t.Context(), "registry-writer"); !processed || err != nil {
				t.Fatal(processed, err)
			}
			row, job := storedAssemblyJob(t, api, queued.ID)
			switch mode {
			case "cancel-waiting":
				if err := api.CancelJob(t.Context(), "admin", "alpha", "health", queued.ID, row.Revision); err != nil {
					t.Fatal(err)
				}
			case "expire-waiting", "expired-before-sweep":
				stored, err := api.Store.Get(t.Context(), "source-bundles/alpha", job.Bundle)
				if err != nil {
					t.Fatal(err)
				}
				bundle, err := storage.Decode[sources.Bundle](stored)
				if err != nil {
					t.Fatal(err)
				}
				bundle.ExpiresAt = time.Now().UTC().Add(-time.Second)
				change, _ := storage.Put(stored.Bucket, stored.ID, stored.Project, stored.Revision, bundle)
				if err := api.Store.CommitAll(t.Context(), change); err != nil {
					t.Fatal(err)
				}
				if mode == "expire-waiting" {
					indexed, err := api.Store.Get(t.Context(), "source-bundle-staging", job.Bundle)
					if err != nil {
						t.Fatal(err)
					}
					value, err := storage.Decode[map[string]any](indexed)
					if err != nil {
						t.Fatal(err)
					}
					value["expiresAt"] = bundle.ExpiresAt
					changed, _ := storage.Put(indexed.Bucket, indexed.ID, indexed.Project, indexed.Revision, value)
					if err := api.Store.CommitAll(t.Context(), changed); err != nil {
						t.Fatal(err)
					}
					if _, err := api.SweepBundles(t.Context(), ""); err != nil {
						t.Fatal(err)
					}
				}
			case "revoke-waiting":
				user, err := api.Auth.Directory.User(t.Context(), "admin")
				if err != nil {
					t.Fatal(err)
				}
				current, err := api.Store.Get(t.Context(), "users", "admin")
				if err != nil {
					t.Fatal(err)
				}
				user.Disabled = true
				if err := api.Auth.Directory.SaveUser(t.Context(), user, current.Revision); err != nil {
					t.Fatal(err)
				}
			case "cancel-during-handoff":
				builder.beforeComplete = nil
				expireAssemblyLease(t, writer, queued.ID)
				if err := writer.RecoverPreparations(t.Context(), "cleanup-writer"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := api.ProcessNext(t.Context(), "api-discovery"); err != nil {
				t.Fatal(err)
			}
			_, finished := storedAssemblyJob(t, api, queued.ID)
			if finished.State != "cancelled" && finished.State != "expired" && finished.State != "failed" {
				t.Fatal("withdrawn handoff remained runnable", finished.State)
			}
			if provider.created != 0 || provider.discovered != 0 || builder.calls != 1 || finished.Upload != "" {
				t.Fatal("withdrawn handoff executed or gained review")
			}
		})
	}
}

func TestAssemblyOnlyConcurrentWritersClaimOnceAndRejectDestructiveBuilders(t *testing.T) {
	api, writer, builder, provider := assemblySetup(t)
	queued, _ := enqueueZIP(t, api, "concurrent-assembly")
	unsupported := *writer
	unsupported.Builders = map[string]preparation.ArtifactBuilder{"go": builder.hostedBuilder}
	if processed, err := unsupported.ProcessBuildNext(t.Context(), "unsupported"); processed || err == nil {
		t.Fatal("assembly-only worker accepted a builder without persistent artifact handoff")
	}
	var group sync.WaitGroup
	for _, worker := range []string{"writer-one", "writer-two"} {
		group.Go(func() {
			if _, err := writer.ProcessBuildNext(t.Context(), worker); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	assertPendingAssembly(t, api, queued.ID)
	if builder.calls != 1 || builder.completed != 1 || provider.created != 0 {
		t.Fatal("concurrent writers published twice or ran discovery")
	}
}
