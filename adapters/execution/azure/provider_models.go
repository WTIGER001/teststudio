package azureworker

import (
	"context"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

const Bucket = "azure-dispatches"

type Jobs interface {
	Get(context.Context, string) (a.Job, error)
	Create(context.Context, string, a.Job) error
	Start(context.Context, string, *a.JobsClientBeginStartOptions) (*a.JobExecutionBase, error)
	Executions(context.Context, string) ([]a.JobExecution, error)
	Stop(context.Context, string, string) error
	Delete(context.Context, string) error
}

type Inspector interface {
	InspectImage(context.Context, string) (hostedexecution.Image, error)
}

type Network interface {
	Admit(context.Context, Config) error
	VerifyExecution(context.Context, Config, string, a.JobExecution) error
}

type DeadlineSpec struct {
	Reference     execution.Reference
	ResourceID    string
	At            time.Time
	Specification string
}

type Deadlines interface {
	Ensure(context.Context, DeadlineSpec) error
	Verify(context.Context, DeadlineSpec) error
	Remove(context.Context, DeadlineSpec) error
}

type Options struct {
	Config           Config
	Store            *storage.Store
	Jobs             Jobs
	Inspector        Inspector
	Network          Network
	Deadlines        Deadlines
	ResumeDispatch   func(context.Context, execution.Reference) (workerexchange.Ticket, error)
	DiscoveryCleanup func(context.Context, execution.Reference) error
	Now              func() time.Time
}

type Provider struct {
	config           Config
	store            *storage.Store
	jobs             Jobs
	inspector        Inspector
	network          Network
	deadlines        Deadlines
	resumeDispatch   func(context.Context, execution.Reference) (workerexchange.Ticket, error)
	discoveryCleanup func(context.Context, execution.Reference) error
	now              func() time.Time
}

type dispatch struct {
	Proof             Proof                   `json:"proof"`
	CreatedAt         time.Time               `json:"createdAt"`
	CreateAttempted   bool                    `json:"createAttempted"`
	CreateRejected    bool                    `json:"createRejected,omitempty"`
	CreationBlocked   bool                    `json:"creationBlocked,omitempty"`
	DefinitionSeen    bool                    `json:"definitionSeen,omitempty"`
	DeadlineAttempted bool                    `json:"deadlineAttempted,omitempty"`
	DeadlineVerified  bool                    `json:"deadlineVerified,omitempty"`
	StartAttempted    bool                    `json:"startAttempted,omitempty"`
	StartRejected     bool                    `json:"startRejected,omitempty"`
	Receipt           *a.JobExecutionBase     `json:"receipt,omitempty"`
	Stopping          bool                    `json:"stopping,omitempty"`
	Deleting          bool                    `json:"deleting,omitempty"`
	DeleteCheckAfter  time.Time               `json:"deleteCheckAfter,omitempty"`
	Removed           bool                    `json:"removed,omitempty"`
	Terminal          *hostedexecution.Status `json:"terminal,omitempty"`
}

func New(options Options) (*Provider, error) {
	if Validate(options.Config) != nil || options.Store == nil || options.Jobs == nil || options.Inspector == nil || options.Network == nil || options.Deadlines == nil || options.ResumeDispatch == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Provider{config: options.Config, store: options.Store, jobs: options.Jobs, inspector: options.Inspector, network: options.Network, deadlines: options.Deadlines, resumeDispatch: options.ResumeDispatch, discoveryCleanup: options.DiscoveryCleanup, now: now}, nil
}

func (p *Provider) InspectImage(ctx context.Context, reference string) (hostedexecution.Image, error) {
	image, err := p.inspector.InspectImage(ctx, reference)
	if err != nil {
		return image, hostedexecution.ErrUnavailable
	}
	if image.Reference != reference || image.OS != "linux" || image.Architecture != "amd64" || image.Supervisor != "worker-exchange/v1" {
		return image, hostedexecution.ErrConfiguration
	}
	image.ExecutionProfile, err = ExecutionProfile(p.config)
	return image, err
}

func (p *Plan) deadlineSpec() DeadlineSpec {
	return DeadlineSpec{Reference: p.job.Reference, ResourceID: p.ResourceID(), At: p.job.Ticket.Deadline, Specification: p.fingerprint}
}

var _ hostedexecution.Provider = (*Provider)(nil)
