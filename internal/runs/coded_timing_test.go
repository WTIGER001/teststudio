package runs_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type calibratedExecutor struct {
	mu              sync.Mutex
	runner, runtime string
	durations       []float64
	unavailable     bool
	inputs          []execution.CodedInput
}

func (e *calibratedExecutor) Profile() string { return e.runner + "-local-isolated-v1" }
func (e *calibratedExecutor) TimingProfile(context.Context, catalog.Artifact) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Profile() + ":" + e.runtime, nil
}
func (e *calibratedExecutor) Admit(context.Context, catalog.Test, catalog.Artifact) error { return nil }
func (e *calibratedExecutor) Execute(_ context.Context, input execution.CodedInput) execution.CodedReport {
	e.mu.Lock()
	defer e.mu.Unlock()
	duration := e.durations[min(len(e.inputs), len(e.durations)-1)]
	e.inputs = append(e.inputs, input)
	now := time.Now().UTC()
	return execution.CodedReport{Result: execution.Result{TestID: input.Test.ID, State: "passed", StartedAt: now.Add(-time.Duration(duration) * time.Millisecond), FinishedAt: now, DurationMs: duration, DurationUnavailable: e.unavailable, WorkerDurationMs: duration + 20000, Steps: []execution.Step{{Title: "Setup", Category: "test.step"}, {Title: "Request", Category: "test.step"}, {Title: "Assert", Category: "test.step"}}}}
}
func configureCodedTiming(t *testing.T, s *runs.Service, p *projects.Service, runner string) *calibratedExecutor {
	t.Helper()
	testType := "browser"
	if runner == "api" {
		testType = "api"
		runner = "playwright"
	}
	configureGo(t, s, p)
	if runner == "playwright" {
		row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
		if err != nil {
			t.Fatal(err)
		}
		definition, err := storage.Decode[catalog.Test](row)
		if err != nil {
			t.Fatal(err)
		}
		definition.Runner, definition.Type, definition.Selector = "playwright", testType, "@HEALTH-1"
		if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, row.Revision); err != nil {
			t.Fatal(err)
		}
		draft, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{Image: "sha256:" + strings.Repeat("a", 64), Discovered: map[string]string{definition.ID: definition.Selector}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "browser-timing", 1, true); err != nil {
			t.Fatal(err)
		}
	}
	e := &calibratedExecutor{runner: runner, runtime: "first-runtime", durations: []float64{1000}}
	if runner == "go" {
		s.Go = e
	} else {
		s.Playwright = e
	}
	return e
}

func TestCodedTimingUsesFrozenAdaptiveBudgetAndKeepsRetriesOutsideBaseline(t *testing.T) {
	for _, runner := range []string{"go", "playwright", "api"} {
		t.Run(runner, func(t *testing.T) {
			s, p, _ := setup(t, nil)
			e := configureCodedTiming(t, s, p, runner)
			e.durations = []float64{2100, 1000}
			req := runs.Request{Suite: "health", Retries: 1}
			seedTiming(t, s, req)
			queued, err := s.Create(t.Context(), "admin", "project", req, "coded-regression", "manual")
			if err != nil {
				t.Fatal(err)
			}
			prediction := queued.Timing.Tests["HEALTH-1"]
			start(t, s)
			final := finished(t, s, queued.ID)
			if final.Outcome != "flaky" || len(final.Results) != 1 || final.Counts.Total != 1 || final.Counts.Completed != 1 {
				t.Fatal("retry inflated logical results", final)
			}
			result := final.Results[0]
			if len(result.Steps) != 3 || result.Attempt != 2 || result.TotalDurationMs != 3100 || result.TimingRegressions != 1 || result.WorkerDurationMs != 21000 || result.DurationMs != 1000 {
				t.Fatal("coded timing mixed measurement sources", result)
			}
			e.mu.Lock()
			inputs := append([]execution.CodedInput(nil), e.inputs...)
			e.mu.Unlock()
			if len(inputs) != 2 {
				t.Fatal("unexpected attempt count", len(inputs))
			}
			for _, input := range inputs {
				if input.TimeoutMs != prediction.EffectiveTimeoutMs || input.SlowThresholdMs == nil || *input.SlowThresholdMs != *prediction.ThresholdMs || input.TimingMode != "fail" {
					t.Fatal("frozen prediction did not reach coded attempt", input)
				}
			}
			estimate, err := s.Estimate(t.Context(), "admin", "project", req)
			if err != nil || estimate.Tests["HEALTH-1"].Samples != 5 || *estimate.ExpectedMs != 1000 {
				t.Fatal("retry trained history or included worker preparation", estimate, err)
			}
		})
	}
}

