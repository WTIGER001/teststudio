package trends

import (
	"cmp"
	"slices"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/runs"
)

func (c *Counts) add(state string) {
	c.Total++
	switch state {
	case "passed":
		c.Passed++
	case "failed":
		c.Failed++
	case "timed-out":
		c.TimedOut++
	case "flaky":
		c.Flaky++
	case "skipped":
		c.Skipped++
	default:
		c.Unknown++
	}
}

func addCounts(a *Counts, b Counts) {
	a.Total += b.Total
	a.Passed += b.Passed
	a.Failed += b.Failed
	a.TimedOut += b.TimedOut
	a.Flaky += b.Flaky
	a.Skipped += b.Skipped
	a.Unknown += b.Unknown
}

func inPeriod(run runs.Run, f Filter) bool {
	return runs.Terminal(run.State) && !run.CreatedAt.Before(f.Since) && !run.CreatedAt.After(f.Until)
}

func matches(run runs.Run, f Filter) bool {
	if !runs.InApplication(run, f.Application) {
		return false
	}
	if f.Suite != "" && run.Request.Suite != f.Suite {
		return false
	}
	if f.Location != "" && run.Request.Location != f.Location {
		return false
	}
	if f.Environment == "none" {
		return run.Environment == nil
	}
	return f.Environment == "" || run.Request.Environment == f.Environment
}

type previous struct {
	index           int
	state, category string
}

func Aggregate(input []runs.Run, f Filter) Report {
	ordered := slices.Clone(input)
	slices.SortFunc(ordered, func(a, b runs.Run) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	report := Report{Filter: f, Points: []Point{}, FailureRankings: []Ranking{}, VolatilityRankings: []Ranking{}, Options: options(ordered, f)}
	ranks := map[string]*Ranking{}
	prior := map[string]previous{}
	indices := map[string]int{}
	for _, stored := range ordered {
		if !inPeriod(stored, f) || !matches(stored, f) {
			continue
		}
		run := runs.NormalizeTimeouts(stored)
		point := Point{Run: run.ID, At: run.CreatedAt, Suite: run.Request.Suite, Environment: run.Request.Environment, Location: run.Request.Location, State: run.State, Outcome: run.Outcome}
		chain := run.Request.Suite + "\x00" + run.Request.Environment + "\x00" + run.Request.Location
		indices[chain]++
		results := map[string]string{}
		for _, result := range run.Results {
			if _, duplicate := results[result.TestID]; duplicate {
				results[result.TestID] = "unknown"
			} else {
				results[result.TestID] = execution.NormalizeTimeout(result).State
			}
		}
		seen := map[string]bool{}
		for _, test := range run.Tests {
			if !runs.TestInApplication(run, test, f.Application) || test.ID == "" || seen[test.ID] || (f.Category != "" && test.Category != f.Category) || (f.Test != "" && test.ID != f.Test) {
				continue
			}
			seen[test.ID] = true
			state := results[test.ID]
			point.Counts.add(state)
			key := run.Request.Suite + "/" + test.ID
			rank := ranks[key]
			if rank == nil {
				rank = &Ranking{Suite: run.Request.Suite, Test: test.ID}
				ranks[key] = rank
			}
			rank.Title, rank.Category = test.Title, test.Category
			rank.Counts.add(state)
			comparison := key + "\x00" + chain
			if state != "passed" && state != "failed" && state != "timed-out" && state != "flaky" {
				delete(prior, comparison)
				continue
			}
			if before, ok := prior[comparison]; ok && before.index == indices[chain]-1 && before.category == test.Category {
				rank.Comparisons++
				if before.state != state {
					rank.Changes++
				}
			}
			prior[comparison] = previous{index: indices[chain], state: state, category: test.Category}
		}
		if point.Counts.Total == 0 {
			point.Gap = "No matching test was selected in this run."
			if len(run.Tests) == 0 {
				point.Gap = "The selected test snapshot is unavailable."
			}
		} else {
			report.Samples++
			addCounts(&report.Totals, point.Counts)
		}
		report.Points = append(report.Points, point)
	}
	report.ContextRuns = len(report.Points)
	for _, rank := range ranks {
		if rank.Counts.Failed+rank.Counts.TimedOut > 0 {
			report.FailureRankings = append(report.FailureRankings, *rank)
		}
		if rank.Comparisons > 0 {
			rate := float64(rank.Changes) / float64(rank.Comparisons)
			rank.Volatility = &rate
			report.VolatilityRankings = append(report.VolatilityRankings, *rank)
		}
	}
	slices.SortFunc(report.FailureRankings, func(a, b Ranking) int {
		if c := cmp.Compare(b.Counts.Failed+b.Counts.TimedOut, a.Counts.Failed+a.Counts.TimedOut); c != 0 {
			return c
		}
		if c := cmp.Compare(b.Counts.Total, a.Counts.Total); c != 0 {
			return c
		}
		return cmp.Compare(a.Suite+"/"+a.Test, b.Suite+"/"+b.Test)
	})
	slices.SortFunc(report.VolatilityRankings, func(a, b Ranking) int {
		if c := cmp.Compare(*b.Volatility, *a.Volatility); c != 0 {
			return c
		}
		if c := cmp.Compare(b.Comparisons, a.Comparisons); c != 0 {
			return c
		}
		return cmp.Compare(a.Suite+"/"+a.Test, b.Suite+"/"+b.Test)
	})
	report.FailureRankings = report.FailureRankings[:min(20, len(report.FailureRankings))]
	report.VolatilityRankings = report.VolatilityRankings[:min(20, len(report.VolatilityRankings))]
	return report
}
