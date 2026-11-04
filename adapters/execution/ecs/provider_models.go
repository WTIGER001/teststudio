package ecsworker

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

const Bucket = "ecs-dispatches"

type ECS interface {
	RegisterTaskDefinition(context.Context, *ecs.RegisterTaskDefinitionInput, ...func(*ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error)
	DescribeTaskDefinition(context.Context, *ecs.DescribeTaskDefinitionInput, ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error)
	ListTaskDefinitions(context.Context, *ecs.ListTaskDefinitionsInput, ...func(*ecs.Options)) (*ecs.ListTaskDefinitionsOutput, error)
	RunTask(context.Context, *ecs.RunTaskInput, ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)
	DescribeTasks(context.Context, *ecs.DescribeTasksInput, ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
	ListTasks(context.Context, *ecs.ListTasksInput, ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	StopTask(context.Context, *ecs.StopTaskInput, ...func(*ecs.Options)) (*ecs.StopTaskOutput, error)
	DeregisterTaskDefinition(context.Context, *ecs.DeregisterTaskDefinitionInput, ...func(*ecs.Options)) (*ecs.DeregisterTaskDefinitionOutput, error)
	DeleteTaskDefinitions(context.Context, *ecs.DeleteTaskDefinitionsInput, ...func(*ecs.Options)) (*ecs.DeleteTaskDefinitionsOutput, error)
}

type Scheduler interface {
	CreateSchedule(context.Context, *scheduler.CreateScheduleInput, ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
	GetSchedule(context.Context, *scheduler.GetScheduleInput, ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error)
	DeleteSchedule(context.Context, *scheduler.DeleteScheduleInput, ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
}

type Inspector interface {
	InspectImage(context.Context, string) (hostedexecution.Image, error)
}

type Network interface {
	Admit(context.Context, Config) error
	VerifyTask(context.Context, Config, ecstypes.Task) error
}

type Options struct {
	Config           Config
	Store            *storage.Store
	ECS              ECS
	Scheduler        Scheduler
	Inspector        Inspector
	Network          Network
	DiscoveryCleanup func(context.Context, execution.Reference) error
	Now              func() time.Time
}

type Provider struct {
	config           Config
	store            *storage.Store
	ecs              ECS
	scheduler        Scheduler
	inspector        Inspector
	network          Network
	discoveryCleanup func(context.Context, execution.Reference) error
	now              func() time.Time
}

type dispatch struct {
	Proof              Proof                   `json:"proof"`
	CreatedAt          time.Time               `json:"createdAt"`
	RegisterAttempted  bool                    `json:"registerAttempted"`
	RegisterRejected   bool                    `json:"registerRejected,omitempty"`
	DefinitionARN      string                  `json:"definitionArn,omitempty"`
	LaunchAttempted    bool                    `json:"launchAttempted,omitempty"`
	LaunchRejected     bool                    `json:"launchRejected,omitempty"`
	TaskARN            string                  `json:"taskArn,omitempty"`
	DeadlineAttempted  bool                    `json:"deadlineAttempted,omitempty"`
	DeadlineVerified   bool                    `json:"deadlineVerified,omitempty"`
	DeadlineDeleted    bool                    `json:"deadlineDeleted,omitempty"`
	DefinitionDeleting bool                    `json:"definitionDeleting,omitempty"`
	DeleteCheckAfter   time.Time               `json:"deleteCheckAfter,omitempty"`
	Stopping           bool                    `json:"stopping,omitempty"`
	Removed            bool                    `json:"removed,omitempty"`
	Terminal           *hostedexecution.Status `json:"terminal,omitempty"`
}

func New(options Options) (*Provider, error) {
	if Validate(options.Config) != nil || options.Store == nil || options.ECS == nil || options.Scheduler == nil || options.Inspector == nil || options.Network == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Provider{config: canonical(options.Config), store: options.Store, ecs: options.ECS, scheduler: options.Scheduler, inspector: options.Inspector, network: options.Network, discoveryCleanup: options.DiscoveryCleanup, now: now}, nil
}

func (p *Provider) InspectImage(ctx context.Context, image string) (hostedexecution.Image, error) {
	result, err := p.inspector.InspectImage(ctx, image)
	if err != nil {
		return result, hostedexecution.ErrUnavailable
	}
	if result.Reference != image || result.OS != "linux" || result.Architecture != p.config.Architecture {
		return result, hostedexecution.ErrConfiguration
	}
	result.ExecutionProfile, err = ExecutionProfile(p.config)
	return result, err
}

func singleECS(options *ecs.Options)             { options.RetryMaxAttempts = 1 }
func singleScheduler(options *scheduler.Options) { options.RetryMaxAttempts = 1 }

var _ hostedexecution.Provider = (*Provider)(nil)
