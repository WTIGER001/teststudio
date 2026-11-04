package ecsworker

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type cloudFixture struct {
	mu                                                        sync.Mutex
	t                                                         *testing.T
	plan                                                      *Plan
	mode                                                      string
	storage                                                   string
	database                                                  string
	definition                                                *ecs.DescribeTaskDefinitionOutput
	task                                                      *ecstypes.Task
	schedule                                                  *scheduler.GetScheduleOutput
	launch                                                    *ecs.RunTaskInput
	registrations, launches, schedules, stops, deletes, reads int
	networkPending, networkRejected, checkpointed             bool
	now                                                       time.Time
}

func copyValue[T any](value T) T {
	data, _ := json.Marshal(value)
	var result T
	json.Unmarshal(data, &result)
	return result
}

func lifecycle(t *testing.T, runner, operation, mode string) (*Provider, *cloudFixture, hostedexecution.Job) {
	t.Helper()
	config, job, plan := fixture(t, runner, operation)
	database := filepath.Join(t.TempDir(), "ecs.db")
	db, err := storage.Open(t.Context(), database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	f := &cloudFixture{t: t, plan: plan, mode: mode, storage: hostedexecution.NonRootTaskStorage, now: time.Now().UTC(), database: database}
	provider, err := New(Options{Config: config, Store: db, ECS: f, Scheduler: f, Inspector: f, Network: f, Now: func() time.Time { f.mu.Lock(); defer f.mu.Unlock(); return f.now }, DiscoveryCleanup: func(context.Context, execution.Reference) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if !f.checkpointed {
			return hostedexecution.ErrUnavailable
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	transport, _ := storage.Put(workerexchange.Bucket, job.Reference.ID, job.Reference.Project, 0, workerexchange.Record{Reference: job.Reference, Operation: operation, Deadline: job.Ticket.Deadline, DispatchGated: true})
	if err := db.CommitAll(t.Context(), transport); err != nil {
		t.Fatal(err)
	}
	_, changes, err := (workerstate.Ledger{Store: db}).Begin(job.Reference)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.CommitAll(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	return provider, f, job
}

func (f *cloudFixture) InspectImage(_ context.Context, image string) (hostedexecution.Image, error) {
	return hostedexecution.Image{TaskStorage: f.storage, Reference: image, OS: "linux", Architecture: f.plan.config.Architecture}, nil
}
func (f *cloudFixture) Admit(context.Context, Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.networkRejected {
		return hostedexecution.ErrOwnership
	}
	return nil
}
func (f *cloudFixture) VerifyTask(context.Context, Config, ecstypes.Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.networkRejected {
		return hostedexecution.ErrOwnership
	}
	if f.networkPending {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
func (f *cloudFixture) RegisterTaskDefinition(_ context.Context, input *ecs.RegisterTaskDefinitionInput, _ ...func(*ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registrations++
	if fingerprint(input) != fingerprint(f.plan.Definition()) {
		f.t.Error("registered another definition")
	}
	f.definition = definition(f.t, f.plan)
	if f.mode == "lost-register" {
		return nil, errors.New("private-registration-canary")
	}
	return &ecs.RegisterTaskDefinitionOutput{TaskDefinition: copyValue(f.definition.TaskDefinition), Tags: copyValue(f.definition.Tags)}, nil
}
func (f *cloudFixture) DescribeTaskDefinition(context.Context, *ecs.DescribeTaskDefinitionInput, ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.mode == "read-outage" {
		return nil, errors.New("private-provider-error-canary")
	}
	if f.definition == nil {
		return nil, hostedexecution.ErrNotFound
	}
	return copyValue(f.definition), nil
}

func reopenProvider(t *testing.T, p *Provider, f *cloudFixture) *Provider {
	t.Helper()
	if err := p.store.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(t.Context(), f.database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	replacement := *p
	replacement.store = db
	return &replacement
}
func (f *cloudFixture) ListTaskDefinitions(_ context.Context, input *ecs.ListTaskDefinitionsInput, _ ...func(*ecs.Options)) (*ecs.ListTaskDefinitionsOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := &ecs.ListTaskDefinitionsOutput{}
	if f.definition != nil && f.definition.TaskDefinition.Status == input.Status {
		result.TaskDefinitionArns = []string{*f.definition.TaskDefinition.TaskDefinitionArn}
	}
	return result, nil
}
func (f *cloudFixture) makeTask() {
	f.task = &ecstypes.Task{TaskArn: aws.String(taskARN(f.plan)), ClusterArn: aws.String(f.plan.config.Cluster), TaskDefinitionArn: f.definition.TaskDefinition.TaskDefinitionArn, StartedBy: f.launch.StartedBy, Group: f.launch.Group, Cpu: aws.String("2048"), Memory: aws.String("4096"), PlatformVersion: aws.String("1.4.0"), PlatformFamily: aws.String("Linux"), LaunchType: ecstypes.LaunchTypeFargate, LastStatus: aws.String("RUNNING"), Overrides: copyValue(f.launch.Overrides), Tags: copyValue(f.launch.Tags), Containers: []ecstypes.Container{{Name: aws.String("runner"), Image: aws.String(f.plan.job.Image)}}}
}
func (f *cloudFixture) RunTask(_ context.Context, input *ecs.RunTaskInput, _ ...func(*ecs.Options)) (*ecs.RunTaskOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.launches++
	f.launch = copyValue(input)
	if f.mode == "launch-rejected" {
		return &ecs.RunTaskOutput{Failures: []ecstypes.Failure{{Reason: aws.String("rejected")}}}, nil
	}
	if f.mode == "hidden-launch" {
		return nil, errors.New("private-create-canary")
	}
	f.makeTask()
	if f.mode == "lost-launch" {
		return nil, errors.New("private-create-canary")
	}
	return &ecs.RunTaskOutput{Tasks: []ecstypes.Task{copyValue(*f.task)}}, nil
}
func (f *cloudFixture) DescribeTasks(_ context.Context, input *ecs.DescribeTasksInput, _ ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.task == nil {
		return &ecs.DescribeTasksOutput{Failures: []ecstypes.Failure{{Arn: aws.String(input.Tasks[0]), Reason: aws.String("MISSING")}}}, nil
	}
	return &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{copyValue(*f.task)}}, nil
}
func (f *cloudFixture) ListTasks(_ context.Context, input *ecs.ListTasksInput, _ ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := &ecs.ListTasksOutput{}
	if f.task != nil && (input.DesiredStatus == ecstypes.DesiredStatusStopped) == (*f.task.LastStatus == "STOPPED") {
		result.TaskArns = []string{*f.task.TaskArn}
	}
	return result, nil
}
func (f *cloudFixture) StopTask(_ context.Context, input *ecs.StopTaskInput, _ ...func(*ecs.Options)) (*ecs.StopTaskOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	if f.task == nil || *input.Task != *f.task.TaskArn {
		return nil, hostedexecution.ErrNotFound
	}
	f.task.LastStatus, f.task.Tags = aws.String("STOPPING"), nil
	return &ecs.StopTaskOutput{Task: copyValue(f.task)}, nil
}
func (f *cloudFixture) DeregisterTaskDefinition(context.Context, *ecs.DeregisterTaskDefinitionInput, ...func(*ecs.Options)) (*ecs.DeregisterTaskDefinitionOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.definition.TaskDefinition.Status = ecstypes.TaskDefinitionStatusInactive
	return &ecs.DeregisterTaskDefinitionOutput{TaskDefinition: copyValue(f.definition.TaskDefinition)}, nil
}
func (f *cloudFixture) DeleteTaskDefinitions(context.Context, *ecs.DeleteTaskDefinitionsInput, ...func(*ecs.Options)) (*ecs.DeleteTaskDefinitionsOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletes++
	if f.mode == "deletion-pending" {
		f.definition.TaskDefinition.Status = ecstypes.TaskDefinitionStatusDeleteInProgress
	} else {
		f.definition = nil
	}
	return &ecs.DeleteTaskDefinitionsOutput{}, nil
}
func (f *cloudFixture) CreateSchedule(_ context.Context, input *scheduler.CreateScheduleInput, _ ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.schedules++
	data, _ := json.Marshal(input)
	json.Unmarshal(data, &f.schedule)
	f.schedule.Arn = aws.String("arn:aws:scheduler:" + f.plan.config.Region + ":" + f.plan.config.Account + ":schedule/" + f.plan.config.ScheduleGroup + "/" + f.plan.job.Reference.Container)
	if f.mode == "lost-deadline" {
		return nil, errors.New("private-deadline-canary")
	}
	return &scheduler.CreateScheduleOutput{ScheduleArn: f.schedule.Arn}, nil
}
func (f *cloudFixture) GetSchedule(context.Context, *scheduler.GetScheduleInput, ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.schedule == nil {
		return nil, hostedexecution.ErrNotFound
	}
	return copyValue(f.schedule), nil
}
func (f *cloudFixture) DeleteSchedule(context.Context, *scheduler.DeleteScheduleInput, ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.schedule = nil
	return &scheduler.DeleteScheduleOutput{}, nil
}

func revoke(t *testing.T, p *Provider, job hostedexecution.Job, checkpoint bool) {
	t.Helper()
	row, err := p.store.Get(t.Context(), workerexchange.Bucket, job.Reference.ID)
	if err != nil {
		t.Fatal(err)
	}
	transport, _ := storage.Decode[workerexchange.Record](row)
	transport.Revoked = true
	change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, transport)
	if err := p.store.CommitAll(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if checkpoint {
		entry, row, err := (workerstate.Ledger{Store: p.store}).Get(t.Context(), job.Reference.Run, job.Reference.ID)
		if err != nil {
			t.Fatal(err)
		}
		entry.ResultSaved, entry.EvidenceComplete = true, true
		change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, entry)
		if err := p.store.CommitAll(t.Context(), change); err != nil {
			t.Fatal(err)
		}
	}
}

func stopObserved(t *testing.T, p *Provider, f *cloudFixture, job hostedexecution.Job) {
	t.Helper()
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.task.LastStatus = aws.String("STOPPED")
	f.task.Containers[0].ExitCode = aws.Int32(137)
	f.mu.Unlock()
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || status.Phase != "stopped" {
		t.Fatal("owned stop was not retained", status, err)
	}
}
