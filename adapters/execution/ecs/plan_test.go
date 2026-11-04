package ecsworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	scheduletypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func fixture(t *testing.T, runner, operation string) (Config, hostedexecution.Job, *Plan) {
	t.Helper()
	c := Config{Region: "us-east-1", Account: "123456789012", Cluster: "arn:aws:ecs:us-east-1:123456789012:cluster/studio-workers", ExecutionRole: "arn:aws:iam::123456789012:role/studio-image-pull", DeadlineRole: "arn:aws:iam::123456789012:role/studio-deadlines", ScheduleGroup: "studio-deadlines", FamilyPrefix: "studio-worker", Architecture: "arm64", Platform: "1.4.0", Subnets: []string{"subnet-aaaaaaaa", "subnet-bbbbbbbb"}, SecurityGroups: []string{"sg-aaaaaaaa"}, ExchangeOrigin: "https://workers.studio.internal.example.test"}
	ref := execution.Reference{ID: "attempt-one", Installation: "studio", Project: "invoices", Run: "run-one", Test: "INVOICE-7", Attempt: 1, Worker: "controller-one"}
	if operation == "discover" {
		ref.Installation, ref.Test = "preparation-studio", "catalog"
		ref = (&hostedexecution.DiscoveryInspector{Name: "ecs", Runner: runner}).Reference(ref)
	} else {
		ref = (&hostedexecution.Executor{Name: "ecs", Runner: runner}).Reference(ref)
	}
	job := hostedexecution.Job{Operation: operation, Reference: ref, Image: "registry.example.test/studio/source@sha256:" + strings.Repeat("a", 64), Runner: runner, ExchangeOrigin: c.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: ref.ID, Token: "private_job_ticket_canary_" + strings.Repeat("x", 17), Deadline: time.Now().UTC().Add(30*time.Second + 123*time.Nanosecond)}}
	p, err := NewPlan(c, job, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return c, job, p
}

func definition(t *testing.T, p *Plan) *ecs.DescribeTaskDefinitionOutput {
	t.Helper()
	input := p.Definition()
	data, _ := json.Marshal(input)
	var result ecstypes.TaskDefinition
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	result.TaskDefinitionArn, result.Revision, result.Status = aws.String(p.config.ecsPrefix()+"task-definition/"+p.family+":1"), 1, ecstypes.TaskDefinitionStatusActive
	result.RegisteredAt, result.RegisteredBy = aws.Time(time.Now()), aws.String("platform-metadata")
	result.Compatibilities = []ecstypes.Compatibility{ecstypes.CompatibilityFargate}
	return &ecs.DescribeTaskDefinitionOutput{TaskDefinition: &result, Tags: input.Tags}
}

func taskARN(p *Plan) string {
	return p.config.ecsPrefix() + "task/" + p.config.clusterName() + "/" + strings.Repeat("a", 32)
}

func schedule(t *testing.T, p *Plan) *scheduler.GetScheduleOutput {
	t.Helper()
	input, err := p.Deadline(taskARN(p))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(input)
	var result scheduler.GetScheduleOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	result.Arn = aws.String("arn:aws:scheduler:" + p.config.Region + ":" + p.config.Account + ":schedule/" + p.config.ScheduleGroup + "/" + p.job.Reference.Container)
	result.CreationDate = aws.Time(time.Now())
	return &result
}

