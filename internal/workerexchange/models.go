package workerexchange

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/artifacts"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

const MaxSubmissionBytes = workerprotocol.MaxSubmissionBytes
const Bucket = "worker-exchanges"

var ErrDenied = workerprotocol.ErrDenied
var ErrInvalid = workerprotocol.ErrInvalid
var ErrQuota = workerprotocol.ErrQuota
var ErrConflict = workerprotocol.ErrConflict
var ErrUnavailable = workerprotocol.ErrUnavailable

type File = workerprotocol.File
type Submission = workerprotocol.Submission
type Packet = workerprotocol.Packet

type Ticket struct {
	Execution string
	Token     string `json:"-"`
	Deadline  time.Time
}

type UploadLease struct {
	Claim string    `json:"claim"`
	Until time.Time `json:"until"`
}

type Record struct {
	CredentialReference string                 `json:"credentialReference,omitempty"`
	Gateway             *GatewayRecord         `json:"gateway,omitempty"`
	Operation           string                 `json:"operation,omitempty"`
	DispatchGated       bool                   `json:"dispatchGated,omitempty"`
	DispatchReady       bool                   `json:"dispatchReady,omitempty"`
	Reference           execution.Reference    `json:"reference"`
	Input               json.RawMessage        `json:"input,omitempty"`
	InputHash           string                 `json:"inputHash"`
	Nonce               string                 `json:"nonce"`
	CreatedAt           time.Time              `json:"createdAt"`
	Deadline            time.Time              `json:"deadline"`
	Instance            string                 `json:"instance,omitempty"`
	StartedAt           *time.Time             `json:"startedAt,omitempty"`
	Submission          *Submission            `json:"submission,omitempty"`
	StagingBytes        int64                  `json:"stagingBytes,omitempty"`
	SubmissionHash      string                 `json:"submissionHash,omitempty"`
	Uploaded            map[string]bool        `json:"uploaded,omitempty"`
	Uploads             map[string]UploadLease `json:"uploads,omitempty"`
	Revoked             bool                   `json:"revoked"`
}

type Service struct {
	Secrets                   secrets.Provider
	Store                     *storage.Store
	Blobs                     artifacts.Store
	Authorize                 func(context.Context, execution.Reference) (workerstate.Access, error)
	AuthorizeDiscoveryCleanup func(context.Context, execution.Reference) error
	uploadOnce                sync.Once
	uploads                   chan struct{}
}
