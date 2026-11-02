package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func preparedRun(t *testing.T) (*client, string) {
	t.Helper()
	c := fixture(t)
	c.signIn()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"status":"up"}`)) }))
	t.Cleanup(target.Close)
	steps := []struct {
		method, path string
		input        any
		headers      map[string]string
	}{
		{"POST", "/api/v1/projects", projects.Project{ID: "example", Name: "Example"}, nil},
		{"PUT", "/api/v1/projects/example/applications/app", projects.Application{ID: "app", Name: "Application"}, map[string]string{"If-Match": "0"}},
		{"PUT", "/api/v1/projects/example/environments/local", projects.Environment{ID: "local", Name: "Local", Application: "app", Stage: "local", Endpoints: map[string]string{"api": target.URL}}, map[string]string{"If-Match": "0"}},
		{"PUT", "/api/v1/projects/example/suites/health", projects.Suite{ID: "health", Name: "Health", Applications: []string{"app"}, RequiresTarget: true}, map[string]string{"If-Match": "0"}},
		{"POST", "/api/v1/projects/example/suites/health/tests", catalog.Test{ID: "HEALTH-1", Title: "Availability", Application: "app", Category: "Health", Type: "https", Runner: "http", State: "enabled", Safety: "safe", Rationale: "Reads the registered health endpoint.", TimeoutSeconds: 10, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/healthz", Method: "GET", StatusCodes: []int{200}, RequiredText: "up"}}, nil},
	}
	for _, step := range steps {
		if status, data := c.call(step.method, step.path, step.input, step.headers); status != 200 {
			t.Fatalf("%s: %d %v", step.path, status, data)
		}
	}
	base := "/api/v1/projects/example/suites/health/catalog-drafts"
	status, draft := c.call("POST", base, catalog.Artifact{}, nil)
	if status != 200 {
		t.Fatal(draft)
	}
	if status, data := c.call("POST", base+"/"+draft["id"].(string)+"/publish", map[string]bool{"activate": true}, map[string]string{"If-Match": "1", "Idempotency-Key": "release"}); status != 200 {
		t.Fatal(data)
	}
	status, run := c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local"}, map[string]string{"Idempotency-Key": "run"})
	if status != 200 {
		t.Fatal(run)
	}
	return c, run["id"].(string)
}

func TestRunEvidenceWorkflowThroughLocalAPI(t *testing.T) {
	c, id := preparedRun(t)
	base := "/api/v1/projects/example/runs/" + id
	deadline := time.Now().Add(5 * time.Second)
	var observed map[string]any
	for time.Now().Before(deadline) {
		status, data := c.call("GET", base, nil, nil)
		if status != 200 {
			t.Fatal(data)
		}
		observed = data["data"].(map[string]any)
		if runs.Terminal(observed["state"].(string)) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if observed["outcome"] != "passed" || observed["evidenceState"] != "complete" {
		t.Fatalf("run: %v", observed)
	}
	for _, kind := range []string{"attempts", "events", "artifacts"} {
		status, data := c.call("GET", base+"/"+kind, nil, nil)
		if status != 200 || len(data["items"].([]any)) == 0 {
			t.Fatalf("%s: %d %v", kind, status, data)
		}
	}
	if status, data := c.call("GET", base+"/logs?tail=true", nil, nil); status != 200 || len(data["segments"].([]any)) == 0 {
		t.Fatalf("logs: %d %v", status, data)
	}
	if status, data := c.call("GET", "/api/v1/projects/example/runs?environment=local&selection=full", nil, nil); status != 200 || len(data["items"].([]any)) != 1 {
		t.Fatalf("list: %d %v", status, data)
	}
	status, data := c.call("GET", base+"/artifacts", nil, nil)
	if status != 200 {
		t.Fatal(data)
	}
	a := data["items"].([]any)[0].(map[string]any)
	access := base + "/artifacts/" + a["id"].(string) + "/access"
	status, grant := c.call("POST", access, map[string]string{"action": "download"}, nil)
	if status != 200 {
		t.Fatal(grant)
	}
	u := grant["url"].(string)
	response, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || len(content) == 0 || !strings.HasPrefix(response.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("download: %d %s %v", response.StatusCode, content, err)
	}
	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Range", "bytes=0-2")
	response, err = http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	partial, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 206 || len(partial) != 3 {
		t.Fatalf("range: %d %q", response.StatusCode, partial)
	}
	if status, _ := c.call("POST", "/auth/logout", map[string]any{}, nil); status != 200 {
		t.Fatal(status)
	}
	response, err = http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 401 {
		t.Fatalf("grant survives logout: %d", response.StatusCode)
	}
}

func TestArtifactGrantsEnforceCurrentPermissionsAndOriginIsolation(t *testing.T) {
	c, id := preparedRun(t)
	artifact, err := c.api.Evidence.SaveBytes(t.Context(), evidence.Artifact{Project: "example", Run: id, Kind: "report", Filename: "report.html", ContentType: "text/html"}, []byte("<script>document.cookie</script>"), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/projects/example/runs/" + id + "/artifacts/" + artifact.ID + "/access"
	status, data := c.call("POST", base, map[string]string{"action": "view"}, nil)
	if status != 200 {
		t.Fatal(data)
	}
	u := data["url"].(string)
	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", c.origin)
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 || !strings.HasPrefix(response.Header.Get("Content-Disposition"), "attachment") || response.Header.Get("Access-Control-Allow-Origin") != c.origin || !strings.Contains(response.Header.Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("unsafe artifact response: %d %v", response.StatusCode, response.Header)
	}
	if err := c.api.Auth.Engine.UnassignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	response, err = http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatalf("grant survives revocation: %d", response.StatusCode)
	}
	response, err = http.Get(c.api.ArtifactOrigin + "/access/" + strings.Repeat("x", 43))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 404 {
		t.Fatalf("unknown grant: %d", response.StatusCode)
	}
}

func TestArtifactGrantExpiryAndMissingContent(t *testing.T) {
	c, id := preparedRun(t)
	a, err := c.api.Evidence.SaveBytes(t.Context(), evidence.Artifact{Project: "example", Run: id, Kind: "log", Filename: "extra.log", ContentType: "text/plain"}, []byte("extra log"), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	access := "/api/v1/projects/example/runs/" + id + "/artifacts/" + a.ID + "/access"
	status, data := c.call("POST", access, map[string]string{"action": "view"}, nil)
	if status != 200 {
		t.Fatal(data)
	}
	u := data["url"].(string)
	token := u[strings.LastIndex(u, "/")+1:]
	row, err := c.api.Store.Get(t.Context(), "artifact-access", sessions.Digest(token))
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[map[string]json.RawMessage](row)
	if err != nil {
		t.Fatal(err)
	}
	value["expiresAt"], err = json.Marshal(time.Now().Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err == nil {
		err = c.api.Store.Commit(t.Context(), change)
	}
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 401 {
		t.Fatalf("expired grant: %d", response.StatusCode)
	}
	status, data = c.call("POST", access, map[string]string{"action": "view"}, nil)
	if status != 200 {
		t.Fatal(data)
	}
	if err := c.api.Evidence.Blobs.Delete(t.Context(), a.Project+"/"+a.Run+"/"+a.ID+"/"+a.Checksum); err != nil {
		t.Fatal(err)
	}
	response, err = http.Get(data["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 410 {
		t.Fatalf("missing artifact: %d", response.StatusCode)
	}
	row, err = c.api.Store.Get(t.Context(), evidence.Bucket(id), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := storage.Decode[evidence.Artifact](row)
	if err != nil || missing.State != "missing" {
		t.Fatalf("missing state: %+v %v", missing, err)
	}
}
