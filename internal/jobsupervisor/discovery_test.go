package jobsupervisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	workerhttp "github.com/wtiger001/teststudio/adapters/workerexchange/http"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func TestDiscoveryFixtureProcess(t *testing.T) {
	index := -1
	for position, value := range os.Args {
		if value == "--" {
			index = position
			break
		}
	}
	if index < 0 {
		return
	}
	args := os.Args[index+1:]
	if len(args) != 3 {
		os.Exit(20)
	}
	root, runner, mode := args[0], args[1], args[2]
	data, _ := io.ReadAll(os.Stdin)
	if len(data) != 0 {
		os.Exit(21)
	}
	for _, key := range []string{"TESTSTUDIO_JOB_TOKEN", "TESTSTUDIO_DATABASE", "AWS_ACCESS_KEY_ID", "AZURE_CLIENT_SECRET"} {
		if os.Getenv(key) != "" {
			os.Exit(22)
		}
	}
	manifest := suitespec.Manifest{Schema: 1, Suite: "checks", Runner: runner, Type: "unit", ModuleRoot: ".", Catalog: "catalog.json", BuildProfile: "go-module", Tests: suitespec.Tests{Packages: []string{"./unit"}}}
	test := catalog.Test{ID: "DISC-1", Title: "Prepared arithmetic", Category: "Unit", Type: "unit", Runner: runner, State: "enabled", Safety: "safe", Rationale: "In-memory arithmetic only.", TimeoutSeconds: 5, Source: "unit/discover_test.go", Selector: "./unit::TestPrepared"}
	if runner == "playwright" {
		manifest.Type, manifest.BuildProfile, manifest.Tests = "browser", "playwright-node", suitespec.Tests{Config: "playwright.config.ts"}
		test.Type, test.Source, test.Selector = "browser", "checks/discover.spec.ts", "@DISC-1"
	}
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{test}}
	origin := suitespec.Origin{SnapshotDigest: "sha256:" + strings.Repeat("c", 64)}
	var report any = suitespec.Discovery{Schema: suitespec.DiscoverySchema, Manifest: manifest, Catalog: document, Origin: origin, Tests: []suitespec.DiscoveredTest{{ID: test.ID, Selector: test.Selector, File: test.Source, Line: 1}}}
	if runner == "playwright" {
		catalogJSON, _ := json.Marshal(document)
		report = suitespec.PlaywrightDiscovery{Schema: suitespec.DiscoverySchema, Manifest: manifest, Origin: origin, CatalogText: string(catalogJSON), Tests: []suitespec.ListedTest{{Tags: []string{test.Selector}, File: test.Source, Line: 1}}}
	}
	data, _ = json.Marshal(report)
	if mode == "malformed" {
		data = []byte("invalid discovery")
	}
	if mode == "extra-output" {
		os.WriteFile(filepath.Join(root, "unexpected.txt"), []byte("unapproved output"), 0600)
	}
	os.WriteFile(filepath.Join(root, "discovery.json"), data, 0600)
	fmt.Println("Owned discovery diagnostic worker-token-canary")
	os.Exit(0)
}

func TestHostedDiscoverySupervisorUsesSeparateGatedReportsAndNoTestResult(t *testing.T) {
	t.Setenv("TESTSTUDIO_JOB_TOKEN", "must-not-reach-source")
	t.Setenv("AWS_ACCESS_KEY_ID", "must-not-reach-source")
	for _, item := range []struct{ runner, mode string }{{"go", "ready"}, {"playwright", "ready"}, {"go", "malformed"}, {"go", "extra-output"}} {
		t.Run(item.runner+"-"+item.mode, func(t *testing.T) {
			store, err := testkit.Open(t, filepath.Join(t.TempDir(), "discovery.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			deadline := time.Now().UTC().Add(time.Minute)
			checkpointed := false
			exchange := &workerexchange.Service{Store: store, Authorize: func(context.Context, execution.Reference) (workerstate.Access, error) {
				return workerstate.Access{Deadline: deadline}, nil
			}, AuthorizeDiscoveryCleanup: func(context.Context, execution.Reference) error {
				if !checkpointed {
					return storage.ErrConflict
				}
				return nil
			}}
			ref := execution.Reference{ID: "discovery", Installation: "preparation-installation", Project: "project", Run: "source-job", Test: "catalog", Attempt: 1, Worker: "preparer", Profile: item.runner + "-fixture-discovery-v1"}
			input := workerprotocol.DiscoveryInput{Project: ref.Project, Suite: "checks", Runner: item.runner, Image: "registry.fixture.test/source@sha256:" + strings.Repeat("a", 64), SourceDigest: "sha256:" + strings.Repeat("c", 64)}
			ticket, err := exchange.PrepareDiscovery(t.Context(), ref, input, deadline)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewTLSServer(workerhttp.Handler(exchange))
			defer server.Close()
			client, err := workerclient.New(server.URL, ref.ID, "pod", ticket.Token, server.Client().Transport)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if _, err := client.Input(t.Context()); !errors.Is(err, workerprotocol.ErrUnavailable) {
				t.Fatal("unverified dispatch exposed discovery input", err)
			}
			if err := exchange.Activate(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			output := t.TempDir()
			process := &Process{Runner: item.runner, Directory: output, Output: output, PrivateValues: []string{"worker-token-canary"}, Command: func(ctx context.Context) *exec.Cmd {
				return exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDiscoveryFixtureProcess$", "--", output, item.runner, item.mode)
			}}
			supervisor := &Supervisor{Operation: "discover", Transport: client, Runner: process, Attempts: 2, RetryDelay: time.Millisecond}
			outcome, err := supervisor.Run(t.Context())
			if err != nil || !outcome.ReportAccepted {
				t.Fatal(outcome, err)
			}
			submission, found, err := exchange.DiscoveryObservation(t.Context(), ref, input)
			if err != nil || !found || submission.Discovery == nil || submission.Report.TestID != "" || len(submission.Files) != 0 || strings.Contains(submission.Discovery.Log, "worker-token-canary") {
				t.Fatal("discovery created a test result or exposed private data", err)
			}
			if (item.mode == "ready") != (submission.Discovery.Failure == "") {
				t.Fatal("discovery failure was represented as success", submission.Discovery)
			}
			if err := client.Submit(t.Context(), submission); err != nil {
				t.Fatal("report retry changed its discovery observation", err)
			}
			if err := exchange.Revoke(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if err := exchange.CheckCleanup(t.Context(), ref); err == nil {
				t.Fatal("discovery cleanup bypassed its preparation checkpoint")
			}
			checkpointed = true
			if err := exchange.Cleanup(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Get(t.Context(), workerexchange.Bucket, ref.ID); !errors.Is(err, storage.ErrNotFound) {
				t.Fatal("owned discovery transport remained after cleanup", err)
			}
		})
	}
}

func TestDiscoveryProcessSelectsOnlyTheFixedGoDiscoveryOperation(t *testing.T) {
	command, err := (&Process{Runner: "go"}).command(t.Context(), true)
	if err != nil || len(command.Args) != 2 || command.Args[0] != "/runner/teststudio-go" || command.Args[1] != "--discover" {
		t.Fatal(command, err)
	}
}
