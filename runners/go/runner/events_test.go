package runner

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/gosuite"
)

func feed(e *Events, action, name string, elapsed float64, output string) {
	b, _ := json.Marshal(Event{Time: time.Now().UTC(), Package: "./unit", Test: name, Action: action, Elapsed: elapsed, Output: output})
	e.Line(string(b))
}

func TestGoEventsCountOnlyTheExactRegisteredTestAndRetainSubtests(t *testing.T) {
	starts := 0
	e := NewEvents(gosuite.Entry{Package: "./unit", Test: "TestRows/b"}, func(time.Time) { starts++ })
	feed(e, "run", "TestRows", 0, "")
	feed(e, "pass", "TestRows/a", 0.2, "")
	feed(e, "run", "TestRows/b", 0, "")
	feed(e, "run", "TestRows/b/parallel", 0, "")
	feed(e, "pass", "TestRows/b/parallel", 0.1, "")
	feed(e, "pass", "TestRows/b", 0.12, "")
	feed(e, "pass", "TestRows", 0.4, "")
	feed(e, "pass", "", 0.5, "")
	r := e.Result("GO-1", 0, false, false)
	if r.State != "passed" || r.DurationUnavailable || r.DurationMs != 120 || len(r.Steps) != 2 || starts != 1 {
		t.Fatalf("logical selection: %+v, starts %d", r, starts)
	}
}

func TestGoEventFailuresCannotBecomePasses(t *testing.T) {
	for _, test := range []struct {
		name, terminal, output, state, kind string
		exit                                int
		cancelled, deadline                 bool
	}{
		{"assertion", "fail", "assertion details", "failed", "assertion", 1, false, false},
		{"panic", "", "panic: unexpected state", "failed", "panic", 1, false, false},
		{"timeout", "", "panic: test timed out after 10ms", "timed-out", "timeout", 1, false, false},
		{"deadline", "", "", "timed-out", "timeout", -1, false, true},
		{"interrupted", "", "", "unknown", "cancelled", -1, true, false},
		{"missing terminal", "", "", "unknown", "worker-report", 0, false, false},
		{"package failure", "pass", "", "failed", "package", 1, false, false},
		{"skip", "skip", "", "skipped", "", 0, false, false},
		{"panic text in successful output", "pass", "panic: test timed out after 10ms", "passed", "", 0, false, false},
		{"panic text in assertion log", "fail", "    check_test.go:10: panic: test timed out after 10ms", "failed", "assertion", 1, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := NewEvents(gosuite.Entry{Package: "./unit", Test: "TestRows"}, nil)
			feed(e, "run", "TestRows", 0, "")
			feed(e, "output", "TestRows", 0, test.output)
			if test.terminal != "" {
				feed(e, test.terminal, "TestRows", 0.025, "")
				packageState := "pass"
				if test.exit != 0 {
					packageState = "fail"
				}
				feed(e, packageState, "", 0.03, "")
			}
			r := e.Result("GO-1", test.exit, test.cancelled, test.deadline)
			if r.State != test.state || r.FailureKind != test.kind || (r.State == "unknown" && (!r.DurationUnavailable || r.DurationMs != 0)) {
				t.Fatalf("classification: %+v", r)
			}
		})
	}
	for _, scenario := range []string{"package only", "duplicate start", "duplicate terminal", "corrupt JSON"} {
		e := NewEvents(gosuite.Entry{Package: "./unit", Test: "TestRows"}, nil)
		switch scenario {
		case "duplicate start":
			feed(e, "run", "TestRows", 0, "")
			feed(e, "run", "TestRows", 0, "")
		case "duplicate terminal":
			feed(e, "run", "TestRows", 0, "")
			feed(e, "pass", "TestRows", 0, "")
			feed(e, "pass", "TestRows", 0, "")
		case "corrupt JSON":
			e.Line("broken")
		}
		feed(e, "pass", "", 0, "")
		if r := e.Result("GO-1", 0, false, false); r.State != "unknown" || !r.DurationUnavailable {
			t.Fatalf("%s became pass: %+v", scenario, r)
		}
	}
}

func TestGoOutputDrainsAndParsesAfterTheRawCaptureLimit(t *testing.T) {
	lines := 0
	o := output{onLine: func(string) { lines++ }}
	data := []byte(strings.Repeat("a\n", 2<<20))
	n, err := o.Write(data)
	if err != nil || n != len(data) || o.buffer.Len() != 2<<20 || !o.truncated || o.oversized || lines != 2<<20 {
		t.Fatal("raw output truncation stopped event processing")
	}
	_, _ = o.Write([]byte(strings.Repeat("b", (256<<10)+1) + "\nfinished\n"))
	if !o.oversized || o.discard || len(o.line) != 0 || lines != (2<<20)+1 {
		t.Fatal("oversized event was hidden or prevented later events")
	}
}
