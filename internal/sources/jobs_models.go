package sources

import (
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
)

type PrebuiltInput struct {
	Image  string `json:"image"`
	Runner string `json:"runner"`
}

type Job struct {
	ImageID         string               `json:"localImageId,omitempty"`
	Runtime         string               `json:"runtime,omitempty"`
	Release         string               `json:"releaseId,omitempty"`
	CleanupPending  bool                 `json:"cleanupPending"`
	ID              string               `json:"id"`
	Project         string               `json:"projectId"`
	Suite           string               `json:"suiteId"`
	Actor           string               `json:"actor"`
	Image           string               `json:"image"`
	Runner          string               `json:"runner"`
	Location        string               `json:"executionLocation"`
	State           string               `json:"state"`
	Phase           string               `json:"phase"`
	CancelRequested bool                 `json:"cancelRequested"`
	Reference       *execution.Reference `json:"reference,omitempty"`
	Worker          string               `json:"worker,omitempty"`
	LeaseUntil      *time.Time           `json:"leaseUntil,omitempty"`
	Upload          string               `json:"uploadId,omitempty"`
	Error           string               `json:"error,omitempty"`
	Log             string               `json:"log,omitempty"`
	LogTruncated    bool                 `json:"logTruncated,omitempty"`
	CreatedAt       time.Time            `json:"createdAt"`
	StartedAt       *time.Time           `json:"startedAt,omitempty"`
	FinishedAt      *time.Time           `json:"finishedAt,omitempty"`
}

type jobIndex struct {
	Job      string `json:"jobId"`
	Suite    string `json:"suiteId"`
	Location string `json:"executionLocation"`
}

func JobBucket(project, suite string) string { return "source-jobs/" + project + "/" + suite }
func terminalJob(state string) bool {
	return state == "published" || state == "ready" || state == "failed" || state == "cancelled"
}

func queueBucket(location string) string { return "preparation-queue/" + location }
func leaseBucket(location string) string { return "preparation-leases/" + location }
func queueID(job Job) string {
	return job.CreatedAt.UTC().Format("20060102T150405.000000000") + "-" + job.ID
}
