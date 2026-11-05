package jobsupervisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	workerhttp "github.com/wtiger001/teststudio/adapters/workerexchange/http"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func TestJobFixtureProcess(t *testing.T) {
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
	arguments := os.Args[index+1:]
	if len(arguments) != 2 {
		os.Exit(20)
	}
	root, mode := arguments[0], arguments[1]
	if mode == "child" {
		for {
			time.Sleep(time.Second)
		}
	}
	if mode == "hang" {
		stopped := make(chan os.Signal, 1)
		signal.Notify(stopped, syscall.SIGTERM)
		child := exec.Command(os.Args[0], "-test.run=^TestJobFixtureProcess$", "--", root, "child")
		if child.Start() != nil {
			os.Exit(21)
		}
		os.WriteFile(filepath.Join(root, "child.pid"), []byte(fmt.Sprint(child.Process.Pid)), 0600)
		os.WriteFile(filepath.Join(root, "runner.pid"), []byte(fmt.Sprint(os.Getpid())), 0600)
		<-stopped
		child.Process.Kill()
		child.Wait()
		os.Exit(0)
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(22)
	}
	var input execution.CodedInput
	if json.Unmarshal(data, &input) != nil {
		os.Exit(23)
	}
	for _, key := range []string{"TESTSTUDIO_JOB_TOKEN", "TESTSTUDIO_GATEWAY_TOKEN", "TESTSTUDIO_DATABASE", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AZURE_CLIENT_SECRET", "IDENTITY_HEADER", "HTTPS_PROXY"} {
		if os.Getenv(key) != "" {
			os.Exit(24)
		}
	}
	os.WriteFile(filepath.Join(root, "executed"), []byte("once"), 0600)
	if path, ok := strings.CutPrefix(mode, "gateway:"); ok {
		transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		}}
		client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
		response, err := client.Get("http://bridge/approved")
		if err != nil {
			os.Exit(27)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		client.CloseIdleConnections()
		if err != nil || response.StatusCode != 200 || string(body) != "reviewed gateway response" {
			os.Exit(28)
		}
	}
	now := time.Now().UTC()
	event, _ := json.Marshal(map[string]any{"event": "test.started", "testId": input.Test.ID, "selector": input.Test.Selector, "at": now})
	fmt.Println("TESTSTUDIO_EVENT " + string(event))
	fmt.Fprintln(os.Stderr, "private runner diagnostic")
	if mode == "missing" {
		os.Exit(2)
	}
	if mode == "flood" {
		fmt.Print(strings.Repeat("drain without blocking ", 160000))
	}
	report := execution.CodedReport{Schema: "teststudio." + input.Test.Runner + "/v1", TestID: input.Test.ID, Selector: input.Test.Selector, Result: execution.Result{TestID: input.Test.ID, State: "passed", StartedAt: now, FinishedAt: now.Add(time.Millisecond), DurationMs: 1}, Attachments: []execution.Attachment{}, EvidenceErrors: []string{}}
	if input.Test.Runner == "go" {
		report.Result.Go = &execution.GoObservation{Package: "./unit", Test: "TestOwned", Toolchain: "go1.26.2", OS: "linux", Architecture: "amd64"}
	}
	exit := 0
	if mode == "assertion" {
		exit = 1
		report.Result.State = "failed"
		report.Result.FailureKind = "assertion"
		report.Result.Error = "owned assertion failed"
	}
	if mode == "false-pass" {
		exit = 1
	}
	if mode == "escape" {
		os.WriteFile(filepath.Join(root, "outside.txt"), []byte("must not enter evidence"), 0600)
		os.Symlink(filepath.Join(root, "outside.txt"), filepath.Join(root, "output", "escape.txt"))
		report.Attachments = append(report.Attachments, execution.Attachment{Path: "escape.txt", Filename: "escape.txt", Kind: "attachment", ContentType: "text/plain"})
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		os.Exit(25)
	}
	if os.WriteFile(filepath.Join(root, "output", "report.json"), encoded, 0600) != nil {
		os.Exit(26)
	}
	os.Exit(exit)
}

type jobFixture struct {
	root       string
	service    *workerexchange.Service
	ref        execution.Reference
	input      execution.CodedInput
	supervisor *Supervisor
	process    *Process
}

