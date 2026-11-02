package execution

import (
	"context"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"time"
)

type Step struct {
	Title      string  `json:"title"`
	Category   string  `json:"category"`
	DurationMs float64 `json:"durationMs"`
	Error      string  `json:"error,omitempty"`
}

type Attachment struct {
	Stream      string `json:"stream,omitempty"`
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Data        []byte `json:"-"`
}

type CodedInput struct {
	Reference       *Reference              `json:"-"`
	BeforeDispatch  func() error            `json:"-"`
	BeforeCleanup   func(CodedReport) error `json:"-"`
	AfterCleanup    func(error) error       `json:"-"`
	OnStart         func(time.Time) error   `json:"-"`
	RunID           string                  `json:"runId"`
	Test            catalog.Test            `json:"test"`
	Artifact        catalog.Artifact        `json:"artifact"`
	Capture         capture.Policy          `json:"capture"`
	TimeoutMs       int                     `json:"timeoutMs"`
	FailedBefore    bool                    `json:"failedBefore"`
	SlowThresholdMs *float64                `json:"slowThresholdMs,omitempty"`
	TimingMode      string                  `json:"timingMode"`
}

type CodedReport struct {
	RecoveryError  error        `json:"-"`
	Schema         string       `json:"schemaVersion"`
	TestID         string       `json:"testId"`
	Selector       string       `json:"selector"`
	Result         Result       `json:"result"`
	Attachments    []Attachment `json:"attachments"`
	EvidenceErrors []string     `json:"evidenceErrors"`
}

type CodedExecutor interface {
	Profile() string
	Admit(context.Context, catalog.Test, catalog.Artifact) error
	Execute(context.Context, CodedInput) CodedReport
}

type RuntimeProfiler interface {
	TimingProfile(context.Context, catalog.Artifact) (string, error)
}
