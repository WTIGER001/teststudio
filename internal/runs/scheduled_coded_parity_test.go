package runs_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/trends"
)

type parityExecutor struct {
	runner, state string
	inputs        chan execution.CodedInput
}

func (e *parityExecutor) Profile() string                                             { return e.runner + "-local-isolated-v1" }
func (e *parityExecutor) Admit(context.Context, catalog.Test, catalog.Artifact) error { return nil }
func (e *parityExecutor) TimingProfile(context.Context, catalog.Artifact) (string, error) {
	return e.Profile() + ":parity-fixture-v1", nil
}
func (e *parityExecutor) Execute(_ context.Context, input execution.CodedInput) execution.CodedReport {
	e.inputs <- input
	now := time.Now().UTC()
	if input.OnStart != nil {
		_ = input.OnStart(now.Add(-20 * time.Millisecond))
	}
	state := e.state
	if state == "flaky" {
		state = "failed"
		if input.FailedBefore {
			state = "passed"
		}
	}
	result := execution.Result{TestID: input.Test.ID, State: state, StartedAt: now.Add(-20 * time.Millisecond), FinishedAt: now, DurationMs: 20, Steps: []execution.Step{{Title: "First child", Category: "test.step", DurationMs: 7}, {Title: "Second child", Category: "test.step", DurationMs: 8}}}
	if state == "timed-out" {
		result.FailureKind = "timeout"
	}
	if state == "failed" {
		result.FailureKind = "assertion"
	}
	if e.runner == "go" {
		result.Go = &execution.GoObservation{Package: "./unit", Test: "TestFresh", Toolchain: "go1.26"}
	}
	return execution.CodedReport{Result: result, Attachments: []execution.Attachment{{Kind: "log", Stream: e.runner + ".stdout", Filename: "worker.log", ContentType: "text/plain", Data: []byte("new targetless execution evidence")}}}
}