func newJob(t *testing.T, runner, mode string) jobFixture {
	t.Helper()
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "output"), 0700)
	db, err := testkit.Open(t, filepath.Join(root, "job.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := filesystem.New(filepath.Join(root, "private"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })
	ref := execution.Reference{ID: "execution", Installation: "installation", Project: "project", Run: "run", Test: "TEST", Worker: "controller", Attempt: 1, Profile: "supervised-fixture"}
	_, changes, err := (workerstate.Ledger{Store: db}).Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().UTC().Add(time.Minute)
	service := &workerexchange.Service{Store: db, Blobs: files, Authorize: func(ctx context.Context, r execution.Reference) (workerstate.Access, error) {
		if r != ref {
			return workerstate.Access{}, workerexchange.ErrDenied
		}
		return workerstate.Access{Deadline: deadline, ProjectBytes: 1 << 30, RunBytes: 100 << 20}, nil
	}}
	selector := "@TEST"
	if runner == "go" {
		selector = "./unit::TestOwned"
	}
	input := execution.CodedInput{RunID: ref.Run, Test: catalog.Test{ID: ref.Test, Runner: runner, Type: "browser", Selector: selector}, Artifact: catalog.Artifact{Image: "registry.example.test/suite@sha256:" + strings.Repeat("a", 64), SourceProject: ref.Project, Discovered: map[string]string{ref.Test: selector}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000}
	if runner == "go" {
		input.Test.Type = "unit"
	}
	ticket, err := service.Prepare(t.Context(), ref, input, deadline)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(workerhttp.Handler(service))
	t.Cleanup(server.Close)
	client, err := workerclient.New(server.URL, ref.ID, "worker-instance", ticket.Token, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	process := &Process{Runner: runner, Directory: root, Output: filepath.Join(root, "output"), Temporary: root, StopGrace: 100 * time.Millisecond, PrivateValues: []string{ticket.Token}, Command: func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, executable, "-test.run=^TestJobFixtureProcess$", "--", root, mode)
	}}
	supervisor := &Supervisor{Transport: client, Runner: process, RetryDelay: time.Millisecond}
	return jobFixture{root, service, ref, input, supervisor, process}
}

func TestSupervisorDeliversGoAndPlaywrightObservationsWithoutAmbientCredentials(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, mode := range []string{"pass", "assertion"} {
			t.Run(runner+"-"+mode, func(t *testing.T) {
				for _, key := range []string{"TESTSTUDIO_JOB_TOKEN", "TESTSTUDIO_DATABASE", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AZURE_CLIENT_SECRET", "IDENTITY_HEADER", "HTTPS_PROXY"} {
					t.Setenv(key, "private-canary")
				}
				f := newJob(t, runner, mode)
				outcome, err := f.supervisor.Run(t.Context())
				if err != nil || !outcome.ReportAccepted || outcome.PendingAttachments != 0 {
					t.Fatal("supervised delivery failed", outcome, err)
				}
				expected := "passed"
				if mode == "assertion" {
					expected = "failed"
					if outcome.ProcessExitCode != 1 {
						t.Fatal("runner failure exit status was discarded")
					}
				}
				if err := f.service.Revoke(t.Context(), f.ref); err != nil {
					t.Fatal(err)
				}
				report, started, complete, err := f.service.Observation(t.Context(), f.ref, f.input, nil)
				if err != nil || !complete || started == nil || report.Result.State != expected || len(report.Attachments) < 1 {
					t.Fatal("owned observation was lost", report.Result, err)
				}
				for _, file := range report.Attachments {
					if bytes.Contains(file.Data, []byte("private-canary")) {
						t.Fatal("ambient credential reached evidence")
					}
				}
				spools, _ := filepath.Glob(filepath.Join(f.root, "teststudio-job-evidence-*"))
				if len(spools) != 0 {
					t.Fatal("supervisor spool was not removed")
				}
			})
		}
	}
}

func TestSupervisorKeepsInfrastructureUnknownAndPrivateLogsForMissingOrInvalidReports(t *testing.T) {
	for _, mode := range []string{"missing", "false-pass"} {
		t.Run(mode, func(t *testing.T) {
			f := newJob(t, "playwright", mode)
			outcome, err := f.supervisor.Run(t.Context())
			if err != nil || !outcome.ReportAccepted {
				t.Fatal(err)
			}
			report, _, complete, err := f.service.Observation(t.Context(), f.ref, f.input, nil)
			if err != nil || !complete || report.Result.State != "unknown" || !report.Result.DurationUnavailable || report.Result.DurationMs != 0 || len(report.Attachments) == 0 {
				t.Fatal("infrastructure failure became a test result", report.Result, err)
			}
		})
	}
}

