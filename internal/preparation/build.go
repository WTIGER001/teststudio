package preparation

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type BuildRequest struct {
	Snapshot  sourcebundle.Snapshot
	Origin    suitespec.Origin
	Reference execution.Reference
}

type Build struct {
	Image        string    `json:"image"`
	SourceDigest string    `json:"sourceDigest"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
	Log          string    `json:"log,omitempty"`
	LogTruncated bool      `json:"logTruncated,omitempty"`
}

type ArtifactBuilder interface {
	Build(context.Context, BuildRequest) (Build, error)
	CleanupBuild(context.Context, execution.Reference) error
}

func LocalSourceProfile(runner, profile string) bool {
	return runner == "go" && (profile == "go-module" || profile == "go-environment") || runner == "playwright" && (profile == "playwright-node" || profile == "playwright-environment")
}

type AssemblyBuilder interface {
	ArtifactBuilder
	CompleteAssembly(context.Context, execution.Reference) error
}