func TestFargatePlansKeepDiscoveryAndExecutionFixedAndCredentialsOutsideDefinitions(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"-"+operation, func(t *testing.T) {
				c, job, p := fixture(t, runner, operation)
				actual := definition(t, p)
				if err := p.VerifyDefinition(actual); err != nil {
					t.Fatal(err)
				}
				draft := p.Definition()
				container := draft.ContainerDefinitions[0]
				if len(draft.ContainerDefinitions) != 1 || aws.ToString(draft.TaskRoleArn) != "" || aws.ToString(draft.ExecutionRoleArn) != c.ExecutionRole || aws.ToString(draft.Cpu) != "2048" || aws.ToString(draft.Memory) != "4096" || len(draft.Volumes) != 0 || len(container.Environment) != 0 || len(container.Secrets) != 0 || container.LogConfiguration != nil || aws.ToString(container.User) != "1000:1000" || !aws.ToBool(container.ReadonlyRootFilesystem) || len(container.LinuxParameters.Tmpfs) != 2 {
					t.Fatal("worker definition changed its fixed isolation contract")
				}
				launch, err := p.Launch(*actual.TaskDefinition.TaskDefinitionArn)
				if err != nil || *launch.Count != 1 || launch.EnableExecuteCommand || launch.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp != ecstypes.AssignPublicIpDisabled || launch.Overrides.TaskRoleArn != nil || len(launch.Overrides.ContainerOverrides[0].Environment) != 2 {
					t.Fatal("worker launch changed its fixed isolation contract", err)
				}
				if (len(container.Command) == 3) != (operation == "discover") || container.EntryPoint[0] != "/runner/teststudio-job" {
					t.Fatal("source selected an arbitrary command")
				}
				again, err := NewPlan(c, job, time.Now())
				if err != nil || again.Fingerprint() != p.Fingerprint() || *launch.ClientToken != again.Fingerprint() {
					t.Fatal("launch retry identity changed", err)
				}
				data, _ := json.Marshal(draft)
				if strings.Contains(string(data), job.Ticket.Token) || strings.Contains(fmt.Sprintf("%#v %v", p, p), job.Ticket.Token) {
					t.Fatal("worker ticket escaped into task definition or plan diagnostics")
				}
				c.Subnets[0] = "subnet-cccccccc"
				draft.ContainerDefinitions[0].Command[0] = "untrusted"
				if err := p.VerifyDefinition(actual); err != nil {
					t.Fatal("caller mutated the retained plan", err)
				}
			})
		}
	}
}

func TestFargateDefinitionProofRejectsIdentityCommandsCredentialsAndResourceMutation(t *testing.T) {
	mutations := []func(*ecstypes.TaskDefinition){
		func(d *ecstypes.TaskDefinition) {
			d.TaskRoleArn = aws.String("arn:aws:iam::123456789012:role/api-credentials")
		},
		func(d *ecstypes.TaskDefinition) { d.ContainerDefinitions[0].User = aws.String("0") },
		func(d *ecstypes.TaskDefinition) { d.ContainerDefinitions[0].ReadonlyRootFilesystem = aws.Bool(false) },
		func(d *ecstypes.TaskDefinition) { d.ContainerDefinitions[0].EntryPoint = []string{"sh", "-c"} },
		func(d *ecstypes.TaskDefinition) {
			d.ContainerDefinitions[0].Secrets = []ecstypes.Secret{{Name: aws.String("injected"), ValueFrom: aws.String("private-secret")}}
		},
		func(d *ecstypes.TaskDefinition) {
			d.ContainerDefinitions = append(d.ContainerDefinitions, d.ContainerDefinitions[0])
		},
		func(d *ecstypes.TaskDefinition) {
			d.ContainerDefinitions[0].LinuxParameters.Capabilities.Add = []string{"SYS_ADMIN"}
		},
		func(d *ecstypes.TaskDefinition) {
			d.ContainerDefinitions[0].Environment = []ecstypes.KeyValuePair{{Name: aws.String("AWS_ACCESS_KEY_ID"), Value: aws.String("credential-canary")}}
		},
		func(d *ecstypes.TaskDefinition) { d.ContainerDefinitions[0].RestartPolicy.Enabled = aws.Bool(true) },
		func(d *ecstypes.TaskDefinition) { d.Memory = aws.String("8192") },
		func(d *ecstypes.TaskDefinition) { d.Revision = 2 },
		func(d *ecstypes.TaskDefinition) { d.PidMode = ecstypes.PidModeHost },
	}
	for index, mutate := range mutations {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			_, _, p := fixture(t, "go", "")
			actual := definition(t, p)
			mutate(actual.TaskDefinition)
			if !errors.Is(p.VerifyDefinition(actual), hostedexecution.ErrOwnership) {
				t.Fatal("mutated worker definition passed ownership verification")
			}
		})
	}
	_, _, p := fixture(t, "go", "")
	actual := definition(t, p)
	actual.Tags = append(actual.Tags, actual.Tags[0])
	if p.VerifyDefinition(actual) == nil {
		t.Fatal("duplicate ownership tags were accepted")
	}
}

