package runs_test

import (
	"context"
	"fmt"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/timing"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestLocalHTTPHistoryPredictsAndFailsSlowAssertionsWithoutTrainingRegression(t *testing.T) {
	var delay atomic.Int64
	delay.Store(40)
	s, _, _ := setup(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(delay.Load()) * time.Millisecond)
		w.Write([]byte(`{"status":"up"}`))
	}))
	req := runs.Request{Suite: "health", Environment: "local", TimeoutSeconds: 20}
	initial, err := s.Estimate(t.Context(), "admin", "project", req)
	if err != nil || initial.ExpectedMs != nil || initial.ReadyTests != 0 {
		t.Fatalf("cold estimate: %+v %v", initial, err)
	}
	rows, err := s.Store.All(t.Context(), "runs", "project")
	if err != nil || len(rows) != 0 {
		t.Fatal("estimate created a run")
	}
	start(t, s)
	for i := range 5 {
		v, err := s.Create(t.Context(), "admin", "project", req, fmt.Sprintf("warm-%d", i), "manual")
		if err != nil {
			t.Fatal(err)
		}
		v = finished(t, s, v.ID)
		if v.Outcome != "passed" || v.Results[0].Timing.Status != "learning" || v.Results[0].DurationMs < 30 {
			t.Fatalf("warmup: %+v", v.Results)
		}
	}
	estimate, err := s.Estimate(t.Context(), "admin", "project", req)
	p := estimate.Tests["HEALTH-1"]
	if err != nil || !p.Ready || p.Samples != 5 || p.EffectiveTimeoutMs != 1000 || estimate.ExpectedMs == nil {
		t.Fatalf("warm estimate: %+v %v", estimate, err)
	}
	delay.Store(350)
	queued, err := s.Create(t.Context(), "admin", "project", req, "slow", "manual")
	if err != nil {
		t.Fatal(err)
	}
	v := finished(t, s, queued.ID)
	result := v.Results[0]
	if v.Outcome != "failed" || result.FailureKind != "duration-regression" || result.ObservedState != "passed" || result.Timing.Status != "slow" || result.TotalDurationMs != result.DurationMs || result.TimingRegressions != 1 {
		t.Fatalf("slow result: %+v", result)
	}
	after, err := s.Estimate(t.Context(), "admin", "project", req)
	if err != nil || after.Tests["HEALTH-1"].Samples != 5 {
		t.Fatal("regression contaminated baseline")
	}
	if result.Timing.Prediction.Samples != queued.Timing.Tests["HEALTH-1"].Samples {
		t.Fatal("prediction changed after queue")
	}
}

type timedExecutor struct {
	calls     atomic.Int32
	durations []float64
	states    []string
	kind      string
}

func (e *timedExecutor) Execute(ctx context.Context, test catalog.Test, _ projects.Environment) execution.Result {
	i := min(int(e.calls.Add(1))-1, len(e.durations)-1)
	now := time.Now().UTC()
	return execution.Result{TestID: test.ID, State: e.states[min(i, len(e.states)-1)], FailureKind: e.kind, DurationMs: e.durations[i], StartedAt: now, FinishedAt: now}
}

