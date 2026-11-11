package httpapi_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	githubsource "github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/internal/storage"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/sources"
)

type apiRepositoryProvider struct{ calls int }

func (p *apiRepositoryProvider) Validate(sourceprovider.Access, string) error { return nil }
func (p *apiRepositoryProvider) Resolve(context.Context, sourceprovider.Access, string) (sourceprovider.Revision, error) {
	p.calls++
	return sourceprovider.Revision{}, sourceprovider.ErrUnavailable
}
func (p *apiRepositoryProvider) Archive(context.Context, sourceprovider.Access, sourceprovider.Revision) (sourceprovider.Archive, error) {
	p.calls++
	return sourceprovider.Archive{}, sourceprovider.ErrUnavailable
}

func TestRepositoryAPIRegistersAndQueuesWithoutRemoteRequests(t *testing.T) {
	provider := &apiRepositoryProvider{}
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", BundleRunners: []string{"go"}, Providers: map[string]sourceprovider.Provider{"github": provider}}
	})
	c.signIn()
	if status, value := c.call("POST", "/api/v1/projects", map[string]string{"id": "repository", "name": "Repository"}, nil); status != 200 {
		t.Fatal(value)
	}
	project := "/api/v1/projects/repository"
	if status, value := c.call("PUT", project+"/applications/app", map[string]string{"id": "app", "name": "Application"}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(value)
	}
	if status, value := c.call("PUT", project+"/suites/unit", map[string]any{"id": "unit", "name": "Unit", "requiresTarget": false, "applications": []string{"app"}}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(value)
	}
	config := map[string]any{"name": "Project repository", "provider": "github", "repository": "owner/suite", "reference": "heads/main", "enabled": true, "autoActivate": false, "bindings": []map[string]string{{"suiteId": "unit", "manifestPath": "suites/unit.json", "runner": "go", "buildProfile": "go-module"}}}
	base := project + "/repository-sources/source"
	if status, _ := c.call("PUT", base, config, map[string]string{"If-Match": "0", "X-CSRF-Token": ""}); status != 403 {
		t.Fatal("repository configuration skipped CSRF protection")
	}
	status, source := c.call("PUT", base, config, map[string]string{"If-Match": "0"})
	if status != 200 {
		t.Fatal(source)
	}
	if provider.calls != 0 {
		t.Fatal("registration contacted a remote provider")
	}
	status, options := c.call("GET", project+"/repository-source-options", nil, nil)
	if status != 200 || len(options["providers"].([]any)) != 1 || len(options["credentials"].([]any)) != 0 {
		t.Fatal("repository options exposed credentials or lost enabled providers")
	}
	if status, _ := c.call("POST", base+"/syncs", map[string]bool{"force": false}, nil); status != 422 {
		t.Fatal("sync accepted a missing idempotency key")
	}
	status, queued := c.call("POST", base+"/syncs", map[string]bool{"force": false}, map[string]string{"Idempotency-Key": "manual-sync"})
	if status != 200 {
		t.Fatal(queued)
	}
	status, repeated := c.call("POST", base+"/syncs", map[string]bool{"force": false}, map[string]string{"Idempotency-Key": "manual-sync"})
	if status != 200 || repeated["id"] != queued["id"] || provider.calls != 0 {
		t.Fatal("request duplicated synchronization or executed remote work")
	}
	if processed, err := c.api.Sources.ProcessRepositorySync(t.Context(), "source-worker"); err != nil || !processed {
		t.Fatal("queued source worker", processed, err)
	}
	status, finished := c.call("GET", base+"/syncs/"+queued["id"].(string), nil, nil)
	if status != 200 || finished["data"].(map[string]any)["state"] != "failed" {
		t.Fatal("provider failure became a successful sync")
	}
	status, runs := c.call("GET", project+"/runs", nil, nil)
	if status != 200 || len(runs["items"].([]any)) != 0 {
		t.Fatal("source failure created a test run")
	}
	status, current := c.call("GET", base, nil, nil)
	if status != 200 {
		t.Fatal(current)
	}
	config["enabled"] = false
	if status, value := c.call("PUT", base, config, map[string]string{"If-Match": jsonRevision(current)}); status != 200 {
		t.Fatal(value)
	}
	if status, value := c.call("POST", base+"/syncs", map[string]bool{"force": false}, map[string]string{"Idempotency-Key": "disabled"}); status != 422 || !strings.Contains(value["message"].(string), "disabled") {
		t.Fatal("disabled source accepted synchronization")
	}
}
func jsonRevision(row map[string]any) string {
	return strconv.FormatInt(int64(row["revision"].(float64)), 10)
}

