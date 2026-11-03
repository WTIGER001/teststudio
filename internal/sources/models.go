package sources

import (
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/storage"
)

const ContentLimit = 1 << 20
const StagingLifetime = 30 * time.Minute

type UploadInput struct {
	Name    string `json:"name"`
	Format  string `json:"format"`
	Content string `json:"content"`
}

type Upload struct {
	ID        string          `json:"id"`
	Project   string          `json:"projectId"`
	Suite     string          `json:"suiteId"`
	Name      string          `json:"name"`
	Format    string          `json:"format"`
	Digest    string          `json:"digest"`
	Bytes     int             `json:"bytes"`
	Actor     string          `json:"actor"`
	State     string          `json:"state"`
	PlanID    string          `json:"planId,omitempty"`
	Issues    []catalog.Issue `json:"issues"`
	CreatedAt time.Time       `json:"createdAt"`
	ExpiresAt time.Time       `json:"expiresAt"`
	ReleaseID string          `json:"releaseId,omitempty"`
	Activated bool            `json:"activated"`
}

type Preview struct {
	Upload storage.Record      `json:"upload"`
	Plan   *catalog.ImportPlan `json:"plan,omitempty"`
}

type Content struct {
	Content string `json:"content"`
	Digest  string `json:"digest"`
}

func Bucket(project, suite string) string { return "source-uploads/" + project + "/" + suite }
