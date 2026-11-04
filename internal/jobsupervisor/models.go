package jobsupervisor

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

var ErrWork = errors.New("hosted worker input is unavailable or invalid")
var ErrDelivery = errors.New("hosted worker report delivery did not complete")
var ErrRunner = errors.New("hosted worker runner could not be initialized")

type Transport interface {
	Input(context.Context) (workerprotocol.Packet, error)
	Started(context.Context, time.Time) error
	Submit(context.Context, workerprotocol.Submission) error
	Upload(context.Context, string, io.Reader) error
}

type File struct {
	Descriptor workerprotocol.File
	Open       func() (io.ReadCloser, error)
}

type Prepared struct {
	Submission workerprotocol.Submission
	Files      []File
	Cleanup    func()
}

type Runner interface {
	Run(context.Context, execution.CodedInput, []byte, func(time.Time)) (Prepared, error)
}

type DiscoveryRunner interface {
	Discover(context.Context, workerprotocol.DiscoveryInput, func(time.Time)) (Prepared, error)
}

type Supervisor struct {
	Operation  string
	Transport  Transport
	Runner     Runner
	Attempts   int
	RetryDelay time.Duration
}

type Outcome struct {
	EvidenceIncomplete bool
	ReportAccepted     bool
	PendingAttachments int
	ProcessExitCode    int
}
