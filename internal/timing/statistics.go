package timing

import (
	"math"
	"slices"
	"time"
)

func Predict(window Window, revision int64, p Policy, hardTimeoutMs int, asOf time.Time) Prediction {
	p = Normalize(p)
	v := Prediction{Key: Key(window.Context), Context: window.Context, Generation: max(1, window.Generation), Revision: revision, Policy: p, EffectiveTimeoutMs: hardTimeoutMs, AsOf: asOf}
	cutoff := asOf.Add(-time.Duration(p.MaxAgeDays) * 24 * time.Hour)
	samples := []Sample{}
	for _, sample := range window.Samples {
		if finite(sample.DurationMs) && sample.DurationMs > 0 && !sample.At.Before(cutoff) && !sample.At.After(asOf) {
			samples = append(samples, sample)
		}
	}
	slices.SortFunc(samples, func(a, b Sample) int {
		if n := a.At.Compare(b.At); n != 0 {
			return n
		}
		if a.Run < b.Run {
			return -1
		}
		if a.Run > b.Run {
			return 1
		}
		return 0
	})
	if len(samples) > p.Window {
		samples = samples[len(samples)-p.Window:]
	}
	v.Samples = len(samples)
	if v.Samples == 0 {
		return v
	}
	first, last := samples[0].At, samples[len(samples)-1].At
	v.FirstSampleAt = &first
	v.LastSampleAt = &last
	values := make([]float64, len(samples))
	for i, sample := range samples {
		values[i] = sample.DurationMs
	}
	slices.Sort(values)
	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + median) / 2
	}
	p95 := values[int(math.Ceil(float64(len(values))*0.95))-1]
	v.MedianMs = &median
	v.P95Ms = &p95
	v.Ready = v.Samples >= p.MinimumSamples
	if !v.Ready {
		return v
	}
	threshold := max(median*p.SlowdownFactor, median+p.MinimumIncreaseMs, p95)
	v.ThresholdMs = &threshold
	suggested := int(math.Ceil(max(float64(p.MinimumTimeoutMs), p95*p.TimeoutFactor+p.TimeoutMarginMs)/1000)) * 1000
	suggested = min(suggested, hardTimeoutMs)
	v.SuggestedTimeoutMs = &suggested
	if p.TimeoutMode == "adaptive" {
		v.EffectiveTimeoutMs = suggested
	}
	return v
}

func Assess(prediction Prediction, durationMs float64, observedState, failureKind string) Assessment {
	a := Assessment{Prediction: prediction, Status: "learning", TimeoutMs: prediction.EffectiveTimeoutMs}
	if observedState == "skipped" {
		a.Status = "not-executed"
		return a
	}
	if observedState == "timed-out" || failureKind == "timeout" || failureKind == "cancelled" || failureKind == "panic" || observedState == "unknown" || !finite(durationMs) || durationMs < 0 {
		a.Status = "censored"
		a.Censored = true
		return a
	}
	if prediction.Policy.Mode == "off" {
		a.Status = "disabled"
		return a
	}
	if prediction.MedianMs != nil && *prediction.MedianMs > 0 {
		ratio := durationMs / (*prediction.MedianMs)
		a.Ratio = &ratio
	}
	if prediction.Ready && prediction.ThresholdMs != nil {
		a.Status = "normal"
		if durationMs >= *prediction.ThresholdMs {
			a.Status = "slow"
		}
	}
	return a
}

func Summarize(predictions map[string]Prediction, retries, runLimitSeconds int) Estimate {
	e := Estimate{Tests: predictions, TotalTests: len(predictions)}
	var expected, high float64
	for _, p := range predictions {
		if p.Ready {
			e.ReadyTests++
		}
		if p.MedianMs != nil && p.P95Ms != nil {
			e.MeasuredTests++
			expected += *p.MedianMs
			high += *p.P95Ms
		}
		e.TimeoutBudgetMs += int64(p.EffectiveTimeoutMs) * int64(retries+1)
	}
	if e.MeasuredTests == e.TotalTests && e.TotalTests > 0 {
		e.ExpectedMs = &expected
		e.P95Ms = &high
	}
	e.Provisional = e.ReadyTests < e.TotalTests
	e.ExceedsRunLimit = e.TimeoutBudgetMs > int64(runLimitSeconds)*1000
	return e
}
