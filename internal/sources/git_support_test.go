package sources_test

import (
	"context"
	"errors"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestGitRunStartChecksFreshSourceAndPreservesRetainedVersion(t *testing.T) {
	s, c, p, _ := repositorySetup(t, false, true)
	first := syncRepository(t, s, "initial", false)
	processBundle(t, s, first.Preparations["health"])
	if _, err := s.ProcessRepositoryPublication(t.Context()); err != nil {
		t.Fatal(err)
	}
	active, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil {
		t.Fatal(err)
	}
	beforeFetches, beforeResolves := p.fetches, p.resolves
	for range 2 {
		checked, guards, err := s.CheckRunSource(t.Context(), "admin", "alpha", "health", active)
		if err != nil || checked == nil || len(guards) < 1 {
			t.Fatal("fresh-source check", err)
		}
	}
	if p.resolves != beforeResolves+2 || p.fetches != beforeFetches {
		t.Fatal("start skipped provider resolution or repeated immutable download")
	}
	p.commit = strings.Repeat("b", 40)
	if _, _, err = s.CheckRunSource(t.Context(), "admin", "alpha", "health", active); err == nil {
		t.Fatal("changed branch used stale source")
	}
	row, err := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	if err != nil {
		t.Fatal(err)
	}
	registered, _ := storage.Decode[sources.RepositorySource](row)
	syncRow, err := s.ReadRepositorySync(t.Context(), "admin", "alpha", "monorepo", registered.LastSync)
	if err != nil {
		t.Fatal(err)
	}
	queued, _ := storage.Decode[sources.RepositorySync](syncRow)
	if queued.Trigger != "run" || queued.Revision == nil || queued.Revision.Commit != p.commit {
		t.Fatal("run refresh was not pinned before queueing")
	}
	if processed, err := s.ProcessRepositorySync(t.Context(), "sync-worker"); err != nil || !processed {
		t.Fatal(err)
	}
	syncRow, _ = s.ReadRepositorySync(t.Context(), "admin", "alpha", "monorepo", queued.ID)
	queued, _ = storage.Decode[sources.RepositorySync](syncRow)
	processBundle(t, s, queued.Preparations["health"])
	if _, err = s.ProcessRepositoryPublication(t.Context()); err != nil {
		t.Fatal(err)
	}
	next, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.CheckRunSource(t.Context(), "admin", "alpha", "health", next); err != nil {
		t.Fatal("latest prepared source rejected", err)
	}
	if active.Draft.Artifact.Provenance.Commit == next.Draft.Artifact.Provenance.Commit {
		t.Fatal("retained version was rewritten")
	}
	p.failure = errors.New("private fixture diagnostic")
	if _, _, err = s.CheckRunSource(t.Context(), "admin", "alpha", "health", next); err == nil || strings.Contains(err.Error(), "private fixture") {
		t.Fatal("outage silently used cached source or exposed provider diagnostics")
	}
}
func TestGitDatabaseMetadataRevisionConflictAndNoWrite(t *testing.T) {
	s, c, _, builder := repositorySetup(t, false, false)
	builder.useSnapshot = true
	first := syncRepository(t, s, "metadata", false)
	_, job := processBundle(t, s, first.Preparations["health"])
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	row, err := s.ReadRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health")
	if err != nil {
		t.Fatal(err)
	}
	m, err := storage.Decode[sources.RepositoryMetadata](row)
	if err != nil {
		t.Fatal(err)
	}
	m.Manifest.Applications = []string{"app"}
	m.Catalog.Tests[0].Application = "app"
	m.Catalog.Tests[0].Title = "Reviewed database metadata"
	if err = s.SaveRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", m, row.Revision); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", m, row.Revision); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale metadata save accepted", err)
	}
	if _, err = s.CommitRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", "write", row.Revision+1); err == nil {
		t.Fatal("read-only connection accepted a Git write")
	}
	history, err := s.RepositoryMetadataHistory(t.Context(), "admin", "alpha", "monorepo", "health", "", 20)
	if err != nil || len(history.Items) != 2 {
		t.Fatal("metadata snapshots were not retained", err, len(history.Items))
	}
	retained, err := storage.Decode[sources.RepositoryMetadata](history.Items[1])
	if err != nil || retained.Catalog.Tests[0].Title == m.Catalog.Tests[0].Title {
		t.Fatal("later metadata save rewrote the earlier snapshot", err)
	}
	next := syncRepository(t, s, "updated-metadata", false)
	_, ready := processBundle(t, s, next.Preparations["health"])
	if _, err = s.Publish(t.Context(), "admin", "alpha", "health", ready.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	active, err := c.ActiveRelease(t.Context(), "alpha", "health")
	if err != nil {
		t.Fatal(err)
	}
	if active.Draft.Tests[0].Title != "Reviewed database metadata" {
		t.Fatal("explicit database JSON metadata was overwritten")
	}
	if active.Draft.Artifact.OriginalSourceDigest == "" || active.Draft.Artifact.Provenance.SnapshotDigest == active.Draft.Artifact.OriginalSourceDigest {
		t.Fatal("database overlay did not retain separate original source provenance")
	}
}

type gitWriteProvider struct {
	*repositoryProvider
	writes int
	fail   bool
}

func (p *gitWriteProvider) WriteMetadata(_ context.Context, _ sourceprovider.Access, r sourceprovider.WriteRequest) (sourceprovider.WriteResult, error) {
	p.writes++
	if p.fail {
		return sourceprovider.WriteResult{}, errors.New("fixture disconnected after remote commit")
	}
	return sourceprovider.WriteResult{Commit: strings.Repeat("c", 40), Branch: "teststudio/" + r.Key, ReviewURL: "https://github.com/owner/suite/pull/1"}, nil
}