func seedTiming(t *testing.T, s *runs.Service, req runs.Request) timing.Prediction {
	t.Helper()
	estimate, err := s.Estimate(t.Context(), "admin", "project", req)
	if err != nil {
		t.Fatal(err)
	}
	p := estimate.Tests["HEALTH-1"]
	service := timing.Service{Store: s.Store}
	for i := range 5 {
		changes, err := service.RecordChanges(t.Context(), p, timing.Sample{Run: fmt.Sprintf("seed-%d", i), DurationMs: 1000, At: time.Now().UTC().Add(-time.Minute)})
		if err == nil {
			err = s.Store.Commit(t.Context(), changes...)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	estimate, err = s.Estimate(t.Context(), "admin", "project", req)
	if err != nil {
		t.Fatal(err)
	}
	return estimate.Tests["HEALTH-1"]
}

func TestTimingPolicyModesRetriesAndFunctionalFailureRemainDistinct(t *testing.T) {
	cases := []struct {
		name, mode            string
		durations             []float64
		states                []string
		kind, outcome, status string
		samples, regressions  int
	}{
		{"fail", "fail", []float64{2100}, []string{"passed"}, "", "failed", "slow", 5, 1},
		{"warn", "warn", []float64{2100}, []string{"passed"}, "", "passed", "slow", 5, 1},
		{"off", "off", []float64{2100}, []string{"passed"}, "", "passed", "disabled", 5, 0},
		{"retry", "fail", []float64{2100, 1000}, []string{"passed", "passed"}, "", "flaky", "normal", 5, 1},
		{"functional", "fail", []float64{2100}, []string{"failed"}, "assertion", "failed", "slow", 5, 1},
		{"timeout", "fail", []float64{4000}, []string{"failed"}, "timeout", "failed", "censored", 5, 0},
		{"clean", "fail", []float64{1100}, []string{"passed"}, "", "passed", "normal", 6, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := setup(t, nil)
			row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
			if err != nil {
				t.Fatal(err)
			}
			test, err := storage.Decode[catalog.Test](row)
			if err != nil {
				t.Fatal(err)
			}
			policy := timing.DefaultPolicy()
			policy.Mode = tc.mode
			test.Timing = &policy
			if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", test, row.Revision); err != nil {
				t.Fatal(err)
			}
			draft, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", draft.ID, "timing-release", 1, true); err != nil {
				t.Fatal(err)
			}
			req := runs.Request{Suite: "health", Environment: "local"}
			if tc.name == "retry" {
				req.Retries = 1
			}
			seedTiming(t, s, req)
			s.HTTP = &timedExecutor{durations: tc.durations, states: tc.states, kind: tc.kind}
			v, err := s.Create(t.Context(), "admin", "project", req, "timed", "manual")
			if err != nil {
				t.Fatal(err)
			}
			start(t, s)
			v = finished(t, s, v.ID)
			result := v.Results[0]
			if v.Outcome != tc.outcome || result.Timing.Status != tc.status || result.TimingRegressions != tc.regressions {
				t.Fatalf("unexpected result: %+v", result)
			}
			if tc.kind != "" && result.FailureKind != tc.kind {
				t.Fatal("timing overwrote functional failure")
			}
			total := float64(0)
			for _, ms := range tc.durations {
				total += ms
			}
			if result.TotalDurationMs != total {
				t.Fatalf("retry time omitted: %f", result.TotalDurationMs)
			}
			e, err := s.Estimate(t.Context(), "admin", "project", req)
			if err != nil || e.Tests[test.ID].Samples != tc.samples {
				t.Fatalf("baseline training: %+v %v", e, err)
			}
		})
	}
}

