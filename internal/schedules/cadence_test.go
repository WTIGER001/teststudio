package schedules

import (
	"github.com/wtiger001/teststudio/internal/runs"
	"testing"
	"time"
)

func definition() Definition {
	return normalize(Definition{ID: "daily", Name: "Daily", Kind: "cron", Cron: "30 2 * * *", Timezone: "America/New_York", Request: runs.Request{Suite: "health"}}, "local")
}
func TestCronUsesExplicitZonesAndSkipsNonexistentDSTTimes(t *testing.T) {
	d := definition()
	start := time.Date(2026, 3, 8, 6, 0, 0, 0, time.UTC)
	next, err := Next(d, start)
	want := time.Date(2026, 3, 9, 6, 30, 0, 0, time.UTC)
	if err != nil || !next.Equal(want) {
		t.Fatalf("DST transition: %s %v", next, err)
	}
	d.Cron = "30 1 * * *"
	start = time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)
	first, err := Next(d, start)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Next(d, first)
	if err != nil || first.Hour() != 5 || second.Hour() != 6 || first.Day() != 1 || second.Day() != 1 {
		t.Fatalf("repeated DST hour: %s %s %v", first, second, err)
	}
}
func TestIntervalsCoalesceMissedTicksWithoutMovingTheirPhase(t *testing.T) {
	d := normalize(Definition{ID: "probe", Name: "Probe", Kind: "interval", IntervalSeconds: 5, Request: runs.Request{Suite: "health"}}, "local")
	due := time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC)
	next, coalesced, err := advance(d, due, due.Add(31*time.Second))
	if err != nil || !coalesced || !next.Equal(due.Add(35*time.Second)) {
		t.Fatalf("coalescing: %s %t %v", next, coalesced, err)
	}
}
func TestScheduleCadenceRejectsAmbiguityAndUnboundedInput(t *testing.T) {
	for _, change := range []func(*Definition){
		func(d *Definition) { d.Cron = "@every 1s" }, func(d *Definition) { d.Cron = "* * * * * *" }, func(d *Definition) { d.Cron = "0 0 31 2 *" },
		func(d *Definition) { d.Timezone = "Local" }, func(d *Definition) { d.Timezone = "Unavailable/Zone" }, func(d *Definition) { d.IntervalSeconds = 5 },
		func(d *Definition) { d.Request.Parent = "parent" }, func(d *Definition) { d.Request.IDs = []string{"TEST-1", "TEST-1"} },
	} {
		d := definition()
		change(&d)
		if _, err := Next(d, time.Now()); err == nil {
			t.Fatalf("invalid cadence accepted: %+v", d)
		}
	}
}
