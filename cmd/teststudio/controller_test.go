package main

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"flag"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httptrust "github.com/wtiger001/teststudio/adapters/httptrust/filesystem"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/cliauth"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestControllerRejectsApplicationAndWorkerFlagsBeforeCreatingState(t *testing.T) {
	for _, option := range []string{"local", "local-accounts", "recover-local-admin", "oidc-config", "bootstrap-subject", "worker-addr", "prepared-sources", "source-bundles", "go-images", "recording-cli", "recording-worker-config", "recording-worker-config-env", "recording-embedded-cli", "recording-embedded-helper", "recording-companion", "artifact-addr", "ui", "scheduler", "execution-worker"} {
		t.Run(option, func(t *testing.T) {
			originalFlags, originalArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
			flag.CommandLine = flag.NewFlagSet("owned-controller-check", flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			database := filepath.Join(t.TempDir(), "must-not-exist.db")
			value := "private-canary"
			if option == "recording-companion" || option == "local" || option == "local-accounts" || option == "recover-local-admin" || option == "prepared-sources" || option == "source-bundles" || option == "scheduler" || option == "execution-worker" {
				value = "true"
			}
			os.Args = []string{"teststudio", "--serve-role=controller", "--addr=127.0.0.1:0", "--database=" + database, "--" + option + "=" + value}
			err := run()
			if err == nil || strings.Contains(err.Error(), "private-canary") {
				t.Fatal("controller accepted unrelated flags or disclosed values", err)
			}
			if _, err := os.Stat(database); !os.IsNotExist(err) {
				t.Fatal("invalid controller role created state", err)
			}
		})
	}
	flags := flag.NewFlagSet("controller", flag.ContinueOnError)
	flags.String("addr", "", "")
	flags.Parse([]string{"--addr=127.0.0.1:0"})
	if err := validateControllerFlags(flags, "127.0.0.1:0", "local", false, false); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []struct {
		address, location string
		ecs, kube         bool
	}{{"invalid", "local", false, false}, {":65536", "local", false, false}, {":8043", "azure-container-apps", false, false}, {":8043", "local", true, false}, {":8043", "ecs", true, true}} {
		if validateControllerFlags(flags, mode.address, mode.location, mode.ecs, mode.kube) == nil {
			t.Fatal("invalid controller placement or listener accepted", mode)
		}
	}
	if validateControllerFlags(flag.NewFlagSet("missing-address", flag.ContinueOnError), "127.0.0.1:8040", "local", false, false) == nil {
		t.Fatal("implicit controller listener accepted")
	}
}

func TestControllerHealthOmitsApplicationAndExchangeRoutes(t *testing.T) {
	f := newBackendFixture(t)
	handler := serviceHealthHandler(f.db, "controller")
	for _, path := range []string{"/", "/api/projects", "/auth/login", "/internal/worker/claim", "/artifacts/anything"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 404 {
			t.Fatal("controller exposed another service route", path, response.Code)
		}
	}
	check := func(path string, want int) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != want || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("controller health classification changed", path, response.Code)
		}
	}
	check("/healthz", 200)
	check("/livez", 200)
	f.db.Close()
	check("/healthz", 503)
	check("/livez", 200)
}