func TestSupervisorDrainsOversizedOutputAndOmitsEscapeLinks(t *testing.T) {
	for _, mode := range []string{"flood", "escape"} {
		t.Run(mode, func(t *testing.T) {
			f := newJob(t, "playwright", mode)
			outcome, err := f.supervisor.Run(t.Context())
			if err != nil || !outcome.ReportAccepted {
				t.Fatal(err)
			}
			report, _, _, err := f.service.Observation(t.Context(), f.ref, f.input, nil)
			if err != nil || report.Result.State != "passed" || len(report.EvidenceErrors) == 0 {
				t.Fatal("capture problem changed the observation or disappeared", err)
			}
			for _, file := range report.Attachments {
				if len(file.Data) > 2<<20 || bytes.Contains(file.Data, []byte("must not enter evidence")) {
					t.Fatal("unsafe process evidence escaped capture limits")
				}
			}
		})
	}
}

type retryTransport struct {
	Transport
	submissions atomic.Int32
	quota       bool
	failUploads bool
	accepted    bool
}

func (r *retryTransport) Submit(ctx context.Context, s workerprotocol.Submission) error {
	call := r.submissions.Add(1)
	if r.quota && call == 1 {
		return workerprotocol.ErrQuota
	}
	if err := r.Transport.Submit(ctx, s); err != nil {
		return err
	}
	if !r.quota && call == 1 {
		return workerprotocol.ErrUnavailable
	}
	return nil
}
func (r *retryTransport) Upload(ctx context.Context, path string, reader io.Reader) error {
	if r.failUploads {
		return workerprotocol.ErrUnavailable
	}
	return r.Transport.Upload(ctx, path, reader)
}

func TestSupervisorRetriesAcknowledgementsAndPreservesOutcomeWhenEvidenceIsUnavailable(t *testing.T) {
	for _, mode := range []string{"lost-ack", "quota", "upload"} {
		t.Run(mode, func(t *testing.T) {
			f := newJob(t, "playwright", "pass")
			transport := &retryTransport{Transport: f.supervisor.Transport, quota: mode == "quota", failUploads: mode == "upload"}
			f.supervisor.Transport = transport
			outcome, err := f.supervisor.Run(t.Context())
			if err != nil || !outcome.ReportAccepted {
				t.Fatal(err)
			}
			if mode != "lost-ack" && !outcome.EvidenceIncomplete {
				t.Fatal("evidence omission was hidden from the supervisor outcome")
			}
			if mode == "upload" && outcome.PendingAttachments == 0 {
				t.Fatal("missing evidence was hidden")
			}
			report, _, complete, err := f.service.Observation(t.Context(), f.ref, f.input, nil)
			if err != nil || report.Result.State != "passed" {
				t.Fatal("evidence changed the observed outcome", err)
			}
			if mode == "quota" && (len(report.Attachments) != 0 || len(report.EvidenceErrors) == 0) {
				t.Fatal("quota omission was not explicit")
			}
			if mode == "upload" && complete {
				t.Fatal("unuploaded attachments appeared complete")
			}
		})
	}
}

func TestSupervisorCancellationStopsAndReapsTheOwnedRunnerGroup(t *testing.T) {
	f := newJob(t, "playwright", "hang")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := f.supervisor.Run(ctx); done <- err }()
	deadline := time.Now().Add(3 * time.Second)
	var pid int
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(filepath.Join(f.root, "child.pid"))
		if err == nil {
			fmt.Sscan(string(data), &pid)
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if pid == 0 {
		cancel()
		<-done
		t.Fatal("fixture child did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled job reported successful delivery")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("owned runner group did not stop")
	}
	process, _ := os.FindProcess(pid)
	if err := process.Signal(syscall.Signal(0)); err == nil {
		t.Fatal("owned descendant remained alive")
	}
}

func TestSupervisorRejectsStaleReportsBeforeLaunchingAnotherAttempt(t *testing.T) {
	f := newJob(t, "playwright", "pass")
	if err := os.WriteFile(filepath.Join(f.root, "output", "report.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.supervisor.Run(t.Context()); err == nil {
		t.Fatal("stale output was accepted for another attempt")
	}
	if _, err := os.Stat(filepath.Join(f.root, "executed")); !os.IsNotExist(err) {
		t.Fatal("stale output did not block dispatch")
	}
}
