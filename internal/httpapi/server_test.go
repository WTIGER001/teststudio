package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	permissionstore "github.com/wtiger001/teststudio/adapters/permissions/sqlite"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type client struct {
	http   *http.Client
	origin string
	csrf   string
	t      *testing.T
	api    *httpapi.Server
}

func fixture(t *testing.T) *client {
	t.Helper()
	root := t.TempDir()
	uiRoot := filepath.Join(root, "ui")
	if err := os.Mkdir(uiRoot, 0700); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(t.Context(), filepath.Join(root, "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	dir := &identity.Directory{Store: db}
	if err := dir.SaveUser(t.Context(), identity.User{ID: "admin", Name: "Administrator"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), permissionstore.New(db), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	sessions := &sessions.Service{Store: db, Directory: dir}
	oidc := &authoidc.Service{Store: db, Directory: dir, Sessions: sessions}
	if err := os.WriteFile(filepath.Join(uiRoot, "index.html"), []byte("<!doctype html><html lang=\"en\"><title>TestStudio</title><main>TestStudio</main></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiRoot, "theme.js"), []byte("document.documentElement.dataset.theme='dark'"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := filesystem.New(filepath.Join(root, "private-artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })
	evidenceService := &evidence.Service{Store: db, Blobs: files}
	runService := &runs.Service{Store: db, Auth: auth, Catalog: &catalog.Service{Store: db, Auth: auth}, Evidence: evidenceService, HTTP: &execution.HTTP{}, Location: "local", PollInterval: 5 * time.Millisecond}
	workerContext, cancelWorkers := context.WithCancel(t.Context())
	workersDone := runService.Start(workerContext)
	t.Cleanup(func() { cancelWorkers(); <-workersDone })
	api := &httpapi.Server{Store: db, Auth: auth, Projects: &projects.Service{Store: db, Auth: auth}, Catalog: &catalog.Service{Store: db, Auth: auth}, Sessions: sessions, OIDC: oidc, LocalUser: "admin", UI: uiRoot, Runs: runService, Evidence: evidenceService}
	artifactServer := httptest.NewServer(api.ArtifactHandler())
	t.Cleanup(artifactServer.Close)
	api.ArtifactOrigin = artifactServer.URL
	server := httptest.NewServer(api.Handler())
	t.Cleanup(server.Close)
	sessions.Origin = server.URL
	oidc.Origin = server.URL
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &client{http: &http.Client{Jar: jar}, origin: server.URL, t: t, api: api}
}

func (c *client) call(method, path string, value any, headers map[string]string) (int, map[string]any) {
	c.t.Helper()
	var data []byte
	var err error
	if value != nil {
		data, err = json.Marshal(value)
		if err != nil {
			c.t.Fatal(err)
		}
	}
	r, err := http.NewRequest(method, c.origin+path, bytes.NewReader(data))
	if err != nil {
		c.t.Fatal(err)
	}
	r.Header.Set("Origin", c.origin)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", c.csrf)
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	response, err := c.http.Do(r)
	if err != nil {
		c.t.Fatal(err)
	}
	defer response.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		c.t.Fatal(err)
	}
	return response.StatusCode, result
}

func (c *client) signIn() {
	c.t.Helper()
	if status, data := c.call("POST", "/auth/local", map[string]any{}, nil); status != 200 {
		c.t.Fatalf("local sign-in %d: %v", status, data)
	}
	status, data := c.call("GET", "/auth/session", nil, nil)
	if status != 200 {
		c.t.Fatal(data)
	}
	c.csrf = data["session"].(map[string]any)["csrfToken"].(string)
}

func TestCatalogWorkflowThroughAuthenticatedAPI(t *testing.T) {
	c := fixture(t)
	if status, _ := c.call("GET", "/api/v1/projects", nil, nil); status != 401 {
		t.Fatalf("anonymous projects: %d", status)
	}
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", projects.Project{ID: "example", Name: "Example"}, nil); status != 200 {
		t.Fatalf("project %d: %v", status, data)
	}
	base := "/api/v1/projects/example/suites/health"
	if status, data := c.call("PUT", base, projects.Suite{ID: "health", Name: "Health", RequiresTarget: true}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatalf("suite %d: %v", status, data)
	}
	test := catalog.Test{ID: "HEALTH-1", Title: "Availability", Category: "Health", Type: "https", Runner: "http", State: "enabled", Safety: "safe", Rationale: "Reads the registered health endpoint.", TimeoutSeconds: 10, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/healthz", Method: "GET", StatusCodes: []int{200}}}
	if status, data := c.call("POST", base+"/tests", test, nil); status != 200 {
		t.Fatalf("test %d: %v", status, data)
	}
	status, draft := c.call("POST", base+"/catalog-drafts", catalog.Artifact{}, nil)
	if status != 200 {
		t.Fatalf("draft %d: %v", status, draft)
	}
	draftID := draft["id"].(string)
	if status, result := c.call("POST", base+"/catalog-drafts/"+draftID+"/validate", map[string]any{}, nil); status != 200 || result["valid"] != true {
		t.Fatalf("validation %d: %v", status, result)
	}
	if status, result := c.call("POST", base+"/catalog-drafts/"+draftID+"/publish", map[string]bool{"activate": true}, map[string]string{"If-Match": "1", "Idempotency-Key": "publication"}); status != 200 {
		t.Fatalf("publication %d: %v", status, result)
	}
	test.Title = "Updated title"
	if status, result := c.call("PUT", base+"/tests/HEALTH-1", test, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatalf("update %d: %v", status, result)
	}
	if status, _ := c.call("PUT", base+"/tests/HEALTH-1", test, map[string]string{"If-Match": "1"}); status != 409 {
		t.Fatalf("stale edit status: %d", status)
	}
	if status, _ := c.call("POST", base+"/tests/HEALTH-1/actions/archive", map[string]any{}, map[string]string{"If-Match": "2"}); status != 200 {
		t.Fatalf("archive: %d", status)
	}
	if status, result := c.call("GET", base+"/tests/HEALTH-1/revisions", nil, nil); status != 200 || len(result["items"].([]any)) != 3 {
		t.Fatalf("history %d: %v", status, result)
	}
}

func TestAPIRejectsCrossOriginAndInvalidJSON(t *testing.T) {
	c := fixture(t)
	c.signIn()
	if status, _ := c.call("POST", "/api/v1/projects", projects.Project{ID: "bad", Name: "Bad"}, map[string]string{"Origin": "https://other.example"}); status != 403 {
		t.Fatalf("cross-origin: %d", status)
	}
	r, err := http.NewRequest("POST", c.origin+"/api/v1/projects", strings.NewReader(`{"id":"bad","name":"Bad"} garbage`))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", c.origin)
	r.Header.Set("X-CSRF-Token", c.csrf)
	response, err := c.http.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 422 {
		t.Fatalf("invalid trailing body: %d", response.StatusCode)
	}
}

func TestServesBuiltAssetsAndSPARoutes(t *testing.T) {
	c := fixture(t)
	for _, path := range []string{"/theme.js", "/projects/example/catalog"} {
		response, err := c.http.Get(c.origin + path)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != 200 || len(data) == 0 {
			t.Fatalf("asset %s: %d %v", path, response.StatusCode, err)
		}
	}
}