func TestTimingScopeTracksDefinitionAndTargetButSurvivesReleaseAndLabelChanges(t *testing.T) {
	s, p, _ := setup(t, nil)
	req := runs.Request{Suite: "health", Environment: "local"}
	original := seedTiming(t, s, req)
	row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), "HEALTH-1")
	if err != nil {
		t.Fatal(err)
	}
	test, err := storage.Decode[catalog.Test](row)
	if err != nil {
		t.Fatal(err)
	}
	publish := func(key string) {
		t.Helper()
		if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", test, row.Revision); err != nil {
			t.Fatal(err)
		}
		row, err = s.Store.Get(t.Context(), row.Bucket, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		d, err := s.Catalog.CreateDraft(t.Context(), "admin", "project", "health", catalog.Artifact{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Catalog.Publish(t.Context(), "admin", "project", "health", d.ID, key, 1, true); err != nil {
			t.Fatal(err)
		}
	}
	test.Title = "Updated label"
	test.TimeoutSeconds = 20
	publish("labels")
	e, err := s.Estimate(t.Context(), "admin", "project", req)
	if err != nil || e.Tests[test.ID].Key != original.Key || e.Tests[test.ID].Samples != 5 {
		t.Fatal("metadata or release reset baseline")
	}
	test.Check.Path = "/different"
	publish("definition")
	e, err = s.Estimate(t.Context(), "admin", "project", req)
	if err != nil || e.Tests[test.ID].Key == original.Key || e.Tests[test.ID].Samples != 0 {
		t.Fatal("definition mixed baselines")
	}
	test.Check.Path = "/healthz"
	publish("restore")
	envRow, err := s.Store.Get(t.Context(), projects.Bucket("environments", "project"), "local")
	if err != nil {
		t.Fatal(err)
	}
	env, err := storage.Decode[projects.Environment](envRow)
	if err != nil {
		t.Fatal(err)
	}
	env.Endpoints["api"] = "http://127.0.0.1:9876"
	if err := p.SaveEnvironment(t.Context(), "admin", "project", env, envRow.Revision); err != nil {
		t.Fatal(err)
	}
	e, err = s.Estimate(t.Context(), "admin", "project", req)
	if err != nil || e.Tests[test.ID].Key == original.Key || e.Tests[test.ID].Samples != 0 {
		t.Fatal("target change mixed baselines")
	}
}

func TestAdaptiveTimeoutActuallyCancelsNativeHTTPAtFrozenLimit(t *testing.T) {
	s, _, _ := setup(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	req := runs.Request{Suite: "health", Environment: "local", TimeoutSeconds: 20}
	p := seedTiming(t, s, req)
	if p.EffectiveTimeoutMs != 4000 {
		t.Fatal(p.EffectiveTimeoutMs)
	}
	service := timing.Service{Store: s.Store}
	if err := service.Reset(t.Context(), "admin", "project", p.Key, "Fast local target", p.Revision); err != nil {
		t.Fatal(err)
	}
	e, err := s.Estimate(t.Context(), "admin", "project", req)
	if err != nil {
		t.Fatal(err)
	}
	p = e.Tests["HEALTH-1"]
	for i := range 5 {
		changes, err := service.RecordChanges(t.Context(), p, timing.Sample{Run: fmt.Sprintf("fast-%d", i), DurationMs: 10, At: time.Now().UTC().Add(-time.Second)})
		if err == nil {
			err = s.Store.Commit(t.Context(), changes...)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	v, err := s.Create(t.Context(), "admin", "project", req, "adaptive", "manual")
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	v = finished(t, s, v.ID)
	r := v.Results[0]
	if r.FailureKind != "timeout" || !r.Timing.Censored || r.Timing.TimeoutMs != 1000 || r.DurationMs < 900 || r.DurationMs > 3000 {
		t.Fatalf("adaptive timeout not applied: %+v", r)
	}
}

func TestConcurrentRunsRetainEachCleanTimingSample(t *testing.T) {
	s, _, _ := setup(t, nil)
	s.Concurrency = 4
	s.HTTP = &timedExecutor{durations: []float64{1000}, states: []string{"passed"}}
	req := runs.Request{Suite: "health", Environment: "local"}
	ids := []string{}
	for i := range 12 {
		v, err := s.Create(t.Context(), "admin", "project", req, fmt.Sprintf("parallel-%d", i), "manual")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.ID)
	}
	start(t, s)
	for _, id := range ids {
		v := finished(t, s, id)
		if v.Outcome != "passed" {
			t.Fatalf("parallel run blocked: %s %s", v.State, v.Error)
		}
	}
	e, err := s.Estimate(t.Context(), "admin", "project", req)
	if err != nil || e.Tests["HEALTH-1"].Samples != 12 {
		t.Fatalf("lost concurrent run samples: %+v %v", e, err)
	}
}
