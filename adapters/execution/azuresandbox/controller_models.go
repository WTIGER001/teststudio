package azuresandbox

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

const DispatchBucket = "azure-sandbox-dispatches"

type Instances interface {
	Create(context.Context, *Plan) (Snapshot, error)
	Read(context.Context, string) (Snapshot, error)
	Inventory(context.Context) ([]Snapshot, error)
	Stop(context.Context, string) error
	Delete(context.Context, string) error
}

type DeadlineSpec struct {
	Config Config `json:"config"`
	Proof  Proof  `json:"proof"`
}

type Deadlines interface {
	Ensure(context.Context, DeadlineSpec) error
	Verify(context.Context, DeadlineSpec) error
	Remove(context.Context, DeadlineSpec) error
}

type Admission interface {
	BeforeCreate(context.Context, Config) error
	VerifyInstance(context.Context, Config, Snapshot) error
}

type ControllerOptions struct {
	Config    Config
	Instances Instances
	Deadlines Deadlines
	Admission Admission
	Exchange  *workerexchange.Service
	Now       func() time.Time
}

type Controller struct {
	config    Config
	instances Instances
	deadlines Deadlines
	admission Admission
	exchange  *workerexchange.Service
	store     *storage.Store
	now       func() time.Time
}

type DispatchState struct {
	Reference        execution.Reference `json:"reference"`
	Instance         Snapshot            `json:"instance"`
	AdmissionPending bool                `json:"admissionPending"`
	CreateAttempted  bool                `json:"createAttempted"`
	Rejected         bool                `json:"rejected"`
	Quiesced         bool                `json:"quiesced"`
	Removed          bool                `json:"removed"`
}

type dispatch struct {
	Proof             Proof     `json:"proof"`
	InputHash         string    `json:"inputHash"`
	CreatedAt         time.Time `json:"createdAt"`
	DeadlineAttempted bool      `json:"deadlineAttempted,omitempty"`
	DeadlineVerified  bool      `json:"deadlineVerified,omitempty"`
	CreateAttempted   bool      `json:"createAttempted,omitempty"`
	CreateRejected    bool      `json:"createRejected,omitempty"`
	CreationBlocked   bool      `json:"creationBlocked,omitempty"`
	Receipt           string    `json:"receipt,omitempty"`
	LastState         string    `json:"lastState,omitempty"`
	Stopping          bool      `json:"stopping,omitempty"`
	StopAfter         time.Time `json:"stopAfter,omitempty"`
	Quiesced          bool      `json:"quiesced,omitempty"`
	DeleteAttempted   bool      `json:"deleteAttempted,omitempty"`
	DeleteAfter       time.Time `json:"deleteAfter,omitempty"`
	InstanceAbsent    bool      `json:"instanceAbsent,omitempty"`
	DeadlineRemoving  bool      `json:"deadlineRemoving,omitempty"`
	DeadlineRemoved   bool      `json:"deadlineRemoved,omitempty"`
	Removed           bool      `json:"removed,omitempty"`
}

func NewController(o ControllerOptions) (*Controller, error) {
	if Validate(o.Config) != nil || o.Instances == nil || o.Deadlines == nil || o.Admission == nil || o.Exchange == nil || o.Exchange.Store == nil || o.Exchange.Authorize == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	return &Controller{config: o.Config, instances: o.Instances, deadlines: o.Deadlines, admission: o.Admission, exchange: o.Exchange, store: o.Exchange.Store, now: now}, nil
}

func controlError(err error) error {
	for _, known := range []error{hostedexecution.ErrConfiguration, hostedexecution.ErrOwnership, hostedexecution.ErrRejected, hostedexecution.ErrNotFound, hostedexecution.ErrUnavailable} {
		if errors.Is(err, known) {
			return known
		}
	}
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func bounded(ctx context.Context, call func(context.Context) error) error {
	step, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if step.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	err := call(step)
	if step.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	return controlError(err)
}

var _ Instances = (*Client)(nil)
