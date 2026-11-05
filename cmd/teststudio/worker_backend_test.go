package main

import (
	"encoding/json"
	"flag"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func TestWorkerBackendRejectsUnrelatedConfigurationBeforeStartup(t *testing.T) {
	for _, option := range []string{"local", "local-accounts", "oidc-config", "http-trust-bundles", "scheduler", "notification-worker", "execution-worker", "prepared-sources", "source-bundles", "github-sources", "ecs-workers-env", "kubernetes-workers-env", "recording-cli", "go-images", "playwright-images", "addr", "ui", "artifact-addr", "public-url", "trace-viewer-directory"} {
		flags := flag.NewFlagSet("worker", flag.ContinueOnError)
		flags.String(option, "", "")
		if err := flags.Parse([]string{"--" + option + "=private-configuration-canary"}); err != nil {
			t.Fatal(err)
		}
		err := validateWorkerBackendFlags(flags, "127.0.0.1:8042", "azure")
		if err == nil || strings.Contains(err.Error(), "private-configuration-canary") {
			t.Fatal("unrelated configuration accepted or disclosed", option, err)
		}
	}
	flags := flag.NewFlagSet("worker", flag.ContinueOnError)
	for _, option := range []string{"serve-role", "worker-addr", "execution-location", "database-env", "artifact-store-env"} {
		flags.String(option, "", "")
	}
	if err := flags.Parse([]string{"--serve-role=worker", "--worker-addr=:8042", "--execution-location=azure", "--database-env=DATABASE", "--artifact-store-env=ARTIFACTS"}); err != nil {
		t.Fatal(err)
	}
	if err := validateWorkerBackendFlags(flags, ":8042", "azure"); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{"", "8042", ":http", ":-1", ":65536"} {
		if validateWorkerBackendFlags(flags, address, "azure") == nil {
			t.Fatal("invalid listener accepted")
		}
	}
	if validateWorkerBackendFlags(flags, ":8042", "invalid/location") == nil {
		t.Fatal("invalid location accepted")
	}
}

type backendFixture struct {
	db       *storage.Store
	auth     *authorization.Service
	exchange *workerexchange.Service
	database string
	files    string
}

func newBackendFixture(t *testing.T) backendFixture {
	t.Helper()
	root := t.TempDir()
	database, files := filepath.Join(root, "backend.db"), filepath.Join(root, "artifacts")
	db, auth, closeStorage, err := openServiceStorage(t.Context(), database, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeStorage)
	blobs, err := filesystem.New(files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close() })
	if err := auth.Directory.SaveUser(t.Context(), identity.User{ID: "owner", Name: "Backend fixture owner"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "owner"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	f := backendFixture{db, auth, workerBackendExchange(db, auth, blobs, "azure"), database, files}
	f.put(t, "projects", "project", projects.Project{ID: "project", Name: "Backend fixture project"})
	f.put(t, projects.Bucket("suites", "project"), "suite", projects.Suite{ID: "suite", Name: "Targetless checks"})
	return f
}

func (f backendFixture) put(t *testing.T, bucket, id string, value any) {
	t.Helper()
	var revision int64
	if row, err := f.db.Get(t.Context(), bucket, id); err == nil {
		revision = row.Revision
	}
	change, err := storage.Put(bucket, id, "project", revision, value)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
}

func (f backendFixture) executionTicket(t *testing.T) workerexchange.Ticket {
	t.Helper()
	now, deadline := time.Now().UTC(), time.Now().UTC().Add(time.Minute)
	test := catalog.Test{ID: "PAGE-1", Runner: "playwright", Type: "browser", Selector: "@PAGE-1", State: "enabled", Safety: "safe"}
	f.put(t, catalog.Bucket("tests", "project", "suite"), test.ID, test)
	ref := execution.Reference{ID: "page-attempt", Installation: "backend-installation", Project: "project", Run: "page-run", Test: test.ID, Attempt: 1, Worker: "controller", Profile: "azure-playwright-v1", Container: "page-job"}
	run := runs.Run{ID: ref.Run, Project: ref.Project, Actor: "owner", Request: runs.Request{Suite: "suite", Location: "azure"}, State: "running", Worker: ref.Worker, CurrentTest: ref.Test, CurrentAttempt: 1, LeaseUntil: &deadline, Deadline: deadline, Tests: []catalog.Test{test}, Capture: capture.DefaultPolicy(), CreatedAt: now}
	f.put(t, "runs", ref.Run, run)
	_, changes, err := (workerstate.Ledger{Store: f.db}).Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	input := execution.CodedInput{RunID: ref.Run, Test: test, Artifact: catalog.Artifact{Image: "registry.example.test/page@sha256:" + strings.Repeat("c", 64), Discovered: map[string]string{test.ID: test.Selector}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000}
	ticket, err := f.exchange.Prepare(t.Context(), ref, input, deadline)
	if err != nil {
		t.Fatal(err)
	}
	return ticket
}

func TestWorkerBackendUsesLiveExecutionAuthorityAndOmitsUserRoutes(t *testing.T) {
	f := newBackendFixture(t)
	ticket := f.executionTicket(t)
	server := httptest.NewServer(workerBackendServer("", f.db, f.exchange).Handler)
	t.Cleanup(server.Close)
	call := func(path, token string, want int) {
		t.Helper()
		req, _ := http.NewRequest("GET", server.URL+path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-TestStudio-Worker-Instance", "page-instance")
		}
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != want {
			t.Fatal(path, response.StatusCode, want)
		}
	}
	path := "/internal/worker/v1/executions/" + ticket.Execution + "/input"
	call(path, "", 404)
	call(path, ticket.Token, 200)
	for _, path := range []string{"/", "/api/v1/projects", "/auth/login", "/access/report", "/internal/worker/v1/prepare"} {
		call(path, ticket.Token, 404)
	}
	row, err := f.db.Get(t.Context(), "users", "owner")
	if err != nil {
		t.Fatal(err)
	}
	user, _ := storage.Decode[identity.User](row)
	user.Disabled = true
	if err := f.auth.Directory.SaveUser(t.Context(), user, row.Revision); err != nil {
		t.Fatal(err)
	}
	call(path, ticket.Token, 404)
}

func TestWorkerBackendPreservesDiscoveryActivationAndLocationChecks(t *testing.T) {
	f := newBackendFixture(t)
	now, until := time.Now().UTC(), time.Now().UTC().Add(time.Minute)
	ref := execution.Reference{ID: "discovery-attempt", Installation: "preparation-backend", Project: "project", Run: "discovery-job", Test: "catalog", Attempt: 1, Worker: "preparer", Profile: "azure-go-discovery-v1", Container: "discovery-container"}
	job := sources.Job{ID: ref.Run, Project: ref.Project, Suite: "suite", Actor: "owner", Runner: "go", Location: "azure", State: "discovering", Worker: ref.Worker, LeaseUntil: &until, StartedAt: &now, Reference: &ref}
	f.put(t, sources.JobBucket("project", "suite"), job.ID, job)
	f.put(t, "preparation-leases/azure", job.ID, map[string]string{"jobId": job.ID, "suiteId": job.Suite, "executionLocation": "azure"})
	input := workerprotocol.DiscoveryInput{Project: "project", Suite: "suite", Runner: "go", Image: "registry.example.test/check@sha256:" + strings.Repeat("d", 64), SourceDigest: "sha256:" + strings.Repeat("e", 64)}
	ticket, err := f.exchange.PrepareDiscovery(t.Context(), ref, input, until)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.exchange.Input(t.Context(), ref.ID, ticket.Token, "discoverer"); err != workerexchange.ErrUnavailable {
		t.Fatal("discovery bypassed dispatch activation", err)
	}
	if err := f.exchange.Activate(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	packet, err := f.exchange.Input(t.Context(), ref.ID, ticket.Token, "discoverer")
	if err != nil || packet.Operation != "discover" {
		t.Fatal("discovery authorization was not wired", err)
	}
	other := workerBackendExchange(f.db, f.auth, f.exchange.Blobs, "other-location")
	if _, err := other.Input(t.Context(), ref.ID, ticket.Token, "discoverer"); err != workerexchange.ErrDenied {
		t.Fatal("discovery crossed its location", err)
	}
	job.CancelRequested = true
	f.put(t, sources.JobBucket("project", "suite"), job.ID, job)
	if _, err := f.exchange.Input(t.Context(), ref.ID, ticket.Token, "discoverer"); err != workerexchange.ErrDenied {
		t.Fatal("cancelled discovery retained access", err)
	}
}

func TestWorkerBackendReadinessFailsWithDatabaseWhileLivenessRemains(t *testing.T) {
	f := newBackendFixture(t)
	handler := workerBackendServer("", f.db, f.exchange).Handler
	check := func(path string, want int) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if want == 200 {
			var state map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &state); err != nil || state["role"] != "worker" {
				t.Fatal("worker probe lost its dedicated role", err)
			}
		}
		if response.Code != want || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(path, response.Code)
		}
	}
	check("/healthz", 200)
	check("/livez", 200)
	f.db.Close()
	check("/healthz", 503)
	check("/livez", 200)
}

func TestWorkerBackendStartupPreservesSignInStateAndStopsItsListener(t *testing.T) {
	f := newBackendFixture(t)
	f.put(t, "local-auth-mode", "installation", map[string]bool{"enabled": true})
	before, _ := f.db.Get(t.Context(), "local-auth-mode", "installation")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	startOwnedServiceCommand(t, address, "--serve-role=worker", "--worker-addr="+address, "--execution-location=azure", "--database="+f.database, "--artifact-directory="+f.files)
	client := &http.Client{Timeout: time.Second}
	t.Cleanup(client.CloseIdleConnections)
	ready := false
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		response, err := client.Get("http://" + address + "/healthz")
		if err == nil {
			var state map[string]string
			err = json.NewDecoder(io.LimitReader(response.Body, 1024)).Decode(&state)
			response.Body.Close()
			ready = err == nil && response.StatusCode == 200 && state["service"] == "teststudio" && state["role"] == "worker"
		}
		if ready {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !ready {
		t.Fatal("backend never became ready")
	}
	after, err := f.db.Get(t.Context(), "local-auth-mode", "installation")
	if err != nil || after.Revision != before.Revision || string(after.Data) != string(before.Data) {
		t.Fatal("worker startup changed sign-in configuration", err)
	}
	for _, bucket := range []string{"providers", "sessions", "runs", "source-jobs", "notification-deliveries"} {
		rows, err := f.db.All(t.Context(), bucket, "")
		if err != nil || len(rows) != 0 {
			t.Fatal("worker startup bootstrapped application state", bucket, err)
		}
	}
}
