package trends

import (
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
)

func sample(id string, at time.Time, states ...string) runs.Run {
	run := runs.Run{ID: id, Project: "p", State: "completed", Outcome: "passed", CreatedAt: at, Request: runs.Request{Suite: "suite", Environment: "dev", Location: "local"}, Environment: &projects.Environment{ID: "dev", Name: "Development"}}
	for i, state := range states {
		id := string(rune('A' + i))
		run.Tests = append(run.Tests, catalog.Test{ID: id, Title: "Check " + id, Category: "Availability"})
		if state != "" {
			run.Results = append(run.Results, execution.Result{TestID: id, State: state, Attempt: 3})
		}
	}
	return run
}

func TestFinalCountsKeepTimeoutsUnknownsAndRetriesDistinct(t *testing.T) {
	now := time.Now().UTC()
	f := Filter{Since: now.Add(-time.Hour), Until: now.Add(time.Hour)}
	one := sample("one", now, "passed", "failed", "failed", "flaky", "skipped", "unknown", "")
	one.Results[2].FailureKind = "timeout"
	two := sample("two", now.Add(time.Minute), "passed")
	two.State = "running"
	three := sample("three", now.Add(2*time.Minute), "")
	three.State = "incomplete"
	three.Outcome = "unknown"
	r := Aggregate([]runs.Run{three, two, one}, f)
	if len(r.Points) != 2 || r.Points[0].Run != "one" || r.Samples != 2 {
		t.Fatalf("ordered terminal samples: %+v", r.Points)
	}
	want := Counts{Total: 8, Passed: 1, Failed: 1, TimedOut: 1, Flaky: 1, Skipped: 1, Unknown: 3}
	if r.Totals != want {
		t.Fatalf("logical counts: got %+v want %+v", r.Totals, want)
	}
	if len(r.FailureRankings) != 2 || r.FailureRankings[0].Counts.Total != 1 {
		t.Fatalf("failure ranks inflated attempts: %+v", r.FailureRankings)
	}
}

func TestTestAndCategoryViewsKeepMissingSelectionsAsGaps(t *testing.T) {
	now := time.Now().UTC()
	f := Filter{Since: now.Add(-time.Hour), Until: now.Add(time.Hour), Suite: "suite", Test: "A"}
	one := sample("one", now, "passed")
	two := sample("two", now.Add(time.Minute), "failed")
	two.Tests[0].ID = "OTHER"
	two.Tests[0].Category = "Other"
	two.Results[0].TestID = "OTHER"
	three := sample("three", now.Add(2*time.Minute), "failed")
	r := Aggregate([]runs.Run{one, two, three}, f)
	if len(r.Points) != 3 || r.Points[1].Gap == "" || r.Points[1].Counts.Total != 0 || r.Samples != 2 {
		t.Fatalf("selection gap: %+v", r)
	}
	if len(r.VolatilityRankings) != 0 {
		t.Fatal("missing selection was interpolated into a volatility comparison")
	}
	f.Test = ""
	f.Category = "Availability"
	r = Aggregate([]runs.Run{one, two, three}, f)
	if r.Points[1].Gap == "" || r.Totals.Total != 2 {
		t.Fatal("category gap was hidden")
	}
	if len(r.Options.Categories) != 2 {
		t.Fatal("historical categories were lost")
	}
}

func TestVolatilityComparesAdjacentKnownResultsWithinTargetAndLocation(t *testing.T) {
	now := time.Now().UTC()
	f := Filter{Since: now.Add(-time.Hour), Until: now.Add(time.Hour)}
	input := []runs.Run{}
	for i, state := range []string{"passed", "failed", "unknown", "passed", "timed-out", "skipped", "failed", "flaky"} {
		input = append(input, sample(string(rune('a'+i)), now.Add(time.Duration(i)*time.Minute), state))
	}
	other := sample("other-target", now.Add(30*time.Second), "failed")
	other.Request.Environment = "prod"
	other.Environment.ID = "prod"
	input = append(input, other)
	otherSuite := sample("other-suite", now.Add(45*time.Second), "failed")
	otherSuite.Request.Suite = "other"
	input = append(input, otherSuite)
	r := Aggregate(input, f)
	if len(r.VolatilityRankings) != 1 {
		t.Fatalf("volatility ranks: %+v", r.VolatilityRankings)
	}
	rank := r.VolatilityRankings[0]
	if rank.Comparisons != 3 || rank.Changes != 3 || rank.Volatility == nil || *rank.Volatility != 1 {
		t.Fatalf("comparable transitions: %+v", rank)
	}
	f.Environment = "prod"
	r = Aggregate(input, f)
	if r.Samples != 1 || r.Totals.Failed != 1 || len(r.VolatilityRankings) != 0 {
		t.Fatal("target history was mixed")
	}
}

func TestSnapshotIdentityDeduplicatesAndDoesNotInventMissingMetadata(t *testing.T) {
	now := time.Now().UTC()
	f := Filter{Since: now.Add(-time.Hour), Until: now.Add(time.Hour)}
	one := sample("one", now, "passed")
	one.Tests = append(one.Tests, one.Tests[0])
	one.Results = append(one.Results, one.Results[0])
	two := sample("two", now.Add(time.Minute))
	two.Counts.Total = 10
	r := Aggregate([]runs.Run{one, two}, f)
	if r.Points[0].Counts.Total != 1 || r.Points[0].Counts.Unknown != 1 || r.Points[1].Gap == "" || r.Samples != 1 {
		t.Fatalf("ambiguous data appeared healthy: %+v", r)
	}
}

func TestApplicationTrendsExcludeForeignRunsTestsAndOptions(t *testing.T) {
	now := time.Now().UTC()
	shared := sample("shared", now, "passed", "failed", "timed-out")
	shared.Environment = nil
	shared.Request.Environment = ""
	shared.Applications = []string{"shop", "warehouse"}
	shared.Tests[0].Application = "shop"
	shared.Tests[1].Application = "warehouse"
	shared.Tests[1].Category = "Warehouse only"
	foreign := sample("foreign", now.Add(time.Minute), "failed")
	foreign.Environment.Application = "warehouse"
	foreign.Request.Suite = "foreign-suite"
	report := Aggregate([]runs.Run{shared, foreign}, Filter{Application: "shop", Since: now.Add(-time.Hour), Until: now.Add(time.Hour)})
	if report.Totals.Total != 2 || report.Totals.Passed != 1 || report.Totals.TimedOut != 1 || len(report.Points) != 1 {
		t.Fatal(report)
	}
	if len(report.Options.Suites) != 1 || len(report.Options.Categories) != 1 || report.Options.Categories[0] != "Availability" {
		t.Fatal(report.Options)
	}
	if len(report.FailureRankings) != 1 || report.FailureRankings[0].Test != "C" {
		t.Fatal(report.FailureRankings)
	}
}
