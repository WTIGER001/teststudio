package runs

import (
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/timing"
)

type Request struct {
	Suite          string          `json:"suiteId"`
	Environment    string          `json:"environmentId,omitempty"`
	Release        string          `json:"releaseId,omitempty"`
	IDs            []string        `json:"testIds,omitempty"`
	Location       string          `json:"executionLocation,omitempty"`
	TimeoutSeconds int             `json:"timeoutSeconds,omitempty"`
	Retries        int             `json:"retries,omitempty"`
	Parent         string          `json:"parentRunId,omitempty"`
	Capture        *capture.Policy `json:"capture,omitempty"`
}

type Counts struct {
	Total     int `json:"total"`
	Passed    int `json:"passed"`
	Failed    int `json:"failed"`
	Flaky     int `json:"flaky"`
	Skipped   int `json:"skipped"`
	Completed int `json:"completed"`
}

type Run struct {
	ID                   string                    `json:"id"`
	Project              string                    `json:"projectId"`
	Request              Request                   `json:"request"`
	Release              catalog.Release           `json:"release"`
	Environment          *projects.Environment     `json:"environment,omitempty"`
	Tests                []catalog.Test            `json:"tests"`
	Capture              catalog.CapturePolicy     `json:"capture"`
	CaptureLimits        capture.Limits            `json:"captureLimits"`
	TestCapture          map[string]capture.Policy `json:"testCapture"`
	Timing               timing.Estimate           `json:"timing"`
	CurrentTestStartedAt *time.Time                `json:"currentTestStartedAt,omitempty"`
	CurrentAttempt       int                       `json:"currentAttempt,omitempty"`
	Actor                string                    `json:"actor"`
	Trigger              string                    `json:"trigger"`
	State                string                    `json:"state"`
	Outcome              string                    `json:"outcome"`
	Phase                string                    `json:"phase"`
	CurrentTest          string                    `json:"currentTestId,omitempty"`
	Counts               Counts                    `json:"counts"`
	Results              []execution.Result        `json:"results"`
	CreatedAt            time.Time                 `json:"createdAt"`
	UpdatedAt            time.Time                 `json:"updatedAt"`
	StartedAt            *time.Time                `json:"startedAt,omitempty"`
	FinishedAt           *time.Time                `json:"finishedAt,omitempty"`
	Deadline             time.Time                 `json:"deadline"`
	Worker               string                    `json:"workerId,omitempty"`
	LeaseUntil           *time.Time                `json:"leaseUntil,omitempty"`
	CancelRequested      bool                      `json:"cancelRequested"`
	Error                string                    `json:"error,omitempty"`
	EvidenceState        string                    `json:"evidenceState"`
	EvidenceErrors       []string                  `json:"evidenceErrors"`
}

func Terminal(state string) bool {
	return state == "completed" || state == "cancelled" || state == "timed-out" || state == "blocked" || state == "incomplete"
}

func Count(tests int, results []execution.Result) Counts {
	c := Counts{Total: tests}
	for _, r := range results {
		switch r.State {
		case "passed":
			c.Passed++
		case "failed":
			c.Failed++
		case "flaky":
			c.Flaky++
		case "skipped":
			c.Skipped++
		default:
			continue
		}
		c.Completed++
	}
	return c
}