type gitCredentials struct{}

func (gitCredentials) References(context.Context, string, string) ([]secrets.Reference, error) {
	return []secrets.Reference{{ID: "writer", Name: "Writer"}}, nil
}
func (gitCredentials) Resolve(context.Context, string, string, string) (secrets.Value, error) {
	return secrets.Value{Bytes: []byte("fixture credential")}, nil
}
func TestGitWriteBackPreservesPendingEditsAndRecoversOperation(t *testing.T) {
	s, c, p, _ := repositorySetup(t, false, false)
	first := syncRepository(t, s, "write-base", false)
	_, job := processBundle(t, s, first.Preparations["health"])
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	writer := &gitWriteProvider{repositoryProvider: p, fail: true}
	s.Providers["github"] = writer
	s.Secrets = gitCredentials{}
	row, _ := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	source, _ := storage.Decode[sources.RepositorySource](row)
	config := source.Configuration
	config.Mode = "read-write"
	config.Credential = "writer"
	if err := s.SaveRepository(t.Context(), "admin", "alpha", "monorepo", config, row.Revision); err == nil {
		t.Fatal("metadata authority changed without confirmation")
	}
	config.ConfirmModeChange = true
	if err := s.SaveRepository(t.Context(), "admin", "alpha", "monorepo", config, row.Revision); err != nil {
		t.Fatal(err)
	}
	metadata, _ := s.ReadRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health")
	m, _ := storage.Decode[sources.RepositoryMetadata](metadata)
	m.Manifest.Applications = []string{"app"}
	m.Catalog.Tests[0].Application = "app"
	m.Catalog.Tests[0].Title = "Pending Git title"
	if err := s.SaveRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", m, metadata.Revision); err != nil {
		t.Fatal(err)
	}
	metadata, _ = s.ReadRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health")
	uncertain, err := s.CommitRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", "commit-1", metadata.Revision)
	if err != nil || uncertain.State != "uncertain" {
		t.Fatal("interrupted commit lost recovery", err)
	}
	writer.fail = false
	done, err := s.CommitRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", "commit-1", metadata.Revision)
	if err != nil || done.State != "committed" {
		t.Fatal("commit reconciliation", err)
	}
	if _, err = s.CommitRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health", "commit-1", metadata.Revision); err != nil || writer.writes != 2 {
		t.Fatal("confirmed commit retried a provider mutation", err)
	}
	if current, err := c.ActiveRelease(t.Context(), "alpha", "health"); err != nil || current.Draft.Tests[0].Title == "Pending Git title" {
		t.Fatal("pending Git edit became executable before merge", err)
	}
	p.commit = strings.Repeat("b", 40)
	next := syncRepository(t, s, "remote-edit", false)
	_, prepared := processBundle(t, s, next.Preparations["health"])
	if _, err = s.Publish(t.Context(), "admin", "alpha", "health", prepared.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	metadata, err = s.ReadRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health")
	if err != nil {
		t.Fatal(err)
	}
	pending, _ := storage.Decode[sources.RepositoryMetadata](metadata)
	if !pending.Pending || pending.Catalog.Tests[0].Title != "Pending Git title" {
		t.Fatal("remote synchronization overwrote pending local edits")
	}
	testRow, _ := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "CODE-000")
	test, _ := storage.Decode[catalog.Test](testRow)
	test.Title = "Uncommitted catalog edit"
	if err = c.SaveTest(t.Context(), "admin", "alpha", "health", test, testRow.Revision); err == nil {
		t.Fatal("ordinary catalog mutation bypassed Git metadata ownership")
	}
}

func TestGitMetadataReadHonorsExactTestDeny(t *testing.T) {
	s, _, _, _ := repositorySetup(t, false, false)
	sync := syncRepository(t, s, "metadata-deny", false)
	_, job := processBundle(t, s, sync.Preparations["health"])
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", job.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	control := &access.Service{Store: s.Store, Auth: s.Auth}
	if _, err := control.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.catalog.read", Effect: permissions.EffectDeny, Object: "test/health/CODE-000", Reason: "Verify metadata does not expose a denied test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadRepositoryMetadata(t.Context(), "admin", "alpha", "monorepo", "health"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("repository metadata bypassed an exact test deny", err)
	}
	if _, err := s.RepositoryMetadataHistory(t.Context(), "admin", "alpha", "monorepo", "health", "", 20); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("metadata history bypassed an exact test deny", err)
	}
}
func TestGitCredentialHealthPersistsWithoutProviderRequestsOnRead(t *testing.T) {
	s, _, p, _ := repositorySetup(t, false, false)
	checked, err := s.CheckRepositoryAccess(t.Context(), "admin", "alpha", "monorepo", "")
	if err != nil || checked.State != "ready" {
		t.Fatal("access validation", err)
	}
	before := p.resolves
	status, err := s.RepositoryAccessStatus(t.Context(), "admin", "alpha", "monorepo")
	if err != nil || status.LastSuccessAt == nil || !status.Current || p.resolves != before {
		t.Fatal("health read repeated provider access or lost its retained success", err)
	}
	p.failure = errors.New("private fixture body")
	if _, err = s.CheckRepositoryAccess(t.Context(), "admin", "alpha", "monorepo", ""); err == nil || strings.Contains(err.Error(), "private fixture") {
		t.Fatal("failed validation did not redact the provider error")
	}
	status, err = s.RepositoryAccessStatus(t.Context(), "admin", "alpha", "monorepo")
	if err != nil || status.State != "failed" || status.LastSuccessAt == nil {
		t.Fatal("failed validation lost prior successful evidence", err)
	}
}
