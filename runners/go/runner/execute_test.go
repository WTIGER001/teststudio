package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/gosuite"
)

func TestRealGoSourceExecutionDistinguishesPreparationAndTestOutcomes(t *testing.T) {
	suite := t.TempDir()
	files := map[string]string{
		"go.mod": "module teststudio.local/fresh-checks\n\ngo 1.24\n",
		"unit/check_test.go": `package check
import ("os"; "testing"; "time")
func TestPass(t *testing.T) { t.Log("fresh test ran"); time.Sleep(12*time.Millisecond) }
func TestFail(t *testing.T) { t.Fatal("fresh assertion detail") }
func TestSkip(t *testing.T) { t.Skip("fresh skip reason") }
func TestPanic(t *testing.T) { panic("fresh panic detail") }
func TestTimeout(t *testing.T) { time.Sleep(5*time.Second) }
func TestRows(t *testing.T) {
 t.Run("one",func(t *testing.T){ t.Parallel(); time.Sleep(12*time.Millisecond) })
 t.Run("two",func(t *testing.T){ t.Parallel(); time.Sleep(15*time.Millisecond) })
}
func TestUncached(t *testing.T) {
 f,err:=os.OpenFile("execution-count",os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600)
 if err!=nil {t.Fatal(err)}; defer f.Close(); f.WriteString("run\n")
}
`,
		"broken/check_test.go": "package broken\nimport \"testing\"\nfunc TestBroken(t *testing.T){ undefinedValue() }\n",
	}
	for name, value := range files {
		path := filepath.Join(suite, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(t *testing.T, pkg, name, state, kind string, timeout int) Report {
		t.Helper()
		entry := gosuite.Entry{ID: "GO-1", Package: pkg, Test: name}
		manifest, _ := json.Marshal(gosuite.Manifest{Schema: "teststudio.go.catalog/v1", Tests: []gosuite.Entry{entry}})
		if err := os.WriteFile(filepath.Join(suite, "teststudio-go.json"), manifest, 0600); err != nil {
			t.Fatal(err)
		}
		input := Input{TimeoutMs: timeout}
		input.Test.ID, input.Test.Selector, input.Capture.Logs = entry.ID, entry.Selector(), true
		ctx, cancel := context.WithTimeout(t.Context(), 55*time.Second)
		defer cancel()
		r := Execute(ctx, input, suite, t.TempDir())
		if r.Result.State != state || r.Result.FailureKind != kind || r.Result.Go.Package != pkg || r.Result.Go.Test != name || r.Result.Go.Toolchain == "" || len(r.EvidenceErrors) != 0 {
			t.Fatalf("%s: %+v, evidence: %v", name, r.Result, r.EvidenceErrors)
		}
		if state == "unknown" && (!r.Result.DurationUnavailable || r.Result.DurationMs != 0) {
			t.Fatal("preparation entered test-duration history")
		}
		if len(r.Attachments) == 0 {
			t.Fatal("execution diagnostics missing")
		}
		wire, _ := json.Marshal(r)
		var common execution.CodedReport
		if err := json.Unmarshal(wire, &common); err != nil || common.Schema != "teststudio.go/v1" || common.Result.State != state || common.Selector != entry.Selector() {
			t.Fatalf("shared wire contract: %+v %v", common, err)
		}
		return r
	}
	for _, test := range []struct{ name, pkg, state, kind string }{
		{"TestPass", "./unit", "passed", ""},
		{"TestFail", "./unit", "failed", "assertion"},
		{"TestSkip", "./unit", "skipped", ""},
		{"TestPanic", "./unit", "failed", "panic"},
		{"TestTimeout", "./unit", "timed-out", "timeout"},
		{"TestMissing", "./unit", "unknown", "discovery"},
		{"TestBroken", "./broken", "unknown", "build"},
	} {
		t.Run(test.name, func(t *testing.T) { run(t, test.pkg, test.name, test.state, test.kind, 200) })
	}
	t.Run("aggregate", func(t *testing.T) {
		r := run(t, "./unit", "TestRows", "passed", "", 1000)
		if len(r.Result.Steps) != 3 {
			t.Fatalf("subtest evidence: %+v", r.Result.Steps)
		}
	})
	t.Run("independent subtest", func(t *testing.T) {
		r := run(t, "./unit", "TestRows/two", "passed", "", 1000)
		if len(r.Result.Steps) != 1 || r.Result.Steps[0].Title != "TestRows/two" {
			t.Fatalf("overlapping logical result: %+v", r)
		}
	})
	t.Run("uncached", func(t *testing.T) {
		for range 2 {
			run(t, "./unit", "TestUncached", "passed", "", 1000)
		}
		data, err := os.ReadFile(filepath.Join(suite, "unit", "execution-count"))
		if err != nil || strings.Count(string(data), "run\n") != 2 {
			t.Fatalf("cached test execution: %q %v", data, err)
		}
	})
}
