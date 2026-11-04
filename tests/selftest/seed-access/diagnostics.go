package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type diagnosticUploadFailure struct{ artifacts.Store }

func (s diagnosticUploadFailure) Put(context.Context, string, io.Reader, int64) (artifacts.Object, error) {
	return artifacts.Object{}, errors.New("Fixture upload unavailable")
}

func prepareDiagnostics(ctx context.Context, db *storage.Store, auth *authorization.Service, project string) error {
	media := filepath.Join("/tmp", "teststudio-diagnostics-"+project)
	screenshot, err := os.ReadFile(filepath.Join(media, "screenshot.png"))
	if err != nil {
		return err
	}
	trace, err := os.ReadFile(filepath.Join(media, "trace.zip"))
	if err != nil {
		return err
	}
	if len(screenshot) > 4<<20 || len(trace) > 8<<20 {
		return errors.New("diagnostic fixture media is oversized")
	}
	if err := (&projects.Service{Store: db, Auth: auth}).SaveSuite(ctx, "local-administrator", project, projects.Suite{ID: "diagnostics", Name: "Diagnostic UI fixtures", RequiresTarget: false}, 0); err != nil {
		return err
	}
	now := time.Now().UTC()
	started := now.Add(-3 * time.Second)
	tests := []catalog.Test{
		{ID: "BROWSER-1", Title: "Checkout confirmation", Runner: "playwright", Type: "browser", State: "enabled", Safety: "needs-review", TimeoutSeconds: 1},
		{ID: "BROWSER-2", Title: "Account summary", Runner: "playwright", Type: "browser", State: "enabled", Safety: "needs-review", TimeoutSeconds: 1},
	}
	failed := execution.Result{TestID: "BROWSER-1", Attempt: 1, State: "failed", FailureKind: "assertion", Error: "Expected the confirmation heading", StartedAt: started, FinishedAt: started.Add(20 * time.Millisecond), DurationMs: 20, Steps: []execution.Step{{Title: "Open local fixture", Category: "pw:api", DurationMs: 5}, {Title: "Verify confirmation heading", Category: "expect", DurationMs: 15, Error: "Confirmation heading did not match"}}}
	timed := execution.Result{TestID: "BROWSER-1", Attempt: 2, State: "timed-out", FailureKind: "timeout", Error: "Confirmation exceeded its test limit", StartedAt: started.Add(time.Second), FinishedAt: started.Add(2 * time.Second), DurationMs: 1000, Steps: []execution.Step{{Title: "Wait for confirmation", Category: "expect", DurationMs: 1000, Error: "Test timeout exceeded"}}}
	flaky := execution.Result{TestID: "BROWSER-2", Attempt: 2, State: "flaky", StartedAt: started, FinishedAt: started.Add(10 * time.Millisecond), DurationMs: 10}
	value := runs.Run{ID: domain.NewID(), Project: project, Request: runs.Request{Suite: "diagnostics", Location: "local", TimeoutSeconds: 30, Retries: 1}, Release: catalog.Release{ID: "diagnostic-fixture-release", Project: project, Suite: "diagnostics", Draft: catalog.Draft{Tests: tests, Capture: catalog.DefaultCapture()}}, Tests: tests, Capture: catalog.DefaultCapture(), Actor: "local-administrator", Trigger: "selftest-fixture", State: "completed", Outcome: "timed-out", Phase: "finished", Counts: runs.Counts{Total: 2, Completed: 2, TimedOut: 1, Flaky: 1}, Results: []execution.Result{timed, flaky}, CreatedAt: started, UpdatedAt: now, FinishedAt: &now, EvidenceState: "incomplete", EvidenceErrors: []string{"Fixture capture has an upload gap. The observed timeout remains unchanged."}}
	changes := []storage.Change{}
	for _, attempt := range []execution.Result{failed, timed, flaky} {
		row, err := storage.Put("attempts/"+value.ID, fmt.Sprintf("%s.%04d", attempt.TestID, attempt.Attempt), project, 0, attempt)
		if err != nil {
			return err
		}
		changes = append(changes, row)
	}
	row, err := storage.Put("runs", value.ID, project, 0, value)
	if err != nil {
		return err
	}
	changes = append(changes, row)
	if err := db.Commit(ctx, changes...); err != nil {
		return err
	}
	files, err := filesystem.New("/tmp/teststudio-artifacts")
	if err != nil {
		return err
	}
	defer files.Close()
	service := &evidence.Service{Store: db, Blobs: files}
	for _, a := range []struct {
		test    string
		attempt int
		text    string
	}{{"BROWSER-1", 1, "Checkout attempt one diagnostic"}, {"BROWSER-1", 2, "Checkout attempt two timeout"}, {"BROWSER-2", 2, "Unrelated account diagnostic"}} {
		if _, err := service.Log(ctx, project, value.ID, "fixture-"+a.test, "stdout", a.test, a.attempt, int64(a.attempt), a.text, 100<<20); err != nil {
			return err
		}
	}
	for _, a := range []struct {
		filename, kind, content string
		attempt                 int
		data                    []byte
	}{
		{"checkout-one.png", "screenshot", "image/png", 1, screenshot},
		{"checkout-two.png", "screenshot", "image/png", 2, screenshot},
		{"checkout-trace.zip", "trace", "application/zip", 1, trace},
	} {
		if _, err := service.SaveBytes(ctx, evidence.Artifact{Project: project, Run: value.ID, Test: "BROWSER-1", Attempt: a.attempt, Kind: a.kind, Filename: a.filename, ContentType: a.content}, a.data, 100<<20); err != nil {
			return err
		}
	}
	missing, err := service.SaveBytes(ctx, evidence.Artifact{Project: project, Run: value.ID, Test: "BROWSER-1", Attempt: 1, Kind: "attachment", Filename: "missing.txt", ContentType: "text/plain"}, []byte("Unavailable fixture content"), 100<<20)
	if err != nil {
		return err
	}
	if err := files.Delete(ctx, missing.Project+"/"+missing.Run+"/"+missing.ID+"/"+missing.Checksum); err != nil {
		return err
	}
	if _, reader, err := service.Open(ctx, project, value.ID, missing.ID); err == nil {
		reader.Close()
		return errors.New("missing fixture content remained readable")
	}
	if _, err := service.SaveBytes(ctx, evidence.Artifact{Project: project, Run: value.ID, Test: "BROWSER-1", Attempt: 1, Kind: "attachment", Filename: "expired.txt", ContentType: "text/plain", ExpiresAt: now.Add(-time.Hour)}, []byte("Expired fixture content"), 100<<20); err != nil {
		return err
	}
	failedService := &evidence.Service{Store: db, Blobs: diagnosticUploadFailure{Store: files}}
	if _, err := failedService.SaveBytes(ctx, evidence.Artifact{Project: project, Run: value.ID, Test: "BROWSER-1", Attempt: 1, Kind: "attachment", Filename: "upload-failed.txt", ContentType: "text/plain"}, []byte("Fixture upload gap"), 100<<20); err == nil {
		return errors.New("fixture upload unexpectedly succeeded")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"run": value.ID})
}
