package kubernetes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type fixture struct {
	t                       *testing.T
	mu                      sync.Mutex
	store                   *storage.Store
	exchange                *workerexchange.Service
	provider                *Provider
	driver                  *hostedexecution.Executor
	input                   execution.CodedInput
	server                  *httptest.Server
	value                   *pod
	mode                    string
	posts, deletes, patches int
	unsafePolicy            bool
	deleteReads             int
	deletePending           bool
	replaceAfterDelete      bool
	checkpointError         bool
	token                   string
	auths                   []string
}

type imageInspector struct{ runner string }

func (i imageInspector) InspectImage(_ context.Context, ref string) (hostedexecution.Image, error) {
	return hostedexecution.Image{Reference: ref, Runner: i.runner, Runtime: i.runner + "-fixture-v1", OS: "linux", Architecture: "arm64", Supervisor: "worker-exchange/v1", Project: "project"}, nil
}

func fresh(t *testing.T, runner string, wrappers ...func(http.Handler) http.Handler) *fixture {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "provider.db"))
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := filesystem.New(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	f := &fixture{t: t, store: db, mode: "passed", token: "api-private-token"}
	f.exchange = &workerexchange.Service{Store: db, Blobs: blobs, Authorize: func(context.Context, execution.Reference) (workerstate.Access, error) {
		return workerstate.Access{Deadline: time.Now().Add(time.Minute), ProjectBytes: 1 << 30, RunBytes: 1 << 30}, nil
	}}
	handler := http.Handler(http.HandlerFunc(f.serve))
	for _, wrap := range wrappers {
		handler = wrap(handler)
	}
	f.server = httptest.NewTLSServer(handler)
	t.Cleanup(func() { f.provider.Close(); f.server.Close(); blobs.Close(); db.Close() })
	config := Config{APIOrigin: f.server.URL, Namespace: "studio-workers", ServiceAccount: "runner", Architecture: "arm64", RuntimeClass: "isolated", PolicyName: "isolate", ExchangeOrigin: "https://worker.example.test", ExchangePeer: Peer{Namespace: "private-ingress", Labels: map[string]string{"app": "private-ingress"}}, DNSPeer: Peer{Namespace: "kube-system", Labels: map[string]string{"k8s-app": "kube-dns"}}}
	f.provider, err = New(Options{Config: config, Store: db, Inspector: imageInspector{runner}, Transport: f.server.Client().Transport, Token: func(context.Context) (string, error) { f.mu.Lock(); defer f.mu.Unlock(); return f.token, nil }})
	if err != nil {
		t.Fatal(err)
	}
	f.driver = &hostedexecution.Executor{Provider: f.provider, Exchange: f.exchange, Name: "kubernetes", Runner: runner, ExchangeOrigin: config.ExchangeOrigin, ApproveArtifact: func(context.Context, catalog.Artifact) error { return nil }, PollInterval: time.Millisecond}
	ref := f.driver.Reference(execution.Reference{ID: "attempt-1", Installation: "installation", Project: "project", Run: "run", Test: "TEST-1", Attempt: 1, Worker: "owner"})
	selected := "@TEST-1"
	if runner == "go" {
		selected = "./checks::TestOne"
	}
	f.input = execution.CodedInput{Reference: &ref, RunID: ref.Run, Test: catalog.Test{ID: ref.Test, Runner: runner, Selector: selected}, Artifact: catalog.Artifact{SourceProject: ref.Project, Image: "registry.example.test/teststudio/suite@sha256:" + strings.Repeat("a", 64), Discovered: map[string]string{ref.Test: selected}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000}
	_, changes, err := (workerstate.Ledger{Store: db}).Begin(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	f.input.BeforeDispatch = func() error { return nil }
	f.input.BeforeCleanup = func(report execution.CodedReport) error {
		if f.checkpointError {
			return errors.New("fixture checkpoint unavailable")
		}
		state, err := f.exchange.State(t.Context(), ref)
		if err == nil && !state.Revoked {
			return errors.New("checkpoint preceded revocation")
		}
		ledger := workerstate.Ledger{Store: db}
		entry, row, err := ledger.Get(t.Context(), ref.Run, ref.ID)
		if err != nil {
			return err
		}
		entry.ResultSaved = true
		entry.EvidenceComplete = len(report.EvidenceErrors) == 0
		entry.EvidenceAbandoned = !entry.EvidenceComplete
		entry.RecoveredResult = &report.Result
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, entry)
		if err != nil {
			return err
		}
		return db.Commit(t.Context(), change)
	}
	f.input.AfterCleanup = func(err error) error {
		return (workerstate.Ledger{Store: db}).Finish(t.Context(), ref.Run, ref.ID, "", err)
	}
	return f
}

func (f *fixture) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auths = append(f.auths, r.Header.Get("Authorization"))
	if r.Header.Get("Authorization") != "Bearer "+f.token || r.URL.Query().Get("timeout") != "4s" {
		http.Error(w, "private api credential-canary", 403)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if strings.HasSuffix(r.URL.Path, "/networkpolicies") {
		spec := f.provider.policy()
		if f.unsafePolicy {
			spec["egress"] = []any{map[string]any{}}
		}
		json.NewEncoder(w).Encode(map[string]any{"items": []any{map[string]any{"metadata": map[string]string{"name": "isolate"}, "spec": spec}}})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/serviceaccounts/runner") {
		json.NewEncoder(w).Encode(map[string]any{"metadata": map[string]string{"name": "runner"}, "automountServiceAccountToken": false})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/runtimeclasses/isolated") {
		json.NewEncoder(w).Encode(map[string]any{"metadata": map[string]string{"name": "isolated"}, "handler": "runsc"})
		return
	}
	if r.URL.Path == "/api/v1/namespaces/studio-workers" {
		json.NewEncoder(w).Encode(map[string]any{"metadata": metadata{Name: "studio-workers", Labels: map[string]string{"pod-security.kubernetes.io/enforce": "restricted"}}})
		return
	}
	if r.URL.Path == f.provider.podsPath() && r.Method == "GET" {
		items := []pod{}
		if f.value != nil {
			items = append(items, *f.value)
		}
		json.NewEncoder(w).Encode(map[string]any{"items": items})
		return
	}
	if r.Method == "POST" && r.URL.Path == f.provider.podsPath() {
		f.posts++
		if f.mode == "rejected" {
			http.Error(w, "private admission diagnostic", 422)
			return
		}
		var value pod
		if json.NewDecoder(r.Body).Decode(&value) != nil {
			http.Error(w, "invalid fixture request", 400)
			return
		}
		if f.value != nil {
			w.WriteHeader(409)
			return
		}
		value.Metadata.UID = "pod-1"
		value.Metadata.Version = "1"
		value.Status.Phase = "Running"
		value.Spec["serviceAccount"] = value.Spec["serviceAccountName"]
		value.Spec["nodeName"] = "worker-node"
		value.Spec["schedulerName"] = "default-scheduler"
		value.Spec["priority"] = 0
		value.Spec["preemptionPolicy"] = "PreemptLowerPriority"
		for _, key := range []string{"hostNetwork", "hostPID", "hostIPC"} {
			delete(value.Spec, key)
		}
		delete(value.Spec, "imagePullSecrets")
		f.value = &value
		if f.mode == "injected" {
			containers := value.Spec["containers"].([]any)
			container := containers[0].(map[string]any)
			container["env"] = append(container["env"].([]any), map[string]any{"name": "AWS_ACCESS_KEY_ID", "value": "fixture-injection-canary"})
		}
		if destination := os.Getenv("TESTSTUDIO_KUBERNETES_FIXTURE"); destination != "" && f.mode == "passed" {
			encoded, _ := json.Marshal(value)
			var sanitized pod
			json.Unmarshal(encoded, &sanitized)
			containers := sanitized.Spec["containers"].([]any)
			env := containers[0].(map[string]any)["env"].([]any)
			for _, item := range env {
				entry := item.(map[string]any)
				if entry["name"] == "TESTSTUDIO_JOB_TOKEN" {
					entry["value"] = "scoped-fixture-ticket"
				}
			}
			encoded, _ = json.MarshalIndent(sanitized, "", "  ")
			if !filepath.IsAbs(destination) || os.WriteFile(destination, encoded, 0600) != nil {
				f.t.Error("could not save sanitized Pod contract")
			}
		}
		if f.mode == "lost-ack" {
			http.Error(w, "private response credential-canary", 503)
			return
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(value)
		return
	}
	if f.deletePending && r.Method == "GET" {
		if f.deleteReads == 0 {
			f.value = nil
		} else {
			f.deleteReads--
		}
	}
	if f.value == nil {
		w.WriteHeader(404)
		return
	}
	if r.URL.Path != f.provider.podPath(f.value.Metadata.Name) {
		f.t.Errorf("unexpected API path %s", r.URL.Path)
		w.WriteHeader(404)
		return
	}
	switch r.Method {
	case "GET":
		if f.value.Status.Phase == "Running" && f.mode != "running" {
			state, err := f.exchange.State(r.Context(), *f.input.Reference)
			if err == nil && !state.DispatchPending && !state.Revoked {
				if err := f.publish(r.Context()); err != nil {
					f.t.Error(err)
					w.WriteHeader(500)
					return
				}
				f.value.Status.Phase = "Succeeded"
				setExit(f.value, 0)
			}
		}
		json.NewEncoder(w).Encode(f.value)
	case "PATCH":
		f.patches++
		var patch struct {
			Metadata metadata `json:"metadata"`
			Spec     struct {
				Seconds int `json:"activeDeadlineSeconds"`
			} `json:"spec"`
		}
		if json.NewDecoder(r.Body).Decode(&patch) != nil || patch.Metadata.UID != f.value.Metadata.UID || patch.Metadata.Version != f.value.Metadata.Version || patch.Spec.Seconds != 1 {
			f.t.Error("stop did not protect UID and revision")
			w.WriteHeader(409)
			return
		}
		f.value.Spec["activeDeadlineSeconds"] = 1
		f.value.Metadata.Version = "2"
		f.value.Status.Phase = "Failed"
		setExit(f.value, 137)
		json.NewEncoder(w).Encode(f.value)
	case "DELETE":
		f.deletes++
		var options struct {
			Grace         int                                   `json:"gracePeriodSeconds"`
			Preconditions struct{ UID, ResourceVersion string } `json:"preconditions"`
		}
		if json.NewDecoder(r.Body).Decode(&options) != nil || options.Grace != 0 || options.Preconditions.UID != f.value.Metadata.UID || options.Preconditions.ResourceVersion != f.value.Metadata.Version || f.value.Status.Phase != "Succeeded" && f.value.Status.Phase != "Failed" {
			f.t.Error("active or foreign Pod was deleted")
			w.WriteHeader(409)
			return
		}
		if f.replaceAfterDelete {
			f.value.Metadata.UID = "foreign-replacement"
		} else if f.deleteReads > 0 {
			f.deletePending = true
		} else {
			f.value = nil
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "Success"})
	default:
		w.WriteHeader(405)
	}
}

func setExit(value *pod, code int) {
	data, _ := json.Marshal(map[string]any{"phase": value.Status.Phase, "containerStatuses": []any{map[string]any{"name": "runner", "restartCount": 0, "state": map[string]any{"terminated": map[string]int{"exitCode": code}}}}})
	json.Unmarshal(data, &value.Status)
}

func (f *fixture) publish(ctx context.Context) error {
	var ref execution.Reference
	if json.Unmarshal([]byte(f.value.Metadata.Annotations[ownerAnnotation]), &ref) != nil {
		return errors.New("fixture ownership")
	}
	data, _ := json.Marshal(f.value.Spec["containers"])
	var containers []struct {
		Env []struct{ Name, Value string }
	}
	json.Unmarshal(data, &containers)
	token := ""
	for _, env := range containers[0].Env {
		if env.Name == "TESTSTUDIO_JOB_TOKEN" {
			token = env.Value
		}
	}
	if _, err := f.exchange.Input(ctx, ref.ID, token, "pod-1"); err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := f.exchange.Started(ctx, ref.ID, token, "pod-1", now); err != nil {
		return err
	}
	result := execution.Result{TestID: ref.Test, State: "passed", StartedAt: now, FinishedAt: now.Add(time.Millisecond), DurationMs: 1}
	processExit := 0
	if f.mode == "failed" {
		result.State = "failed"
		result.FailureKind = "assertion"
		processExit = 1
	}
	if f.input.Test.Runner == "go" {
		result.Go = &execution.GoObservation{Package: "./checks", Test: "TestOne", Toolchain: "go-fixture", OS: "linux", Architecture: "arm64"}
	}
	body := []byte("private provider fixture evidence")
	sum := sha256.Sum256(body)
	report := execution.CodedReport{Schema: "teststudio." + f.input.Test.Runner + "/v1", TestID: ref.Test, Selector: f.input.Test.Selector, Result: result, Attachments: []execution.Attachment{{Path: "run.log", Filename: "run.log", Kind: "log", ContentType: "text/plain"}}}
	if err := f.exchange.Submit(ctx, ref.ID, token, "pod-1", workerexchange.Submission{ProcessExitCode: processExit, Report: report, Files: []workerexchange.File{{Path: "run.log", Size: int64(len(body)), SHA256: hex.EncodeToString(sum[:])}}}); err != nil {
		return err
	}
	return f.exchange.Upload(ctx, ref.ID, token, "pod-1", "run.log", bytes.NewReader(body))
}
