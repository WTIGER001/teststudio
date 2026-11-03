package preparation

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type InspectionRequest struct {
	Image     string              `json:"image"`
	Suite     string              `json:"suiteId"`
	Reference execution.Reference `json:"reference"`
}

type Inspection struct {
	Image        string              `json:"image"`
	ImageID      string              `json:"localImageId"`
	Runtime      string              `json:"runtime"`
	Discovery    suitespec.Discovery `json:"discovery"`
	StartedAt    time.Time           `json:"startedAt"`
	FinishedAt   time.Time           `json:"finishedAt"`
	Log          string              `json:"log,omitempty"`
	LogTruncated bool                `json:"logTruncated,omitempty"`
}

type ArtifactInspector interface {
	Inspect(context.Context, InspectionRequest) (Inspection, error)
	CleanupInspection(context.Context, InspectionRequest) error
}