func TestControllerCommandExecutesQueuedHTTPSWhileAPIWorkerIsDisabled(t *testing.T) {
	f := newBackendFixture(t)
	var calls atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"status":"up"}`)
	}))
	t.Cleanup(target.Close)
	root := t.TempDir()
	ca := filepath.Join(root, "probe-ca.pem")
	trustFile := filepath.Join(root, "trust.json")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: target.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal([]httptrust.Binding{{ID: "owned-ca", Name: "Owned probe trust", Project: "project", Origin: target.URL, File: ca}})
	if err := os.WriteFile(trustFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	trust, err := httptrust.Load(trustFile)
	if err != nil {
		t.Fatal(err)
	}
	f.put(t, "local-auth-mode", "installation", map[string]bool{"enabled": true})
	before, _ := f.db.Get(t.Context(), "local-auth-mode", "installation")
	f.put(t, projects.Bucket("applications", "project"), "app", projects.Application{ID: "app", Name: "Owned application"})
	f.put(t, projects.Bucket("suites", "project"), "suite", projects.Suite{ID: "suite", Name: "Owned checks", Applications: []string{"app"}, RequiresTarget: true})
	f.put(t, projects.Bucket("environments", "project"), "local", projects.Environment{ID: "local", Name: "Owned environment", Application: "app", Stage: "local", AllowPrivate: true, Endpoints: map[string]string{"api": target.URL}})
	run, source, _ := controllerServices(f.db, f.auth, nil, "local", "https://studio.example.test", trust)
	test := catalog.Test{ID: "HEALTH-1", Title: "Owned HTTPS probe", Application: "app", Category: "Availability", Type: "https", Runner: "http", State: "enabled", Safety: "safe", Rationale: "Reads the owned loopback service", TimeoutSeconds: 5, Check: &catalog.HTTPCheck{Endpoint: "api", Path: "/healthz", Method: "GET", StatusCodes: []int{200}}}
	if err := run.Catalog.SaveTest(t.Context(), "owner", "project", "suite", test, 0); err != nil {
		t.Fatal(err)
	}
	draft, err := run.Catalog.CreateDraft(t.Context(), "owner", "project", "suite", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.Catalog.Publish(t.Context(), "owner", "project", "suite", draft.ID, "owned-release", 1, true); err != nil {
		t.Fatal(err)
	}
	f.put(t, "notification-registry", "project", map[string]any{"ids": []string{"owned-channel"}})
	f.put(t, notifications.Channels("project"), "owned-channel", notifications.Channel{Project: "project", Owner: "owner", Definition: notifications.Definition{ID: "owned-channel", Name: "Owned completion", Enabled: true, Kind: "webhook", Reference: "owned-reference", Outcomes: []string{"passed"}, MaxAttempts: 3}})
	queued, err := run.Create(t.Context(), "owner", "project", runs.Request{Suite: "suite", Environment: "local", Location: "local"}, "controller-probe", "manual")
	if err != nil {
		t.Fatal(err)
	}
	run.PollInterval = 5 * time.Millisecond
	background := backgroundServices{DisableExecution: true, Runs: run, Recordings: &recording.Service{Store: f.db, Auth: f.auth}, CLI: &cliauth.Service{Sessions: &sessions.Service{Store: f.db}}, Evidence: run.Evidence, Sources: source}
	stopAPI := background.start(t.Context())
	time.Sleep(25 * time.Millisecond)
	stopAPI()
	pending, _, err := run.Get(t.Context(), "project", queued.ID)
	if err != nil || pending.State != "requested" || calls.Load() != 0 {
		t.Fatal("disabled API worker claimed the queued run", pending.State, err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	stop := startOwnedServiceCommand(t, address, "--serve-role=controller", "--addr="+address, "--execution-location=local", "--database="+f.database, "--artifact-directory="+f.files, "--public-url=https://studio.example.test", "--http-trust-bundles="+trustFile)
	client := &http.Client{Timeout: time.Second}
	t.Cleanup(client.CloseIdleConnections)
	deadline := time.Now().Add(5 * time.Second)
	var finished runs.Run
	for time.Now().Before(deadline) {
		finished, _, err = run.Get(t.Context(), "project", queued.ID)
		if err == nil && finished.State == "finished" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || finished.Outcome != "passed" || finished.Counts.Total != 1 || finished.Counts.Passed != 1 || calls.Load() != 1 {
		t.Fatal("headless controller did not complete the logical HTTPS run", finished.State, finished.Error, err)
	}
	response, err := client.Get("http://" + address + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	var health map[string]string
	json.NewDecoder(response.Body).Decode(&health)
	response.Body.Close()
	if response.StatusCode != 200 || health["role"] != "controller" {
		t.Fatal("controller health role changed")
	}
	after, err := f.db.Get(t.Context(), "local-auth-mode", "installation")
	if err != nil || after.Revision != before.Revision || string(after.Data) != string(before.Data) {
		t.Fatal("controller startup changed sign-in configuration", err)
	}
	deliveries, err := f.db.All(t.Context(), notifications.Deliveries("project"), "")
	if err != nil || len(deliveries) != 1 {
		t.Fatal("controller omitted completion notifications", err, len(deliveries))
	}
	delivery, err := storage.Decode[notifications.Delivery](deliveries[0])
	if err != nil || delivery.Message.URL != "https://studio.example.test/projects/project/runs/"+queued.ID || delivery.Attempts != 0 {
		t.Fatal("controller delivered notifications or changed report origin", err)
	}
	stop()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := f.db.DB.PingContext(ctx); err != nil {
		t.Fatal("controller shutdown closed the caller's database", err)
	}
}
