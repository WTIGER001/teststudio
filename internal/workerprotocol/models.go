package workerprotocol

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
)

const MaxTestSubmissionBytes = (2 << 20) + (64 << 10)

const MaxSubmissionBytes = (8 << 20) + (256 << 10)

var ErrDenied = errors.New("worker exchange is unavailable")
var ErrInvalid = errors.New("worker message does not match its bounded attempt contract")
var ErrQuota = errors.New("worker exchange staging quota is exhausted")
var ErrConflict = errors.New("worker message conflicts with the accepted attempt")
var ErrUnavailable = errors.New("worker exchange storage is unavailable")

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Submission struct {
	ProcessExitCode       int                   `json:"processExitCode"`
	InfrastructureFailure string                `json:"infrastructureFailure,omitempty"`
	Report                execution.CodedReport `json:"report,omitzero"`
	Discovery             *DiscoveryReport      `json:"discovery,omitempty"`
	Files                 []File                `json:"files"`
}

type Packet struct {
	Operation string          `json:"operation,omitempty"`
	Execution string          `json:"executionId"`
	Deadline  time.Time       `json:"deadline"`
	Input     json.RawMessage `json:"input"`
}
