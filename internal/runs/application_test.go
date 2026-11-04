package runs

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
)

func TestApplicationScopeUsesFrozenRunAndSelectedTests(t *testing.T) {
	run := Run{Applications: []string{"shop", "warehouse"}, Tests: []catalog.Test{{ID: "shop-test", Application: "shop"}, {ID: "warehouse-test", Application: "warehouse"}, {ID: "shared"}}, Results: []execution.Result{{TestID: "shop-test", State: "passed"}, {TestID: "warehouse-test", State: "failed"}, {TestID: "shared", State: "timed-out"}}}
	if !InApplication(run, "shop") || !InApplication(run, "warehouse") || InApplication(run, "unknown") {
		t.Fatal("frozen applications were not respected")
	}
	summary := summarizeApplication(run, "shop")
	if !summary.Shared || summary.Counts.Total != 2 || summary.Counts.Passed != 1 || summary.Counts.TimedOut != 1 || summary.Counts.Failed != 0 {
		t.Fatal(summary)
	}
	run.Environment = &projects.Environment{Application: "shop"}
	if InApplication(run, "warehouse") || TestInApplication(run, run.Tests[1], "shop") {
		t.Fatal("environment scope included a foreign application")
	}
	run.Environment = nil
	run.Request.Application = "shop"
	if InApplication(run, "warehouse") {
		t.Fatal("explicit request scope included a foreign application")
	}
	run.Request.Application = ""
	run.Applications = nil
	run.Tests = []catalog.Test{{ID: "legacy"}}
	if InApplication(run, "shop") {
		t.Fatal("unknown legacy application was invented")
	}
}
