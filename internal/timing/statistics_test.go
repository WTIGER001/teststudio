package timing

import (
	"math"
	"testing"
	"time"
)

func fixtureWindow() (Window, time.Time) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	w := Window{Context: Context{Project: "p", Suite: "s", Test: "T-1", Environment: "dev", Location: "local", Runner: "http", Profile: "native-http-v1"}, Generation: 1}
	for i, ms := range []float64{900, 1100, 1000, 950, 1050} {
		w.Samples = append(w.Samples, Sample{Run: string(rune('a' + i)), DurationMs: ms, At: now.Add(time.Duration(i-5) * time.Minute)})
	}
	return w, now
}

func TestPredictionUsesCleanWindowAndBoundsAdaptiveTimeout(t *testing.T) {
	w, now := fixtureWindow()
	p := Predict(w, 7, DefaultPolicy(), 10000, now)
	if !p.Ready || p.Samples != 5 || *p.MedianMs != 1000 || *p.P95Ms != 1100 || *p.ThresholdMs != 2000 || p.EffectiveTimeoutMs != 4000 || p.Revision != 7 {
		t.Fatalf("prediction: %+v", p)
	}
	slow := Assess(p, 2000, "passed", "")
	if slow.Status != "slow" || *slow.Ratio != 2 {
		t.Fatalf("slow: %+v", slow)
	}
	if Assess(p, 1999, "passed", "").Status != "normal" {
		t.Fatal("below threshold was slow")
	}
	if p := Predict(w, 1, DefaultPolicy(), 2000, now); p.EffectiveTimeoutMs != 2000 {
		t.Fatal("adaptive timeout exceeds catalog ceiling")
	}
	policy := DefaultPolicy()
	policy.TimeoutMode = "fixed"
	if p := Predict(w, 1, policy, 10000, now); p.EffectiveTimeoutMs != 10000 || *p.SuggestedTimeoutMs != 4000 {
		t.Fatal("fixed timeout changed")
	}
	for _, kind := range []string{"timeout", "cancelled"} {
		if a := Assess(p, 3000, "failed", kind); a.Status != "censored" || !a.Censored {
			t.Fatalf("interrupted timing: %+v", a)
		}
	}
	if Assess(p, 0, "skipped", "").Status != "not-executed" {
		t.Fatal("skipped test measured")
	}
}

func TestColdAgedAndNoisySamplesDoNotTriggerPrematureFailures(t *testing.T) {
	w, now := fixtureWindow()
	policy := DefaultPolicy()
	w.Samples = append(w.Samples, Sample{DurationMs: 999999, At: now.Add(-31 * 24 * time.Hour)}, Sample{DurationMs: 999999, At: now.Add(time.Minute)}, Sample{DurationMs: math.NaN(), At: now})
	if p := Predict(w, 1, policy, 10000, now); p.Samples != 5 {
		t.Fatalf("invalid samples included: %d", p.Samples)
	}
	w.Samples = w.Samples[:4]
	p := Predict(w, 1, policy, 10000, now)
	if p.Ready || p.ThresholdMs != nil || p.EffectiveTimeoutMs != 10000 || Assess(p, 3000, "passed", "").Status != "learning" {
		t.Fatal("warmup enforced threshold")
	}
	for i := range w.Samples {
		w.Samples[i].DurationMs = 5
	}
	w.Samples = append(w.Samples, w.Samples[0])
	p = Predict(w, 1, policy, 10000, now)
	if *p.ThresholdMs != 105 || Assess(p, 11, "passed", "").Status != "normal" {
		t.Fatal("small clock variation triggers regression")
	}
	for i := range w.Samples {
		w.Samples[i].DurationMs = 1000
	}
	w.Samples[0].DurationMs = 3000
	p = Predict(w, 1, policy, 10000, now)
	if *p.ThresholdMs != 3000 {
		t.Fatal("historical tail guard omitted")
	}
}

func TestEstimateDoesNotInventUnknownDurationsOrIncludeRetriesInTypicalEstimate(t *testing.T) {
	w, now := fixtureWindow()
	p := Predict(w, 1, DefaultPolicy(), 10000, now)
	e := Summarize(map[string]Prediction{"a": p, "b": p}, 1, 10)
	if *e.ExpectedMs != 2000 || *e.P95Ms != 2200 || e.TimeoutBudgetMs != 16000 || !e.ExceedsRunLimit || e.Provisional {
		t.Fatalf("estimate: %+v", e)
	}
	cold := Predict(Window{Context: w.Context}, 0, DefaultPolicy(), 10000, now)
	e = Summarize(map[string]Prediction{"a": p, "b": cold}, 0, 60)
	if e.ExpectedMs != nil || !e.Provisional || e.MeasuredTests != 1 || e.ReadyTests != 1 {
		t.Fatalf("unknown timing: %+v", e)
	}
}

func TestPolicyRejectsInvalidControls(t *testing.T) {
	for _, mutate := range []func(*Policy){func(p *Policy) { p.Mode = "maybe" }, func(p *Policy) { p.SlowdownFactor = math.Inf(1) }, func(p *Policy) { p.MinimumSamples = 2 }, func(p *Policy) { p.Window = 4 }, func(p *Policy) { p.TimeoutFactor = 0 }, func(p *Policy) { p.MaxAgeDays = 0 }} {
		p := DefaultPolicy()
		mutate(&p)
		if Validate(p) == nil {
			t.Fatalf("invalid policy accepted: %+v", p)
		}
	}
}