func TestCodedTimingCensorsUnavailableDurationButRetainsMeasuredZero(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			s, p, _ := setup(t, nil)
			e := configureCodedTiming(t, s, p, runner)
			e.durations = []float64{0}
			e.unavailable = true
			req := runs.Request{Suite: "health"}
			seedTiming(t, s, req)
			start(t, s)
			queued, err := s.Create(t.Context(), "admin", "project", req, "unavailable-measurement", "manual")
			if err != nil {
				t.Fatal(err)
			}
			final := finished(t, s, queued.ID)
			result := final.Results[0]
			if final.Outcome != "passed" || !result.DurationUnavailable || !result.Timing.Censored || result.Timing.Ratio != nil {
				t.Fatal("unavailable duration became a zero measurement", result)
			}
			estimate, err := s.Estimate(t.Context(), "admin", "project", req)
			if err != nil || estimate.Tests["HEALTH-1"].Samples != 5 {
				t.Fatal("unavailable duration trained timing history", estimate, err)
			}
			e.mu.Lock()
			e.unavailable = false
			e.mu.Unlock()
			queued, err = s.Create(t.Context(), "admin", "project", req, "measured-zero", "manual")
			if err != nil {
				t.Fatal(err)
			}
			final = finished(t, s, queued.ID)
			result = final.Results[0]
			if result.DurationUnavailable || result.DurationMs != 0 || result.Timing.Censored {
				t.Fatal("measured zero was discarded", result)
			}
			estimate, err = s.Estimate(t.Context(), "admin", "project", req)
			if err != nil || estimate.Tests["HEALTH-1"].Samples != 6 {
				t.Fatal("valid zero did not enter history", estimate, err)
			}
		})
	}
}

func TestCodedTimingBlocksRuntimeDriftBeforeDispatchAndStartsSeparateHistory(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			s, p, _ := setup(t, nil)
			e := configureCodedTiming(t, s, p, runner)
			req := runs.Request{Suite: "health"}
			seedTiming(t, s, req)
			queued, err := s.Create(t.Context(), "admin", "project", req, "frozen-runtime", "manual")
			if err != nil {
				t.Fatal(err)
			}
			e.mu.Lock()
			e.runtime = "second-runtime"
			e.mu.Unlock()
			start(t, s)
			final := finished(t, s, queued.ID)
			if final.State != "blocked" || final.Outcome != "unknown" || final.Counts.Completed != 0 || !strings.Contains(final.Error, "runtime changed") {
				t.Fatal("changed runtime used frozen criteria", final)
			}
			e.mu.Lock()
			calls := len(e.inputs)
			e.mu.Unlock()
			if calls != 0 {
				t.Fatal("changed runtime dispatched a test")
			}
			if final.Timing.Tests["HEALTH-1"].Context.Profile != queued.Timing.Tests["HEALTH-1"].Context.Profile {
				t.Fatal("dispatch rewrote frozen history")
			}
			estimate, err := s.Estimate(t.Context(), "admin", "project", req)
			if err != nil || estimate.Tests["HEALTH-1"].Samples != 0 || estimate.ExpectedMs != nil || estimate.Tests["HEALTH-1"].Key == queued.Timing.Tests["HEALTH-1"].Key {
				t.Fatal("different runtime reused a baseline", estimate, err)
			}
		})
	}
}

func TestCodedTimingWithZeroMedianUsesNoiseGuardWithoutInventingRatio(t *testing.T) {
	s, p, _ := setup(t, nil)
	e := configureCodedTiming(t, s, p, "go")
	e.durations = []float64{0, 0, 0, 0, 0, 150}
	req := runs.Request{Suite: "health"}
	start(t, s)
	for index := range 5 {
		queued, err := s.Create(t.Context(), "admin", "project", req, fmt.Sprintf("zero-%d", index), "manual")
		if err != nil {
			t.Fatal(err)
		}
		if final := finished(t, s, queued.ID); final.Outcome != "passed" {
			t.Fatal("clean zero did not pass", final)
		}
	}
	estimate, err := s.Estimate(t.Context(), "admin", "project", req)
	prediction := estimate.Tests["HEALTH-1"]
	if err != nil || !prediction.Ready || prediction.MedianMs == nil || *prediction.MedianMs != 0 || prediction.ThresholdMs == nil || *prediction.ThresholdMs != 100 {
		t.Fatal("zero baseline did not keep its absolute noise guard", prediction, err)
	}
	queued, err := s.Create(t.Context(), "admin", "project", req, "zero-regression", "manual")
	if err != nil {
		t.Fatal(err)
	}
	final := finished(t, s, queued.ID)
	result := final.Results[0]
	if final.Outcome != "failed" || result.FailureKind != "duration-regression" || result.Timing.Ratio != nil || !strings.Contains(result.Error, "0.0 ms baseline") {
		t.Fatal("zero baseline failed ratio handling", result)
	}
}
