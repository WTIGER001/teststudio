package sources_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type repositoryProvider struct {
	archive           []byte
	commit            string
	resolves, fetches int
	failure           error
}

func (p *repositoryProvider) Validate(sourceprovider.Access, string) error { return nil }
func (p *repositoryProvider) Resolve(_ context.Context, _ sourceprovider.Access, reference string) (sourceprovider.Revision, error) {
	p.resolves++
	if p.failure != nil {
		return sourceprovider.Revision{}, p.failure
	}
	return sourceprovider.Revision{RepositoryID: "42", Repository: "owner/suite", URL: "https://github.com/owner/suite", Reference: reference, Commit: p.commit}, nil
}
func (p *repositoryProvider) Archive(context.Context, sourceprovider.Access, sourceprovider.Revision) (sourceprovider.Archive, error) {
	p.fetches++
	return sourceprovider.Archive{Content: p.archive, Root: "repository-root"}, nil
}
func repositoryArchive(t *testing.T, second bool) []byte {
	t.Helper()
	input := sourceZIP(t)
	archive, err := zip.NewReader(bytes.NewReader(input), int64(len(input)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range archive.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		entry, err := writer.Create("repository-root/" + file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
		if second && file.Name == "suites/health.json" {
			var manifest suitespec.Manifest
			json.Unmarshal(data, &manifest)
			manifest.Suite = "math"
			extra, _ := json.Marshal(manifest)
			entry, err := writer.Create("repository-root/suites/math.json")
			if err != nil {
				t.Fatal(err)
			}
			entry.Write(extra)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
func repositorySetup(t *testing.T, second, automatic bool) (*sources.Service, *catalog.Service, *repositoryProvider, *bundleBuilder) {
	t.Helper()
	s, c, builder := bundleSetup(t)
	provider := &repositoryProvider{archive: repositoryArchive(t, second), commit: strings.Repeat("a", 40)}
	s.Providers = map[string]sourceprovider.Provider{"github": provider}
	config := sources.RepositoryConfig{Name: "Independent repository", Provider: "github", Repository: "owner/suite", Reference: "heads/main", Enabled: true, AutoActivate: automatic, Bindings: []sources.RepositoryBinding{{Suite: "health", Manifest: "suites/health.json", Runner: "go", Profile: "go-module"}}}
	if second {
		if err := (&projects.Service{Store: s.Store, Auth: s.Auth}).SaveSuite(t.Context(), "admin", "alpha", projects.Suite{ID: "math", Name: "Math", RequiresTarget: false}, 0); err != nil {
			t.Fatal(err)
		}
		config.Bindings = append(config.Bindings, sources.RepositoryBinding{Suite: "math", Manifest: "suites/math.json", Runner: "go", Profile: "go-module"})
	}
	if err := s.SaveRepository(t.Context(), "admin", "alpha", "monorepo", config, 0); err != nil {
		t.Fatal(err)
	}
	return s, c, provider, builder
}
func syncRepository(t *testing.T, s *sources.Service, key string, force bool) sources.RepositorySync {
	t.Helper()
	row, err := s.EnqueueRepositorySync(t.Context(), "admin", "alpha", "monorepo", key, sources.SyncInput{Force: force})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := s.ProcessRepositorySync(t.Context(), "sync-worker")
	if err != nil || !processed {
		t.Fatal("sync worker", processed, err)
	}
	result, err := s.ReadRepositorySync(t.Context(), "admin", "alpha", "monorepo", row.ID)
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[sources.RepositorySync](result)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func TestRepositorySyncSharesSnapshotsAndReusesUnchangedCommitAcrossSuites(t *testing.T) {
	s, c, provider, builder := repositorySetup(t, true, false)
	first := syncRepository(t, s, "first", false)
	if first.State != "completed" || len(first.Preparations) != 2 || provider.fetches != 1 || builder.calls != 0 {
		t.Fatalf("unexpected sync: %+v", first)
	}
	metadata, err := s.Store.Get(t.Context(), "repository-snapshots/alpha", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := storage.Decode[sources.RepositorySnapshot](metadata)
	if snapshot.References != 2 || snapshot.Digest == "" {
		t.Fatal("repository source was not shared across bindings")
	}
	repeated := syncRepository(t, s, "repeat", false)
	if repeated.Preparations["health"] != first.Preparations["health"] || provider.fetches != 1 {
		t.Fatal("unchanged commit repeated source download or preparation")
	}
	for range 2 {
		if processed, err := s.ProcessBuildNext(t.Context(), "builder"); err != nil || !processed {
			t.Fatal("build preparation", processed, err)
		}
	}
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "health", first.Preparations["health"])
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](row)
	if job.State != "ready" || job.Origin.Commit != provider.commit {
		t.Fatalf("repository preparation: %+v", job)
	}
	if _, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "CODE-000"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("preparation altered the live catalog")
	}
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	testRow, _ := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "CODE-000")
	definition, _ := storage.Decode[catalog.Test](testRow)
	definition.Title = "Managed repository title"
	if err := c.SaveTest(t.Context(), "admin", "alpha", "health", definition, testRow.Revision); err != nil {
		t.Fatal(err)
	}
	third := syncRepository(t, s, "after-title", false)
	if third.Preparations["health"] != job.ID || builder.calls != 2 || provider.fetches != 1 {
		t.Fatal("metadata edit rebuilt a published commit")
	}
	if release.Draft.Artifact.Provenance.Commit != provider.commit {
		t.Fatal("release lost exact commit provenance")
	}
	source, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", job.ID)
	if err != nil || !bytes.Equal(source.Content, provider.archive) {
		t.Fatal("exact repository source bytes changed", err)
	}
	if runs, _ := s.Store.All(t.Context(), "runs", "alpha"); len(runs) != 0 {
		t.Fatal("repository sync or publication executed tests")
	}
}
func TestRepositoryReadyReviewRefreshesMetadataWithoutRebuilding(t *testing.T) {
	s, c, provider, builder := repositorySetup(t, false, false)
	first := syncRepository(t, s, "first", false)
	_, job := processBundle(t, s, first.Preparations["health"])
	oldUpload := job.Upload
	managed := catalog.Test{ID: "CODE-000", Title: "Managed before publication", Category: "Unit", Type: "unit", Runner: "go", State: "enabled", Safety: "safe", Rationale: "Computes in memory", TimeoutSeconds: 5, Source: "checks/fresh_test.go", Selector: "./checks::TestVersion1Case0"}
	if err := c.SaveTest(t.Context(), "admin", "alpha", "health", managed, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", oldUpload, 1, true); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("obsolete catalog review did not conflict", err)
	}

	again := syncRepository(t, s, "fresh-review", false)
	row, _ := s.ReadJob(t.Context(), "admin", "alpha", "health", again.Preparations["health"])
	ready, _ := storage.Decode[sources.Job](row)
	if ready.Upload == oldUpload || builder.calls != 1 || provider.fetches != 1 {
		t.Fatal("fresh review repeated a build or kept the obsolete preview")
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	active, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.Draft.Artifact.Provenance.Commit != provider.commit || active.Draft.Tests[0].Title != "Managed before publication" {
		t.Fatal(err)
	}
}
func TestRepositoryAutomaticActivationRejectsOlderSync(t *testing.T) {
	s, c, provider, _ := repositorySetup(t, false, true)
	first := syncRepository(t, s, "first", false)
	_, old := processBundle(t, s, first.Preparations["health"])
	provider.commit = strings.Repeat("b", 40)
	nextRow, err := s.EnqueueRepositorySync(t.Context(), "admin", "alpha", "monorepo", "newer", sources.SyncInput{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ProcessRepositoryPublication(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ActiveRelease(t.Context(), "alpha", "health"); err == nil {
		t.Fatal("older sync activated after a newer request")
	}
	if processed, err := s.ProcessRepositorySync(t.Context(), "sync-worker"); err != nil || !processed {
		t.Fatal(err)
	}
	row, _ := s.ReadRepositorySync(t.Context(), "admin", "alpha", "monorepo", nextRow.ID)
	newer, _ := storage.Decode[sources.RepositorySync](row)
	_, ready := processBundle(t, s, newer.Preparations["health"])
	if _, err := s.ProcessRepositoryPublication(t.Context()); err != nil {
		t.Fatal(err)
	}
	active, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.Draft.Artifact.Provenance.Commit != provider.commit || ready.ID == old.ID {
		t.Fatal("latest prepared commit did not activate", err)
	}
}
func TestRepositoryDisableAndProviderFailurePreserveExistingRelease(t *testing.T) {
	s, c, provider, builder := repositorySetup(t, false, false)
	first := syncRepository(t, s, "first", false)
	_, job := processBundle(t, s, first.Preparations["health"])
	release, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	provider.failure = errors.New("repository temporarily unavailable")
	failed := syncRepository(t, s, "outage", false)
	if failed.State != "failed" || builder.calls != 1 {
		t.Fatal("provider outage became a test failure or started a build")
	}
	active, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil || active.ID != release.ID {
		t.Fatal("provider outage changed active release", err)
	}
	row, _ := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	source, _ := storage.Decode[sources.RepositorySource](row)
	source.Configuration.Enabled = false
	if err := s.SaveRepository(t.Context(), "admin", "alpha", "monorepo", source.Configuration, row.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EnqueueRepositorySync(t.Context(), "admin", "alpha", "monorepo", "disabled", sources.SyncInput{}); err == nil {
		t.Fatal("disabled source queued new sync")
	}
	if sourceBytes, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", job.ID); err != nil || len(sourceBytes.Content) == 0 {
		t.Fatal("disabling source deleted retained release content", err)
	}
}

func TestRepositoryReadDenyProtectsPreparationsAndRetainedSource(t *testing.T) {
	s, _, _, _ := repositorySetup(t, false, false)
	sync := syncRepository(t, s, "source-deny", false)
	_, job := processBundle(t, s, sync.Preparations["health"])
	control := &access.Service{Store: s.Store, Auth: s.Auth}
	if _, err := control.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.sources.read", Effect: permissions.EffectDeny, Object: "source/monorepo", Reason: "Verify exact source read isolation"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("source read deny ignored", err)
	}
	if _, err := s.ReadBundle(t.Context(), "admin", "alpha", "health", job.ID); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("retained archive escaped source read deny", err)
	}
	if _, err := s.Read(t.Context(), "admin", "alpha", "health", job.Upload); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("source review escaped source read deny", err)
	}
	page, err := s.ListJobs(t.Context(), "admin", "alpha", "health", "", 30)
	if err != nil || len(page.Items) != 0 {
		t.Fatal("preparation listing escaped source read deny", err)
	}
}
