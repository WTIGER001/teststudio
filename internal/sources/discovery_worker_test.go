package sources_test

import (
	"context"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestDiscoveryWorkerConsumesHandoffAndNeverBuildsQueuedSources(t *testing.T) {
	api, writer, builder, provider := assemblySetup(t)
	assembled, _ := enqueueZIP(t, api, "dedicated-discovery")
	if processed, err := writer.ProcessBuildNext(t.Context(), "registry-writer"); !processed || err != nil {
		t.Fatal(processed, err)
	}
	waiting := assertPendingAssembly(t, api, assembled.ID)
	unbuilt, _ := enqueueZIP(t, api, "unbuilt-discovery-input")
	rows, err := api.Store.All(t.Context(), "preparation-build-queue/"+api.Location, "")
	if err != nil || len(rows) != 1 {
		t.Fatal("unbuilt input missing", err)
	}
	legacy := rows[0]
	legacy.Bucket = "preparation-queue/" + api.Location
	legacy.Revision = 0
	legacy.ID = "00000000-unbuilt-before-handoff"
	if err := api.Store.CommitAll(t.Context(), storage.Change{Record: legacy}); err != nil {
		t.Fatal(err)
	}
	api.Builders = map[string]preparation.ArtifactBuilder{"go": builder}
	api.RecoveryBuilders = api.Builders
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	done := api.StartDiscoveryPreparation(ctx)
	t.Cleanup(func() { cancel(); <-done })
	for {
		_, job := storedAssemblyJob(t, api, assembled.ID)
		if job.State == "ready" {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("dedicated discovery did not complete", job.State, job.Error)
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-done
	_, ready := storedAssemblyJob(t, api, assembled.ID)
	_, pending := storedAssemblyJob(t, api, unbuilt.ID)
	if ready.Reference == nil || ready.Reference.ID == waiting.AssemblyReference.ID || provider.created != 1 || provider.discovered != 1 || builder.calls != 1 || pending.State != "received" || pending.Reference != nil || len(api.Builders) != 1 {
		t.Fatal("discovery lost phase ownership, built source, or changed caller configuration")
	}
	expireAssemblyLease(t, api, assembled.ID)
	reader := *api
	reader.Builders, reader.RecoveryBuilders = nil, nil
	if err := reader.RecoverPreparations(t.Context(), "discovery-recovery"); err != nil || provider.removed != 1 || builder.cleanups != 0 {
		t.Fatal("discovery recovery required writer authority", err)
	}
}

func TestDiscoveryWorkerPublishesCompletedRepositoryWithoutBuildingOrSyncing(t *testing.T) {
	service, catalog, repository, builder := repositorySetup(t, false, true)
	synced := syncRepository(t, service, "dedicated-publication", false)
	processBundle(t, service, synced.Preparations["health"])
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	done := service.StartDiscoveryPreparation(ctx)
	t.Cleanup(func() { cancel(); <-done })
	for {
		if _, err := catalog.ActiveRelease(t.Context(), "alpha", "health"); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("discovery worker did not process automatic publication")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-done
	active, err := catalog.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.Draft.Artifact.Provenance.Commit != repository.commit || repository.fetches != 1 || builder.calls != 1 {
		t.Fatal("publication repeated source synchronization or assembly", err)
	}
}
