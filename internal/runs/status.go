package runs

import (
	"slices"

	"github.com/wtiger001/teststudio/internal/execution"
)

func NormalizeTimeouts(run Run) Run {
	changed := false
	results := slices.Clone(run.Results)
	for i, result := range results {
		results[i] = execution.NormalizeTimeout(result)
		changed = changed || results[i].State != result.State
	}
	if changed {
		run.Results = results
		run.Counts = Count(len(run.Tests), results)
		if run.Outcome == "failed" && run.Counts.Failed == 0 && run.Counts.TimedOut > 0 {
			run.Outcome = "timed-out"
		}
	}
	return run
}