func TestRepositoryWebhookRequiresSignatureWithoutBrowserSession(t *testing.T) {
	t.Setenv("TESTSTUDIO_API_WEBHOOK", strings.Repeat("secret", 8))
	credentials, err := localsecrets.New([]localsecrets.Binding{{ID: "hook", Project: "repository", Purpose: sourceprovider.WebhookPurpose, Environment: "TESTSTUDIO_API_WEBHOOK"}})
	if err != nil {
		t.Fatal(err)
	}
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", BundleRunners: []string{"go"}, Secrets: credentials, Providers: map[string]sourceprovider.Provider{"github": &githubsource.Client{Secrets: credentials}}}
	})
	c.signIn()
	if status, value := c.call("POST", "/api/v1/projects", map[string]string{"id": "repository", "name": "Repository"}, nil); status != 200 {
		t.Fatal(value)
	}
	base := "/api/v1/projects/repository"
	if status, value := c.call("PUT", base+"/applications/app", map[string]string{"id": "app", "name": "Application"}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(value)
	}
	if status, value := c.call("PUT", base+"/suites/unit", map[string]any{"id": "unit", "name": "Unit", "requiresTarget": false, "applications": []string{"app"}}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(value)
	}
	config := sources.RepositoryConfig{Name: "Repository", Provider: "github", Repository: "owner/suite", Reference: "heads/main", Enabled: true, Push: true, WebhookCredential: "hook", Bindings: []sources.RepositoryBinding{{Suite: "unit", Manifest: "suite.json", Runner: "go", Profile: "go-module"}}}
	if err := c.api.Sources.SaveRepository(t.Context(), "admin", "repository", "source", config, 0); err != nil {
		t.Fatal(err)
	}
	row, err := c.api.Sources.ReadRepository(t.Context(), "admin", "repository", "source")
	if err != nil {
		t.Fatal(err)
	}
	source, _ := storage.Decode[sources.RepositorySource](row)
	source.RepositoryID = "42"
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, source)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"ref":"refs/heads/main","after":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","repository":{"id":42,"full_name":"owner/suite"}}`)
	mac := hmac.New(sha256.New, []byte(strings.Repeat("secret", 8)))
	mac.Write(data)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	deliver := func(signature string) (int, map[string]any) {
		t.Helper()
		request, err := http.NewRequestWithContext(t.Context(), "POST", c.origin+base+"/repository-sources/source/webhooks/github", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("X-GitHub-Delivery", "delivery-one")
		request.Header.Set("X-GitHub-Event", "push")
		request.Header.Set("X-Hub-Signature-256", signature)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var result map[string]any
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, result
	}
	if status, _ := deliver(""); status != 403 {
		t.Fatal("unsigned webhook accepted", status)
	}
	status, receipt := deliver(signature)
	if status != 202 || receipt["state"] != "queued" {
		t.Fatal(status, receipt)
	}
	status, repeated := deliver(signature)
	if status != 202 || repeated["syncId"] != receipt["syncId"] {
		t.Fatal("duplicate delivery", status, repeated)
	}
	history, err := c.api.Sources.ListRepositorySyncs(t.Context(), "admin", "repository", "source", "", 30)
	if err != nil || len(history.Items) != 1 {
		t.Fatal("webhook did not queue exactly once", err)
	}
	if rows, err := c.api.Store.All(t.Context(), "runs", "repository"); err != nil || len(rows) != 0 {
		t.Fatal("webhook executed tests")
	}
}
