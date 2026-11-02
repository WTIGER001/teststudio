package evidence

import (
	"github.com/wtiger001/teststudio/internal/capture"
	"time"
)

type Artifact struct {
	ID           string          `json:"id"`
	Project      string          `json:"projectId"`
	Run          string          `json:"runId"`
	Test         string          `json:"testId,omitempty"`
	Attempt      int             `json:"attempt,omitempty"`
	Worker       string          `json:"workerId,omitempty"`
	Sequence     int64           `json:"sequence,omitempty"`
	Stream       string          `json:"stream,omitempty"`
	Kind         string          `json:"kind"`
	Filename     string          `json:"filename"`
	ContentType  string          `json:"contentType"`
	Size         int64           `json:"size"`
	Checksum     string          `json:"checksum"`
	State        string          `json:"state"`
	Error        string          `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	ExpiresAt    time.Time       `json:"expiresAt"`
	Step         string          `json:"stepId,omitempty"`
	Policy       *capture.Policy `json:"capturePolicy,omitempty"`
	QuotaVersion int             `json:"quotaVersion,omitempty"`
}

type Event struct {
	ID       string    `json:"id"`
	Run      string    `json:"runId"`
	Project  string    `json:"projectId"`
	Sequence int64     `json:"sequence"`
	Kind     string    `json:"kind"`
	Test     string    `json:"testId,omitempty"`
	Attempt  int       `json:"attempt,omitempty"`
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
}

type Quota struct {
	Bytes int64 `json:"reservedBytes"`
	Limit int64 `json:"limitBytes"`
}

type Budget struct {
	RunBytes     int64
	TestBytes    int64
	FileBytes    int64
	ProjectBytes int64
}

type LogInput struct {
	Project  string
	Run      string
	Worker   string
	Stream   string
	Test     string
	Attempt  int
	Sequence int64
	Text     string
	Policy   capture.Policy
	Budget   Budget
}
