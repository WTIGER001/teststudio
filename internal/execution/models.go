package execution

import (
	"github.com/wtiger001/teststudio/internal/timing"
	"time"
)

type HTTPObservation struct {
	TrustBundle        *HTTPTrustBundle   `json:"trustBundle,omitempty"`
	Status             int                `json:"status"`
	URL                string             `json:"url"`
	DurationMs         float64            `json:"durationMs"`
	Timings            map[string]float64 `json:"timingsMs,omitempty"`
	CertificateExpires *time.Time         `json:"certificateExpires,omitempty"`
}

type GoObservation struct {
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

type Result struct {
	DiagnosticsPolicy   string             `json:"diagnosticsPolicy,omitempty"`
	DurationUnavailable bool               `json:"durationUnavailable,omitempty"`
	WorkerDurationMs    float64            `json:"workerDurationMs,omitempty"`
	ExecutionID         string             `json:"executionId,omitempty"`
	Recovered           bool               `json:"recovered,omitempty"`
	Steps               []Step             `json:"steps,omitempty"`
	RunnerStatus        string             `json:"runnerStatus,omitempty"`
	ExpectedStatus      string             `json:"expectedStatus,omitempty"`
	ObservedState       string             `json:"observedState,omitempty"`
	TotalDurationMs     float64            `json:"totalDurationMs"`
	TimingRegressions   int                `json:"timingRegressions"`
	Timing              *timing.Assessment `json:"timing,omitempty"`
	TestID              string             `json:"testId"`
	Attempt             int                `json:"attempt"`
	State               string             `json:"state"`
	FailureKind         string             `json:"failureKind,omitempty"`
	Error               string             `json:"error,omitempty"`
	StartedAt           time.Time          `json:"startedAt"`
	FinishedAt          time.Time          `json:"finishedAt"`
	DurationMs          float64            `json:"durationMs"`
	HTTP                *HTTPObservation   `json:"http,omitempty"`
	Go                  *GoObservation     `json:"go,omitempty"`
}