func TestFargateDeadlineProofRequiresScopedTaskAndUnchangedIndependentStop(t *testing.T) {
	_, _, p := fixture(t, "go", "discover")
	actual := schedule(t, p)
	if err := p.VerifyDeadline(taskARN(p), actual); err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse("2006-01-02T15:04:05", strings.TrimSuffix(strings.TrimPrefix(*actual.ScheduleExpression, "at("), ")"))
	if err != nil || at.Before(p.job.Ticket.Deadline) || at.Sub(p.job.Ticket.Deadline) >= time.Second || actual.Target.EcsParameters != nil || actual.ActionAfterCompletion != scheduletypes.ActionAfterCompletionNone {
		t.Fatal("deadline did not retain a bounded stop-task contract", err)
	}
	mutations := []func(*scheduler.GetScheduleOutput){
		func(s *scheduler.GetScheduleOutput) { s.State = scheduletypes.ScheduleStateDisabled },
		func(s *scheduler.GetScheduleOutput) { s.Target.RoleArn = aws.String(p.config.ExecutionRole) },
		func(s *scheduler.GetScheduleOutput) {
			s.Target.Arn = aws.String("arn:aws:scheduler:::aws-sdk:ecs:runTask")
		},
		func(s *scheduler.GetScheduleOutput) {
			s.Target.Input = aws.String(`{"cluster":"foreign","task":"another"}`)
		},
		func(s *scheduler.GetScheduleOutput) { s.ScheduleExpression = aws.String("rate(1 hour)") },
		func(s *scheduler.GetScheduleOutput) {
			s.FlexibleTimeWindow.Mode = scheduletypes.FlexibleTimeWindowModeFlexible
		},
		func(s *scheduler.GetScheduleOutput) {
			s.ActionAfterCompletion = scheduletypes.ActionAfterCompletionDelete
		},
	}
	for _, mutate := range mutations {
		value := schedule(t, p)
		mutate(value)
		if p.VerifyDeadline(taskARN(p), value) == nil {
			t.Fatal("mutated deadline passed ownership verification")
		}
	}
	if _, err := p.Deadline(strings.Replace(taskARN(p), "123456789012", "999999999999", 1)); err == nil {
		t.Fatal("a foreign task received a stop schedule")
	}
}

func TestFargateRecoveryProofOmitsTicketsAndCannotLaunchWork(t *testing.T) {
	c, job, p := fixture(t, "go", "")
	data, _ := json.Marshal(p.Proof())
	if strings.Contains(string(data), job.Ticket.Token) {
		t.Fatal("recovery proof retained a raw ticket")
	}
	var proof Proof
	if err := json.Unmarshal(data, &proof); err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(c, proof)
	if err != nil || !reflect.DeepEqual(document(restored.Definition()), document(p.Definition())) || restored.VerifyDefinition(definition(t, p)) != nil || restored.VerifyDeadline(taskARN(p), schedule(t, p)) != nil {
		t.Fatal("recovery could not verify the original immutable resources", err)
	}
	if _, err := restored.Launch(*definition(t, p).TaskDefinition.TaskDefinitionArn); err == nil {
		t.Fatal("recovery proof could redispatch a task")
	}
	proof.Reference.Worker = "another-controller"
	if _, err := Restore(c, proof); err == nil {
		t.Fatal("another claimant adopted the saved ownership proof")
	}
	c.SecurityGroups = []string{"sg-bbbbbbbb"}
	if _, err := Restore(c, p.Proof()); err == nil {
		t.Fatal("another execution pool adopted the saved proof")
	}
}
