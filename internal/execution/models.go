package execution

import "time"

type HTTPObservation struct {
	Status             int                `json:"status"`
	URL                string             `json:"url"`
	DurationMs         float64            `json:"durationMs"`
	Timings            map[string]float64 `json:"timingsMs,omitempty"`
	CertificateExpires *time.Time         `json:"certificateExpires,omitempty"`
}

type Result struct {
	TestID      string           `json:"testId"`
	Attempt     int              `json:"attempt"`
	State       string           `json:"state"`
	FailureKind string           `json:"failureKind,omitempty"`
	Error       string           `json:"error,omitempty"`
	StartedAt   time.Time        `json:"startedAt"`
	FinishedAt  time.Time        `json:"finishedAt"`
	DurationMs  float64          `json:"durationMs"`
	HTTP        *HTTPObservation `json:"http,omitempty"`
}
