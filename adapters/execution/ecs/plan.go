package ecsworker

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type Plan struct {
	config      Config
	job         hostedexecution.Job
	family      string
	owner       string
	scope       string
	spec        string
	fingerprint string
	ticketHash  string
	gateway     *GatewayConfig
}

func NewPlan(config Config, job hostedexecution.Job, now time.Time) (*Plan, error) {
	if !ticket.MatchString(job.Ticket.Token) || !job.Ticket.Deadline.After(now) || job.Ticket.Deadline.After(now.Add(2*time.Hour)) {
		return nil, hostedexecution.ErrConfiguration
	}
	return newPlan(config, job, fingerprint(job.Ticket.Token))
}

func newPlan(config Config, job hostedexecution.Job, ticketHash string) (*Plan, error) {
	return newTaskPlan(config, job, ticketHash, nil)
}

func newTaskPlan(config Config, job hostedexecution.Job, ticketHash string, gateway *GatewayConfig) (*Plan, error) {
	ref := job.Reference
	valid := validReference(ref, job.Runner, job.Operation)
	if gateway != nil {
		valid = validGatewayReference(ref, job.Runner, job.Operation) && validateGateway(config, *gateway) == nil && gateway.Image == job.Image
	}
	if Validate(config) != nil || !valid || !immutable.MatchString(job.Image) || len(job.Image) > 1024 || strings.Contains(job.Image, "..") || job.ExchangeOrigin != config.ExchangeOrigin || job.Ticket.Execution != ref.ID || job.Ticket.Deadline.IsZero() || !checksum.MatchString(ticketHash) {
		return nil, hostedexecution.ErrConfiguration
	}
	job.Ticket.Deadline = job.Ticket.Deadline.UTC()
	p := &Plan{config: canonical(config), job: job, owner: fingerprint(ref), scope: fingerprint(canonical(config)), ticketHash: ticketHash}
	p.family = config.FamilyPrefix + "-" + p.owner[:40]
	protocol := "ecs-worker/v1"
	if gateway != nil {
		selected := canonicalGateway(*gateway)
		p.gateway = &selected
		p.scope = fingerprint([]any{canonical(config), selected})
		p.family = config.FamilyPrefix + "-gateway-" + p.owner[:40]
		protocol = "ecs-gateway/v1"
	}
	p.spec = fingerprint(document(p.definition(false)))
	p.fingerprint = fingerprint([]any{protocol, p.scope, ref, job.Image, job.Runner, job.Operation, job.Ticket.Deadline, ticketHash, p.spec})
	return p, nil
}

func validReference(ref execution.Reference, runner, operation string) bool {
	if (runner != "go" && runner != "playwright") || !domain.ValidID(ref.ID) || !domain.ValidID(ref.Installation) || !domain.ValidID(ref.Project) || !domain.ValidID(ref.Run) || !domain.ValidID(ref.Test) || !domain.ValidID(ref.Worker) || ref.Attempt < 1 || ref.Attempt > 3 {
		return false
	}
	expected := (&hostedexecution.Executor{Name: "ecs", Runner: runner}).Reference(ref)
	if operation == "discover" {
		expected = (&hostedexecution.DiscoveryInspector{Name: "ecs", Runner: runner}).Reference(ref)
		return strings.HasPrefix(ref.Installation, "preparation-") && ref.Test == "catalog" && ref.Attempt == 1 && ref == expected
	}
	return operation == "" && ref == expected
}

func (p *Plan) String() string {
	if p.gateway != nil {
		return "ECS gateway plan: " + p.job.Reference.ID
	}
	return "ECS worker plan: " + p.job.Reference.ID
}
func (p *Plan) GoString() string { return p.String() }

func (p *Plan) Fingerprint() string { return p.fingerprint }
func (p *Plan) Family() string      { return p.family }
func (p *Plan) StartedBy() string   { return "ts-" + p.owner[:32] }

func (p *Plan) tags() []types.Tag {
	operation := "execute"
	worker := "isolated-v1"
	if p.gateway != nil {
		operation, worker = "gateway", "gateway-v1"
	}
	if p.job.Operation == "discover" {
		operation = "discover"
	}
	return []types.Tag{
		{Key: aws.String("teststudio:worker"), Value: aws.String(worker)},
		{Key: aws.String("teststudio:owner"), Value: aws.String(p.owner)},
		{Key: aws.String("teststudio:scope"), Value: aws.String(p.scope)},
		{Key: aws.String("teststudio:spec"), Value: aws.String(p.spec)},
		{Key: aws.String("teststudio:execution"), Value: aws.String(p.job.Reference.ID)},
		{Key: aws.String("teststudio:installation"), Value: aws.String(p.job.Reference.Installation)},
		{Key: aws.String("teststudio:project"), Value: aws.String(p.job.Reference.Project)},
		{Key: aws.String("teststudio:deadline"), Value: aws.String(p.job.Ticket.Deadline.UTC().Format(time.RFC3339Nano))},
		{Key: aws.String("teststudio:operation"), Value: aws.String(operation)},
	}
}

func (p *Plan) Definition() *ecs.RegisterTaskDefinitionInput { return p.definition(true) }

func (p *Plan) definition(tagged bool) *ecs.RegisterTaskDefinitionInput {
	if p.gateway != nil {
		return p.gatewayDefinition(tagged)
	}
	architecture := types.CPUArchitectureX8664
	if p.config.Architecture == "arm64" {
		architecture = types.CPUArchitectureArm64
	}
	args := []string{"--runner=" + p.job.Runner, "--exchange-origin=" + p.config.ExchangeOrigin}
	if p.job.Operation == "discover" {
		args = append(args, "--operation=discover")
	}
	value := &ecs.RegisterTaskDefinitionInput{
		Family: aws.String(p.family), Cpu: aws.String("2048"), Memory: aws.String("4096"),
		ExecutionRoleArn: aws.String(p.config.ExecutionRole), EnableFaultInjection: aws.Bool(false),
		NetworkMode: types.NetworkModeAwsvpc, RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		RuntimePlatform:  &types.RuntimePlatform{CpuArchitecture: architecture, OperatingSystemFamily: types.OSFamilyLinux},
		EphemeralStorage: &types.EphemeralStorage{SizeInGiB: 21},
		Volumes:          []types.Volume{{Name: aws.String("temporary")}, {Name: aws.String("output")}},
		ContainerDefinitions: []types.ContainerDefinition{{
			Name: aws.String("runner"), Image: aws.String(p.job.Image), Essential: aws.Bool(true), User: aws.String("1000:1000"),
			EntryPoint: []string{"/runner/teststudio-job"}, Command: args, WorkingDirectory: aws.String("/suite"),
			ReadonlyRootFilesystem: aws.Bool(true), Privileged: aws.Bool(false), Interactive: aws.Bool(false), PseudoTerminal: aws.Bool(false),
			MountPoints: []types.MountPoint{{ContainerPath: aws.String("/tmp"), SourceVolume: aws.String("temporary"), ReadOnly: aws.Bool(false)}, {ContainerPath: aws.String("/output"), SourceVolume: aws.String("output"), ReadOnly: aws.Bool(false)}},
			StopTimeout: aws.Int32(10), RestartPolicy: &types.ContainerRestartPolicy{Enabled: aws.Bool(false)},
			LinuxParameters: &types.LinuxParameters{
				InitProcessEnabled: aws.Bool(true), Capabilities: &types.KernelCapabilities{Drop: []string{"ALL"}},
			},
		}},
	}
	if tagged {
		value.Tags = p.tags()
	}
	return value
}