func TestTargetlessCodedManualAndScheduledRunsKeepIdenticalLogicalContracts(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, state := range []string{"passed", "failed", "timed-out", "flaky", "skipped"} {
			t.Run(runner+"/"+state, func(t *testing.T) {
				service, projectService, _ := setup(t, nil)
				configureGo(t, service, projectService)
				if runner == "playwright" {
					row, err := service.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
					if err != nil {
						t.Fatal(err)
					}
					definition, _ := storage.Decode[catalog.Test](row)
					definition.Runner, definition.Type, definition.Selector = "playwright", "browser", "@HEALTH-1"
					if err := service.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, row.Revision); err != nil {
						t.Fatal(err)
					}
					draft, err := service.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{Image: "sha256:" + strings.Repeat("a", 64), Discovered: map[string]string{definition.ID: definition.Selector}})
					if err != nil {
						t.Fatal(err)
					}
					if _, err := service.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "playwright-parity", 1, true); err != nil {
						t.Fatal(err)
					}
				}
				driver := &parityExecutor{runner: runner, state: state, inputs: make(chan execution.CodedInput, 8)}
				if runner == "go" {
					service.Go = driver
				} else {
					service.Playwright = driver
				}
				request := runs.Request{Suite: "health", Retries: 1}
				manual, err := service.Create(t.Context(), "admin", "project", request, "targetless-manual", "manual")
				if err != nil {
					t.Fatal(err)
				}
				clock := time.Now().UTC()
				scheduler := &schedules.Service{Store: service.Store, Auth: service.Auth, Runs: service, Now: func() time.Time { return clock }}
				definition := scheduleDefinition("targetless")
				definition.Request = request
				if err := scheduler.Save(t.Context(), "admin", "project", definition, 0); err != nil {
					t.Fatal(err)
				}
				configured, _ := readSchedule(t, scheduler, "targetless")
				clock = *configured.NextAt
				tickSchedule(t, scheduler)
				configured, _ = readSchedule(t, scheduler, "targetless")
				scheduled, _, err := service.Get(t.Context(), "project", configured.LastRun)
				if err != nil {
					t.Fatal(err)
				}
				for id, prediction := range scheduled.Timing.Tests {
					prediction.AsOf = manual.Timing.Tests[id].AsOf
					scheduled.Timing.Tests[id] = prediction
				}
				if scheduled.Trigger != "schedule" || manual.Trigger != "manual" || scheduled.Schedule == nil || scheduled.Environment != nil || manual.Environment != nil || !reflect.DeepEqual(manual.Tests, scheduled.Tests) || manual.Release.ID != scheduled.Release.ID || !reflect.DeepEqual(manual.TestCapture, scheduled.TestCapture) || !reflect.DeepEqual(manual.Timing, scheduled.Timing) {
					t.Fatal("trigger changes targetless execution snapshots")
				}
				start(t, service)
				manual = finished(t, service, manual.ID)
				scheduled = finished(t, service, scheduled.ID)
				if manual.State != scheduled.State || manual.Outcome != scheduled.Outcome || !reflect.DeepEqual(manual.Counts, scheduled.Counts) || manual.Counts.Total != 1 || len(manual.Results) != 1 || len(scheduled.Results) != 1 || manual.Results[0].State != state || scheduled.Results[0].State != state {
					t.Fatal("trigger changes logical results", manual.State, scheduled.State, manual.Results, scheduled.Results)
				}
				expectedAttempts := 1
				if state == "failed" || state == "timed-out" || state == "flaky" {
					expectedAttempts = 2
				}
				for _, run := range []runs.Run{manual, scheduled} {
					attempts, err := service.Store.All(t.Context(), "attempts/"+run.ID, "project")
					if err != nil || len(attempts) != expectedAttempts {
						t.Fatal("attempt parity differs", err, len(attempts))
					}
					if run.EvidenceState != "complete" || len(run.Results[0].Steps) != 2 || run.Results[0].Timing == nil || run.Results[0].Timing.Censored != (state == "timed-out") {
						t.Fatal("trigger changes evidence, timing, or step reduction")
					}
					rows, err := service.Store.All(t.Context(), evidence.Bucket(run.ID), "project")
					if err != nil {
						t.Fatal(err)
					}
					found := false
					for _, row := range rows {
						artifact, _ := storage.Decode[evidence.Artifact](row)
						if artifact.Stream == runner+".stdout" {
							_, reader, err := service.Evidence.Open(t.Context(), "project", run.ID, artifact.ID)
							if err != nil {
								t.Fatal(err)
							}
							_ = reader.Close()
							found = true
						}
					}
					if !found {
						t.Fatal("coded evidence missing")
					}
				}
				payloads := map[string][]json.RawMessage{}
				for len(driver.inputs) > 0 {
					input := <-driver.inputs
					if input.Browser != nil {
						t.Fatal("targetless scheduling invents a target")
					}
					id := input.RunID
					input.RunID = ""
					data, err := json.Marshal(input)
					if err != nil {
						t.Fatal(err)
					}
					payloads[id] = append(payloads[id], data)
				}
				if !reflect.DeepEqual(payloads[manual.ID], payloads[scheduled.ID]) {
					t.Fatal("trigger changes coded work items")
				}
				history, err := service.TestHistory(t.Context(), "admin", "project", "health", "HEALTH-1", "", 10)
				if err != nil || len(history.Items) != 2 {
					t.Fatal("targetless history differs", err)
				}
				trend, err := (&trends.Service{Store: service.Store, Auth: service.Auth}).Read(t.Context(), "admin", "project", trends.Filter{Suite: "health", Since: time.Now().Add(-time.Hour), Until: time.Now().Add(time.Minute)}, "", 10)
				if err != nil || trend.Totals.Total != 2 || len(trend.Points) != 2 {
					t.Fatal("children or attempts inflate trends", err, trend.Totals)
				}
				report, err := service.ReliabilityReport(t.Context(), "admin", reliability.Query{Scope: reliability.Scope{Project: "project", Suite: "health", Test: "HEALTH-1", Location: "local"}}, "24h")
				if err != nil {
					t.Fatal(err)
				}
				row := report.Probes[0]
				if row.Metric != "outcomes" || row.Totals.Counts.Total() != 2 || row.Availability.Percent != nil {
					t.Fatal("runner reliability contract", row)
				}
				if state == "timed-out" && (row.Totals.Counts.TimedOut != 2 || row.Totals.Censored != 2) {
					t.Fatal("timeouts lost their distinct state or censoring", row)
				}
				if state == "flaky" && row.Totals.Counts.Flaky != 2 {
					t.Fatal("retries inflated final observations", row)
				}
			})
		}
	}
}
