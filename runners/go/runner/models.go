package runner

import (
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"time"
)

type Input struct {
	Browser *browserpolicy.Target `json:"browser,omitempty"`
	RunID   string                `json:"runId"`
	Test    struct {
		ID               string   `json:"id"`
		Selector         string   `json:"selector"`
		Type             string   `json:"type"`
		RequiredBindings []string `json:"requiredBindings"`
	} `json:"test"`
	Capture struct {
		Logs bool `json:"logs"`
	} `json:"capture"`
	TimeoutMs int `json:"timeoutMs"`
}

type Details struct {
	Package         string  `json:"package"`
	Test            string  `json:"test"`
	Toolchain       string  `json:"toolchain"`
	OS              string  `json:"os"`
	Architecture    string  `json:"architecture"`
	BuildDurationMs float64 `json:"buildDurationMs"`
	BuildExitCode   *int    `json:"buildExitCode,omitempty"`
	ExitCode        *int    `json:"exitCode,omitempty"`
	OutputTruncated bool    `json:"outputTruncated,omitempty"`
}

type Step struct {
	Title      string  `json:"title"`
	Category   string  `json:"category"`
	DurationMs float64 `json:"durationMs"`
	Error      string  `json:"error,omitempty"`
}
type Result struct {
	TestID              string    `json:"testId"`
	State               string    `json:"state"`
	FailureKind         string    `json:"failureKind,omitempty"`
	Error               string    `json:"error,omitempty"`
	StartedAt           time.Time `json:"startedAt"`
	FinishedAt          time.Time `json:"finishedAt"`
	DurationMs          float64   `json:"durationMs"`
	DurationUnavailable bool      `json:"durationUnavailable,omitempty"`
	WorkerDurationMs    float64   `json:"workerDurationMs"`
	RunnerStatus        string    `json:"runnerStatus"`
	Steps               []Step    `json:"steps,omitempty"`
	Go                  Details   `json:"go"`
}
type Attachment struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Stream      string `json:"stream"`
}
type Report struct {
	Schema         string       `json:"schemaVersion"`
	TestID         string       `json:"testId"`
	Selector       string       `json:"selector"`
	Result         Result       `json:"result"`
	Attachments    []Attachment `json:"attachments"`
	EvidenceErrors []string     `json:"evidenceErrors"`
}
