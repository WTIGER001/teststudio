package catalog

import (
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/timing"
	"time"
)

type CapturePolicy = capture.Policy

func DefaultCapture() CapturePolicy {
	return capture.DefaultPolicy()
}

type HTTPCheck struct {
	Endpoint           string            `json:"endpoint"`
	Path               string            `json:"path"`
	Method             string            `json:"method"`
	StatusCodes        []int             `json:"statusCodes"`
	RequiredText       string            `json:"requiredText,omitempty"`
	ForbiddenText      string            `json:"forbiddenText,omitempty"`
	JSON               map[string]any    `json:"json,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	MaxResponseMs      int               `json:"maxResponseMs,omitempty"`
	MinCertificateDays int               `json:"minCertificateDays,omitempty"`
}

type Test struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Application      string         `json:"applicationId,omitempty"`
	Category         string         `json:"category"`
	Tags             []string       `json:"tags"`
	Owner            string         `json:"owner,omitempty"`
	Type             string         `json:"type"`
	Runner           string         `json:"runner"`
	State            string         `json:"state"`
	Safety           string         `json:"safety"`
	Rationale        string         `json:"rationale"`
	Conditions       string         `json:"conditions,omitempty"`
	Dependencies     []string       `json:"dependencies"`
	RequiredBindings []string       `json:"requiredBindings"`
	TimeoutSeconds   int            `json:"timeoutSeconds"`
	Source           string         `json:"source,omitempty"`
	Selector         string         `json:"selector,omitempty"`
	Check            *HTTPCheck     `json:"check,omitempty"`
	Timing           *timing.Policy `json:"timing,omitempty"`
	Capture          *CapturePolicy `json:"capture,omitempty"`
}

type Artifact struct {
	Image          string            `json:"image,omitempty"`
	SourceRevision string            `json:"sourceRevision,omitempty"`
	Discovered     map[string]string `json:"discoveredSelectors,omitempty"`
}

type Draft struct {
	ID               string        `json:"id"`
	Tests            []Test        `json:"tests"`
	Artifact         Artifact      `json:"artifact"`
	Timing           timing.Policy `json:"timing"`
	Capture          CapturePolicy `json:"capture"`
	PublishedRelease string        `json:"publishedReleaseId,omitempty"`
}

type Release struct {
	ActivatedOnPublication bool      `json:"activatedOnPublication"`
	ID                     string    `json:"id"`
	Suite                  string    `json:"suiteId"`
	Project                string    `json:"projectId"`
	DraftRevision          int64     `json:"draftRevision"`
	Draft                  Draft     `json:"catalog"`
	Actor                  string    `json:"actor"`
	CreatedAt              time.Time `json:"createdAt"`
}

type Issue struct {
	TestID  string `json:"testId,omitempty"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Revision struct {
	Test  Test      `json:"test"`
	Actor string    `json:"actor"`
	At    time.Time `json:"at"`
}
