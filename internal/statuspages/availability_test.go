package statuspages

import (
	"math"
	"testing"
	"time"
)

func TestAvailabilityUsesTimeAndClipsUnknownGaps(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(24 * time.Hour)
	points := []Point{{At: start, State: "passed"}, {At: start.Add(10 * time.Hour), State: "failed"}, {At: start.Add(11 * time.Hour), State: "passed"}}
	result := Calculate(points, start, now, 24*time.Hour)
	if result.Percent == nil || math.Abs(*result.Percent-95.833333) > 0.00001 || result.Up != 23*3600 || result.Down != 3600 || result.Unknown != 0 {
		t.Fatal(result)
	}
	dense := append([]Point{}, points...)
	for i := 1; i < 10; i++ {
		dense = append(dense, Point{At: start.Add(10*time.Hour + time.Duration(i)*time.Minute), State: "failed"})
	}
	again := Calculate(dense, start, now, 24*time.Hour)
	if *again.Percent != *result.Percent {
		t.Fatal("extra checks changed duration weighting", again)
	}
	stale := Calculate(points, start, now, 15*time.Minute)
	if stale.Up != 1800 || stale.Down != 900 || stale.Unknown != 24*3600-2700 || stale.Coverage != 3.125 {
		t.Fatal(stale)
	}
	clipped := Calculate([]Point{{At: start.Add(-5 * time.Minute), State: "passed"}}, start, start.Add(time.Hour), 15*time.Minute)
	if clipped.Up != 600 || clipped.Unknown != 3000 {
		t.Fatal(clipped)
	}
	empty := Calculate(nil, start, now, time.Minute)
	if empty.Percent != nil || empty.Unknown != 86400 {
		t.Fatal(empty)
	}
	unknown := Calculate([]Point{{At: start, State: "passed"}, {At: start.Add(time.Minute), State: "unknown"}}, start, start.Add(time.Hour), time.Hour)
	if unknown.Up != 60 || unknown.Unknown != 3540 {
		t.Fatal(unknown)
	}
}
func TestAvailabilitySupportsYearAndAllRetainedHistory(t *testing.T) {
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	for window, days := range map[string]int{"24h": 1, "7d": 7, "1m": 30, "1y": 365} {
		start, err := WindowStart(window, now)
		if err != nil || now.Sub(start) != time.Duration(days)*24*time.Hour {
			t.Fatal(window, start, err)
		}
	}
	start, err := WindowStart("all", now)
	if err != nil || !start.IsZero() {
		t.Fatal(start, err)
	}
	first := now.Add(-500 * 24 * time.Hour)
	result := Calculate([]Point{{At: first, State: "passed"}}, start, now, time.Hour)
	if !result.Since.Equal(first) || result.Up != 3600 || result.Unknown != 500*86400-3600 {
		t.Fatal(result)
	}
	if _, err = WindowStart("invalid", now); err == nil {
		t.Fatal("invalid range accepted")
	}
}
