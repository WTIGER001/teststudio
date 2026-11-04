package workerprotocol

import (
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerwire"
)

const MaxTestSubmissionBytes = workerwire.MaxTestSubmissionBytes

const MaxSubmissionBytes = workerwire.MaxSubmissionBytes

var ErrDenied = workerwire.ErrDenied
var ErrInvalid = workerwire.ErrInvalid
var ErrQuota = workerwire.ErrQuota
var ErrConflict = workerwire.ErrConflict
var ErrUnavailable = workerwire.ErrUnavailable

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

type Packet = workerwire.Packet
