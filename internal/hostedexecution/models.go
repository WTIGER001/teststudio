package hostedexecution

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

var ErrUnavailable = errors.New("hosted worker control is unavailable; its durable reference remains for recovery")
var ErrNotFound = errors.New("hosted worker is authoritatively absent")
var ErrRejected = errors.New("hosted worker creation was rejected before dispatch")
var ErrOwnership = errors.New("hosted worker ownership or immutable execution settings do not match")
var ErrConfiguration = errors.New("hosted execution requires an approved supervised image, private exchange, and durable lifecycle hooks")

type Image struct {
	Discovery        string
	Reference        string
	Runner           string
	Runtime          string
	OS               string
	Architecture     string
	Supervisor       string
	Project          string
	SourceDigest     string
	ExecutionProfile string
}

type Job struct {
	Operation      string
	Reference      execution.Reference
	Image          string
	Runner         string
	ExchangeOrigin string
	Ticket         workerexchange.Ticket `json:"-"`
}

type Status struct {
	AdmissionPending bool
	Reference        execution.Reference
	Image            string
	Deadline         time.Time
	Phase            string
	ExitCode         *int
}

type Provider interface {
	InspectImage(context.Context, string) (Image, error)
	Ensure(context.Context, Job) error
	Observe(context.Context, execution.Reference) (Status, error)
	Stop(context.Context, execution.Reference) error
	Remove(context.Context, execution.Reference) error
	Inventory(context.Context, string, string) ([]execution.Reference, error)
}

type Exchange interface {
	Activate(context.Context, execution.Reference) error
	Prepare(context.Context, execution.Reference, execution.CodedInput, time.Time) (workerexchange.Ticket, error)
	State(context.Context, execution.Reference) (workerexchange.State, error)
	Observation(context.Context, execution.Reference, execution.CodedInput, error) (execution.CodedReport, *time.Time, bool, error)
	Revoke(context.Context, execution.Reference) error
	CheckCleanup(context.Context, execution.Reference) error
	Cleanup(context.Context, execution.Reference) error
}

type Executor struct {
	Provider        Provider
	Exchange        Exchange
	Name            string
	Runner          string
	ExchangeOrigin  string
	ApproveArtifact func(context.Context, catalog.Artifact) error
	PollInterval    time.Duration
}

func (j Job) String() string   { return "hosted job: " + j.Reference.ID }
func (j Job) GoString() string { return j.String() }

var _ execution.ManagedExecutor = (*Executor)(nil)
var _ execution.RuntimeProfiler = (*Executor)(nil)
