package evidence

import "time"

type Artifact struct {
	ID          string    `json:"id"`
	Project     string    `json:"projectId"`
	Run         string    `json:"runId"`
	Test        string    `json:"testId,omitempty"`
	Attempt     int       `json:"attempt,omitempty"`
	Worker      string    `json:"workerId,omitempty"`
	Sequence    int64     `json:"sequence,omitempty"`
	Stream      string    `json:"stream,omitempty"`
	Kind        string    `json:"kind"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	State       string    `json:"state"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
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
