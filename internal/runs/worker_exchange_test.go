package runs_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	workerhttp "github.com/wtiger001/teststudio/adapters/workerexchange/http"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestWorkerTransportUsesLiveRunOwnershipAndRetainsAnAcknowledgementLostReport(t *testing.T) {
	s, _, _ := setup(t, nil)
	run, ref := lostExecution(t, s, recoveredDriver("passed"), true)
	if _, err := s.AuthorizeWorkerAttempt(t.Context(), ref); err != nil {
		t.Fatal("live attempt was denied", err)
	}
	input := execution.CodedInput{RunID: run.ID, Test: catalog.Test{ID: ref.Test, Type: "browser", Runner: "playwright", Selector: "@HEALTH-1"}, Artifact: catalog.Artifact{Image: "registry.example.test/suite@sha256:" + strings.Repeat("a", 64), SourceProject: ref.Project, Discovered: map[string]string{ref.Test: "@HEALTH-1"}}, Capture: capture.DefaultPolicy(), TimeoutMs: 1000}
	service := &workerexchange.Service{Store: s.Store, Blobs: s.Evidence.Blobs, Authorize: s.AuthorizeWorkerAttempt}
	deadline := time.Now().UTC().Add(20 * time.Second)
	ticket, err := service.Prepare(t.Context(), ref, input, deadline)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(workerhttp.Handler(service))
	t.Cleanup(server.Close)
	transport := &lostReportAcknowledgement{RoundTripper: server.Client().Transport}
	client, err := workerclient.New(server.URL, ref.ID, "owned-worker", ticket.Token, transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	packet, err := client.Input(t.Context())
	if err != nil || packet.Execution != ref.ID {
		t.Fatal("private work fetch failed", err)
	}
	now := time.Now().UTC()
	if err := client.Started(t.Context(), now); err != nil {
		t.Fatal(err)
	}
	report := workerexchange.Submission{Report: execution.CodedReport{Schema: "teststudio.playwright/v1", TestID: ref.Test, Selector: input.Test.Selector, Result: execution.Result{TestID: ref.Test, State: "passed", StartedAt: now, FinishedAt: now.Add(time.Millisecond), DurationMs: 1}}}
	body := []byte("bounded private HTTP attachment")
	hash := sha256.Sum256(body)
	report.Report.Attachments = []execution.Attachment{{Path: "run log.txt", Kind: "log", Filename: "run log.txt", ContentType: "text/plain"}}
	report.Files = []workerexchange.File{{Path: "run log.txt", Size: int64(len(body)), SHA256: hex.EncodeToString(hash[:])}}
	if err := client.Submit(t.Context(), report); !errors.Is(err, workerexchange.ErrUnavailable) || strings.Contains(err.Error(), ticket.Token) {
		t.Fatal("lost acknowledgement exposed private transport details", err)
	}
	if err := client.Submit(t.Context(), report); err != nil {
		t.Fatal("exact report retry failed", err)
	}
	for range 2 {
		if err := client.Upload(t.Context(), "run log.txt", bytes.NewReader(body)); err != nil {
			t.Fatal("private attachment upload or exact replay failed", err)
		}
	}
	observation, _, complete, err := service.Observation(t.Context(), ref, input, nil)
	if err != nil || !complete || observation.Result.State != "passed" || len(observation.Attachments) != 1 || !bytes.Equal(observation.Attachments[0].Data, body) {
		t.Fatal("accepted report was lost", err)
	}
	for _, header := range []string{"Cookie", "Origin", "Query"} {
		address := server.URL + "/internal/worker/v1/executions/" + ref.ID + "/input"
		if header == "Query" {
			address += "?ticket=" + ticket.Token
		}
		request, _ := http.NewRequest("GET", address, nil)
		request.Header.Set("Authorization", "Bearer "+ticket.Token)
		request.Header.Set("X-TestStudio-Worker-Instance", "owned-worker")
		if header == "Cookie" {
			request.Header.Set("Cookie", "session=unrelated")
		}
		if header == "Origin" {
			request.Header.Set("Origin", "https://studio.example.test")
		}
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 404 || response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("browser or query credentials entered private transport", header, response.StatusCode)
		}
	}
	if _, err := s.Cancel(t.Context(), "admin", run.Project, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Input(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("cancelled run retained worker access", err)
	}
	if err := client.Submit(t.Context(), report); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("cancelled run retained write access", err)
	}
}

type lostReportAcknowledgement struct {
	http.RoundTripper
	lost atomic.Bool
}

func (t *lostReportAcknowledgement) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.RoundTripper.RoundTrip(request)
	if err == nil && strings.HasSuffix(request.URL.Path, "/report") && response.StatusCode == 204 && t.lost.CompareAndSwap(false, true) {
		response.Body.Close()
		return nil, errors.New("private transport diagnostic " + request.Header.Get("Authorization"))
	}
	return response, err
}

func TestWorkerAuthorityRejectsStaleOwnersAttemptsLeasesAndArchivedProjects(t *testing.T) {
	s, _, _ := setup(t, nil)
	run, ref := lostExecution(t, s, recoveredDriver("passed"), true)
	for _, change := range []func(*runs.Run){
		func(v *runs.Run) { v.Worker = "other-controller" },
		func(v *runs.Run) { v.CurrentTest = "OTHER" },
		func(v *runs.Run) { v.CurrentAttempt = 2 },
		func(v *runs.Run) { v.State = "passed" },
		func(v *runs.Run) { v.CancelRequested = true },
		func(v *runs.Run) { expired := time.Now().Add(-time.Second); v.LeaseUntil = &expired },
		func(v *runs.Run) { v.Deadline = time.Now().Add(-time.Second) },
	} {
		value := run
		change(&value)
		_, row, err := s.Get(t.Context(), run.Project, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		update, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
		if err := s.Store.Commit(t.Context(), update); err != nil {
			t.Fatal(err)
		}
		if _, err := s.AuthorizeWorkerAttempt(t.Context(), ref); err == nil {
			t.Fatal("stale worker authority remained valid")
		}
	}
	_, row, _ := s.Get(t.Context(), run.Project, run.ID)
	update, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, run)
	if err := s.Store.Commit(t.Context(), update); err != nil {
		t.Fatal(err)
	}
	projectRow, err := s.Store.Get(t.Context(), "projects", run.Project)
	if err != nil {
		t.Fatal(err)
	}
	var project map[string]any
	if err := json.Unmarshal(projectRow.Data, &project); err != nil {
		t.Fatal(err)
	}
	project["archived"] = true
	update, _ = storage.Put(projectRow.Bucket, projectRow.ID, projectRow.Project, projectRow.Revision, project)
	if err := s.Store.Commit(t.Context(), update); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthorizeWorkerAttempt(t.Context(), ref); err == nil {
		t.Fatal("archived project retained worker authority")
	}
}

func TestWorkerClientRejectsRedirectsAndInsecureOriginsWithoutLeakingTickets(t *testing.T) {
	var visits atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { visits.Add(1) }))
	t.Cleanup(target.Close)
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(origin.Close)
	token := strings.Repeat("x", 43)
	if _, err := workerclient.New("http://127.0.0.1", "execution", "worker", token, nil); err == nil {
		t.Fatal("insecure transport was accepted")
	}
	client, err := workerclient.New(origin.URL, "execution", "worker", token, origin.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	if _, err := client.Input(context.Background()); err == nil || strings.Contains(err.Error(), token) {
		t.Fatal("redirect exposed transport details", err)
	}
	if visits.Load() != 0 {
		t.Fatal("worker ticket followed an origin redirect")
	}
}
