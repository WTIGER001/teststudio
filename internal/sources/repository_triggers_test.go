package sources_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"strings"
	"sync"
	"testing"
	"time"

	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	githubsource "github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

type triggeredProvider struct {
	*repositoryProvider
	*githubsource.Client
}

func (p *triggeredProvider) Validate(access sourceprovider.Access, ref string) error {
	return p.repositoryProvider.Validate(access, ref)
}
func (p *triggeredProvider) Resolve(ctx context.Context, access sourceprovider.Access, ref string) (sourceprovider.Revision, error) {
	return p.repositoryProvider.Resolve(ctx, access, ref)
}
func (p *triggeredProvider) Archive(ctx context.Context, access sourceprovider.Access, ref sourceprovider.Revision) (sourceprovider.Archive, error) {
	return p.repositoryProvider.Archive(ctx, access, ref)
}
func configureTriggers(t *testing.T, s *sources.Service, p *repositoryProvider, periodic bool) sources.RepositorySource {
	t.Helper()
	t.Setenv("TESTSTUDIO_UNIT_WEBHOOK", strings.Repeat("secret", 8))
	credentials, err := localsecrets.New([]localsecrets.Binding{{ID: "hook", Project: "alpha", Purpose: sourceprovider.WebhookPurpose, Environment: "TESTSTUDIO_UNIT_WEBHOOK"}})
	if err != nil {
		t.Fatal(err)
	}
	s.Secrets = credentials
	s.Providers["github"] = &triggeredProvider{repositoryProvider: p, Client: &githubsource.Client{Secrets: credentials}}
	row, err := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	if err != nil {
		t.Fatal(err)
	}
	source, _ := storage.Decode[sources.RepositorySource](row)
	config := source.Configuration
	config.Push = true
	config.WebhookCredential = "hook"
	if periodic {
		config.IntervalSeconds = 60
	}
	if err := s.SaveRepository(t.Context(), "admin", "alpha", "monorepo", config, row.Revision); err != nil {
		t.Fatal(err)
	}
	row, err = s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	if err != nil {
		t.Fatal(err)
	}
	source, _ = storage.Decode[sources.RepositorySource](row)
	return source
}
func signedPush(t *testing.T, id, ref, repo, commit string, deleted bool) sourceprovider.WebhookRequest {
	t.Helper()
	data, err := json.Marshal(map[string]any{"ref": ref, "after": commit, "deleted": deleted, "repository": map[string]any{"id": 42, "full_name": repo}})
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(strings.Repeat("secret", 8)))
	mac.Write(data)
	return sourceprovider.WebhookRequest{Delivery: id, Event: "push", Body: data, Signature: "sha256=" + hex.EncodeToString(mac.Sum(nil))}
}
func TestRepositoryPushDeduplicatesAndResolvesCurrentReferenceAfterDelayedDelivery(t *testing.T) {
	s, _, provider, builder := repositorySetup(t, false, false)
	configureTriggers(t, s, provider, false)
	request := signedPush(t, "push-one", "refs/heads/main", "owner/suite", strings.Repeat("a", 40), false)
	if _, err := s.ReceiveRepositoryWebhook(t.Context(), "alpha", "monorepo", request); !errors.Is(err, sourceprovider.ErrWebhookRejected) {
		t.Fatal("push pinned repository identity before initial sync", err)
	}
	syncRepository(t, s, "initial", false)
	provider.commit = strings.Repeat("b", 40)
	first, err := s.ReceiveRepositoryWebhook(t.Context(), "alpha", "monorepo", request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.ReceiveRepositoryWebhook(t.Context(), "alpha", "monorepo", request)
	if err != nil || first.Sync != second.Sync {
		t.Fatal("duplicate push", second, err)
	}
	altered := signedPush(t, "push-one", "refs/heads/main", "owner/suite", strings.Repeat("c", 40), false)
	if _, err := s.ReceiveRepositoryWebhook(t.Context(), "alpha", "monorepo", altered); !errors.Is(err, sourceprovider.ErrWebhookRejected) {
		t.Fatal("delivery collision accepted", err)
	}
	if done, err := s.ProcessRepositorySync(t.Context(), "push-worker"); err != nil || !done {
		t.Fatal(done, err)
	}
	row, err := s.ReadRepositorySync(t.Context(), "admin", "alpha", "monorepo", first.Sync)
	if err != nil {
		t.Fatal(err)
	}
	result, _ := storage.Decode[sources.RepositorySync](row)
	if result.Trigger != "push" || result.Revision.Commit != provider.commit || len(result.Preparations) != 1 || builder.calls != 0 {
		t.Fatalf("delayed push: %+v", result)
	}
	for _, request := range []sourceprovider.WebhookRequest{
		signedPush(t, "other-ref", "refs/heads/other", "owner/suite", provider.commit, false),
		signedPush(t, "deleted", "refs/heads/main", "owner/suite", strings.Repeat("0", 40), true),
	} {
		receipt, err := s.ReceiveRepositoryWebhook(t.Context(), "alpha", "monorepo", request)
		if err != nil || receipt.State != "ignored" {
			t.Fatal(receipt, err)
		}
	}
	other := signedPush(t, "foreign", "refs/heads/main", "other/repository", provider.commit, false)
	if _, err := s.ReceiveRepositoryWebhook(t.Context(), "alpha", "monorepo", other); !errors.Is(err, sourceprovider.ErrWebhookRejected) {
		t.Fatal("foreign repository accepted", err)
	}
	rows, _ := s.Store.All(t.Context(), "runs", "alpha")
	if len(rows) != 0 {
		t.Fatal("push executed tests")
	}
}
func TestRepositoryPeriodicSyncClaimsOnceAndCoalescesPendingWork(t *testing.T) {
	s, _, provider, _ := repositorySetup(t, false, false)
	source := configureTriggers(t, s, provider, true)
	before := source.NextSyncAt.Add(-time.Second)
	if _, err := s.TickRepositorySchedules(t.Context(), "", before); err != nil {
		t.Fatal(err)
	}
	history, _ := s.ListRepositorySyncs(t.Context(), "admin", "alpha", "monorepo", "", 30)
	if len(history.Items) != 0 {
		t.Fatal("timer fired early")
	}
	due := source.NextSyncAt.Add(10 * time.Minute)
	var workers sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		workers.Go(func() { _, err := s.TickRepositorySchedules(t.Context(), "", due); failures <- err })
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	history, _ = s.ListRepositorySyncs(t.Context(), "admin", "alpha", "monorepo", "", 30)
	if len(history.Items) != 1 {
		t.Fatal("replicas queued duplicate syncs", len(history.Items))
	}
	first, _ := storage.Decode[sources.RepositorySync](history.Items[0])
	if first.Trigger != "periodic" || provider.resolves != 0 {
		t.Fatal("scheduler executed source fetch")
	}
	row, _ := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	source, _ = storage.Decode[sources.RepositorySource](row)
	if source.NextSyncAt == nil || !source.NextSyncAt.Equal(due.Add(time.Minute)) {
		t.Fatal("missed intervals were not coalesced")
	}
	if _, err := s.TickRepositorySchedules(t.Context(), "", *source.NextSyncAt); err != nil {
		t.Fatal(err)
	}
	history, _ = s.ListRepositorySyncs(t.Context(), "admin", "alpha", "monorepo", "", 30)
	if len(history.Items) != 1 {
		t.Fatal("pending sync was duplicated")
	}
	if done, err := s.ProcessRepositorySync(t.Context(), "timer-worker"); err != nil || !done {
		t.Fatal(done, err)
	}
	row, _ = s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	source, _ = storage.Decode[sources.RepositorySource](row)
	source.Configuration.Enabled = false
	if err := s.SaveRepository(t.Context(), "admin", "alpha", "monorepo", source.Configuration, row.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TickRepositorySchedules(t.Context(), "", due.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	history, _ = s.ListRepositorySyncs(t.Context(), "admin", "alpha", "monorepo", "", 30)
	if len(history.Items) != 1 {
		t.Fatal("disabled source fired")
	}
}

func TestRepositoryPeriodicRevocationAdvancesTimerWithoutQueueingWork(t *testing.T) {
	s, _, provider, _ := repositorySetup(t, false, false)
	source := configureTriggers(t, s, provider, true)
	control := &access.Service{Store: s.Store, Auth: s.Auth}
	if _, err := control.CreateGrant(t.Context(), "admin", "alpha", access.GrantDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: "admin", Permission: "teststudio.sources.manage", Effect: permissions.EffectDeny, Object: "source/monorepo", Reason: "Verify background source revocation"}); err != nil {
		t.Fatal(err)
	}
	due := *source.NextSyncAt
	if _, err := s.TickRepositorySchedules(t.Context(), "", due); err != nil {
		t.Fatal(err)
	}
	row, err := s.ReadRepository(t.Context(), "admin", "alpha", "monorepo")
	if err != nil {
		t.Fatal(err)
	}
	blocked, _ := storage.Decode[sources.RepositorySource](row)
	if blocked.LastState != "blocked" || blocked.LastError == "" || blocked.NextSyncAt == nil || !blocked.NextSyncAt.Equal(due.Add(time.Minute)) {
		t.Fatalf("blocked timer: %+v", blocked)
	}
	history, err := s.ListRepositorySyncs(t.Context(), "admin", "alpha", "monorepo", "", 30)
	if err != nil || len(history.Items) != 0 || provider.resolves != 0 {
		t.Fatal("revoked timer queued work", err)
	}
}
