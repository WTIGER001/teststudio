package ecsworker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type environmentCloud struct {
	*cloudFixture
	gateway *gatewayCloudFixture
	image   hostedexecution.Image
}

func environmentLifecycle(t *testing.T) (*EnvironmentProvider, *environmentCloud, hostedexecution.Job) {
	t.Helper()
	gateway, f, job, _ := gatewayLifecycle(t, "")
	f.nativeTaskID = strings.Repeat("b", 32)
	plan, err := newEnvironmentPlan(gateway.control.config, gateway.control.gateway.config, job, f.now)
	if err != nil {
		t.Fatal(err)
	}
	cloud := &environmentCloud{cloudFixture: &cloudFixture{t: t, plan: plan, now: f.now, database: f.database}, gateway: f}
	cloud.image = hostedexecution.Image{Reference: job.Image, OS: "linux", Architecture: plan.config.Architecture, Runner: job.Runner, Runtime: "go-test", Supervisor: "worker-exchange/v1", SupervisorRevision: strings.Repeat("c", 64), TaskStorage: hostedexecution.NonRootTaskStorage, GatewayTransport: browsergateway.RemoteProtocol, GatewayAddress: browsergateway.RemoteAddressProtocol}
	p, err := NewEnvironment(EnvironmentOptions{Worker: Options{Config: plan.config, Store: gateway.control.store, ECS: cloud, Scheduler: cloud, Inspector: cloud, Network: cloud, Now: gateway.control.now}, Gateway: gateway.control.gateway.config, Authority: f.exchange, Network: f})
	if err != nil {
		t.Fatal(err)
	}
	return p, cloud, job
}

func (f *environmentCloud) role(value string) *cloudFixture {
	if strings.Contains(value, "-gateway-") || strings.HasSuffix(value, "/"+strings.Repeat("b", 32)) {
		return f.gateway.cloudFixture
	}
	return f.cloudFixture
}

func (f *environmentCloud) InspectImage(context.Context, string) (hostedexecution.Image, error) {
	return f.image, nil
}

func (f *environmentCloud) InspectGateway(ctx context.Context, image string) (hostedexecution.GatewayImage, error) {
	return f.gateway.InspectGateway(ctx, image)
}

func (f *environmentCloud) RegisterTaskDefinition(ctx context.Context, input *ecs.RegisterTaskDefinitionInput, options ...func(*ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error) {
	return f.role(aws.ToString(input.Family)).RegisterTaskDefinition(ctx, input, options...)
}

func (f *environmentCloud) DescribeTaskDefinition(ctx context.Context, input *ecs.DescribeTaskDefinitionInput, options ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
	return f.role(aws.ToString(input.TaskDefinition)).DescribeTaskDefinition(ctx, input, options...)
}

func (f *environmentCloud) ListTaskDefinitions(ctx context.Context, input *ecs.ListTaskDefinitionsInput, options ...func(*ecs.Options)) (*ecs.ListTaskDefinitionsOutput, error) {
	return f.role(aws.ToString(input.FamilyPrefix)).ListTaskDefinitions(ctx, input, options...)
}

func (f *environmentCloud) RunTask(ctx context.Context, input *ecs.RunTaskInput, options ...func(*ecs.Options)) (*ecs.RunTaskOutput, error) {
	return f.role(aws.ToString(input.TaskDefinition)).RunTask(ctx, input, options...)
}

func (f *environmentCloud) DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput, options ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	role := f.role(input.Tasks[0])
	if role.mode == "task-outage" {
		return nil, errors.New("private-observation-canary")
	}
	return role.DescribeTasks(ctx, input, options...)
}

func (f *environmentCloud) ListTasks(ctx context.Context, input *ecs.ListTasksInput, options ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	return f.role(aws.ToString(input.Family)).ListTasks(ctx, input, options...)
}

func (f *environmentCloud) StopTask(ctx context.Context, input *ecs.StopTaskInput, options ...func(*ecs.Options)) (*ecs.StopTaskOutput, error) {
	return f.role(aws.ToString(input.Task)).StopTask(ctx, input, options...)
}

func (f *environmentCloud) DeregisterTaskDefinition(ctx context.Context, input *ecs.DeregisterTaskDefinitionInput, options ...func(*ecs.Options)) (*ecs.DeregisterTaskDefinitionOutput, error) {
	return f.role(aws.ToString(input.TaskDefinition)).DeregisterTaskDefinition(ctx, input, options...)
}

func (f *environmentCloud) DeleteTaskDefinitions(ctx context.Context, input *ecs.DeleteTaskDefinitionsInput, options ...func(*ecs.Options)) (*ecs.DeleteTaskDefinitionsOutput, error) {
	return f.role(input.TaskDefinitions[0]).DeleteTaskDefinitions(ctx, input, options...)
}

func (f *environmentCloud) CreateSchedule(ctx context.Context, input *scheduler.CreateScheduleInput, options ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
	return f.role(aws.ToString(input.Name)).CreateSchedule(ctx, input, options...)
}

func (f *environmentCloud) GetSchedule(ctx context.Context, input *scheduler.GetScheduleInput, options ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error) {
	return f.role(aws.ToString(input.Name)).GetSchedule(ctx, input, options...)
}

func (f *environmentCloud) DeleteSchedule(ctx context.Context, input *scheduler.DeleteScheduleInput, options ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	return f.role(aws.ToString(input.Name)).DeleteSchedule(ctx, input, options...)
}

func readyEnvironmentGateway(t *testing.T, p *EnvironmentProvider, f *environmentCloud, job hostedexecution.Job) {
	t.Helper()
	if _, err := p.Observe(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	initialization, err := f.gateway.exchange.PrepareGateway(t.Context(), job.Reference, p.gateway.control.gatewaySettings())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.gateway.exchange.GatewayInput(t.Context(), job.Reference.ID, initialization.Token, strings.Repeat("b", 32)); err != nil {
		t.Fatal(err)
	}
	f.gateway.task.HealthStatus = "HEALTHY"
}

func reopenEnvironment(t *testing.T, p *EnvironmentProvider, f *environmentCloud) {
	t.Helper()
	p.worker = reopenProvider(t, p.worker, f.cloudFixture)
	replacement := *p.gateway.control
	replacement.store = p.worker.store
	p.gateway = &GatewayProvider{control: &replacement}
	f.gateway.exchange.Store = p.worker.store
}
