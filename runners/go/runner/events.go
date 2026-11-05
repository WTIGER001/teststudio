package runner

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/gosuite"
)

type Event struct {
	Time                          time.Time
	Action, Package, Test, Output string
	Elapsed                       float64
}

type Events struct {
	entry           gosuite.Entry
	OnStart         func(time.Time)
	Started         bool
	Finished        bool
	Start           time.Time
	Elapsed         float64
	Finish          time.Time
	Status          string
	PackageFailed   bool
	PackageFinished bool
	Panic           bool
	Timeout         bool
	Problem         string
	Steps           []Step
	tail            string
	clock           time.Time
}

func NewEvents(entry gosuite.Entry, onStart func(time.Time)) *Events {
	return &Events{entry: entry, OnStart: onStart, Steps: []Step{}}
}
func (e *Events) Line(line string) {
	var event Event
	if json.Unmarshal([]byte(line), &event) != nil {
		e.Problem = "Go event stream contains invalid JSON."
		return
	}
	if event.Package != e.entry.Package {
		return
	}
	if event.Output != "" && (event.Test == "" || event.Test == e.entry.Test || strings.HasPrefix(event.Test, e.entry.Test+"/")) {
		e.tail += event.Output
		if len(e.tail) > 8192 {
			e.tail = e.tail[len(e.tail)-8192:]
		}
		e.Panic = e.Panic || strings.HasPrefix(event.Output, "panic:")
		e.Timeout = e.Timeout || strings.HasPrefix(event.Output, "panic: test timed out after")
	}
	if event.Test == "" {
		if event.Action == "pass" || event.Action == "fail" || event.Action == "skip" {
			if e.PackageFinished {
				e.Problem = "The Go package produced duplicate terminal observations."
			}
			e.PackageFinished = true
		}
		e.PackageFailed = e.PackageFailed || event.Action == "fail"
		return
	}
	if event.Test == e.entry.Test && event.Action == "run" {
		if e.Started {
			e.Problem = "The selected Go test started more than once."
			return
		}
		e.Started = true
		e.clock = time.Now()
		e.Start = event.Time
		if e.Start.IsZero() {
			e.Start = time.Now().UTC()
		}
		if e.OnStart != nil {
			e.OnStart(e.Start)
		}
	}
	if event.Action != "pass" && event.Action != "fail" && event.Action != "skip" {
		return
	}
	if math.IsNaN(event.Elapsed) || math.IsInf(event.Elapsed, 0) || event.Elapsed < 0 {
		e.Problem = "Go event duration is invalid."
		return
	}
	duration := event.Elapsed * 1000
	if event.Test == e.entry.Test && duration == 0 && !e.Start.IsZero() && event.Time.After(e.Start) {
		duration = float64(event.Time.Sub(e.Start)) / float64(time.Millisecond)
	}
	if event.Test == e.entry.Test || strings.HasPrefix(event.Test, e.entry.Test+"/") {
		if len(e.Steps) < 500 {
			step := Step{Title: bounded(event.Test, 2048), Category: "go.test", DurationMs: duration}
			if event.Action == "fail" {
				step.Error = "Go test failed. See the retained event log."
			}
			e.Steps = append(e.Steps, step)
		}
	}
	if event.Test == e.entry.Test {
		if !e.Started {
			e.Problem = "The selected Go test finished before it started."
		}
		if e.Finished {
			e.Problem = "The selected Go test produced more than one terminal observation."
			return
		}
		e.Finished = true
		e.Status = event.Action
		e.Elapsed = duration
		e.Finish = event.Time
	}
}

func (e *Events) Result(id string, exit int, cancelled, deadline bool) Result {
	now := time.Now().UTC()
	r := Result{TestID: id, State: "unknown", FailureKind: "worker", Error: e.Problem, StartedAt: now, FinishedAt: now, DurationUnavailable: true, RunnerStatus: "incomplete", Steps: e.Steps}
	if e.Started {
		r.StartedAt = e.Start
		r.DurationUnavailable = false
		r.DurationMs = float64(time.Since(e.clock)) / float64(time.Millisecond)
	}
	if e.Finished {
		r.DurationMs = e.Elapsed
		if !e.Finish.IsZero() && !e.Finish.Before(r.StartedAt) {
			r.FinishedAt = e.Finish
		}
	}
	switch {
	case cancelled:
		r.FailureKind = "cancelled"
		r.RunnerStatus = "cancelled"
		r.Error = "Go execution was interrupted."
	case deadline || (e.Timeout && exit != 0):
		r.State = "timed-out"
		r.FailureKind = "timeout"
		r.RunnerStatus = "timeout"
		r.Error = "Go execution exceeded its timeout."
	case e.Problem != "":
		r.FailureKind = "worker-report"
	case !e.Started && exit != 0:
		r.FailureKind = "worker-launch"
		r.Error = "The Go test process failed before the selected test started. " + e.tail
	case !e.Started:
		r.FailureKind = "discovery"
		r.RunnerStatus = "no-tests"
		r.Error = "The registered Go test did not execute."
	case e.Panic && exit != 0:
		r.State = "failed"
		r.FailureKind = "panic"
		r.RunnerStatus = "panic"
		r.Error = e.tail
	case !e.Finished || !e.PackageFinished:
		r.FailureKind = "worker-report"
		r.Error = "The selected Go test has no complete terminal observation."
	case e.Status == "fail":
		r.State = "failed"
		r.FailureKind = "assertion"
		r.RunnerStatus = "fail"
		r.Error = e.tail
	case exit != 0 || e.PackageFailed:
		r.State = "failed"
		r.FailureKind = "package"
		r.RunnerStatus = "package-failed"
		r.Error = e.tail
	case e.Status == "skip":
		r.State = "skipped"
		r.FailureKind = ""
		r.RunnerStatus = "skip"
		r.Error = ""
	default:
		r.State = "passed"
		r.FailureKind = ""
		r.RunnerStatus = "pass"
		r.Error = ""
	}
	if r.State == "unknown" {
		r.DurationUnavailable = true
		r.WorkerDurationMs = r.DurationMs
		r.DurationMs = 0
	}
	return r
}
