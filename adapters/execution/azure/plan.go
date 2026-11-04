package azureworker

import (
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type Plan struct {
	config      Config
	job         hostedexecution.Job
	name        string
	owner       string
	ticketHash  string
	timeout     int32
	fingerprint string
}

func NewPlan(config Config, job hostedexecution.Job, now time.Time) (*Plan, error) {
	remaining := job.Ticket.Deadline.Sub(now)
	if !token.MatchString(job.Ticket.Token) || remaining <= 0 || remaining > 2*time.Hour {
		return nil, hostedexecution.ErrConfiguration
	}
	seconds := int32((remaining + time.Second - 1) / time.Second)
	return newPlan(config, job, fingerprint(job.Ticket.Token), seconds)
}

func newPlan(c Config, job hostedexecution.Job, ticketHash string, timeout int32) (*Plan, error) {
	if Validate(c) != nil || !validReference(job.Reference, job.Runner, job.Operation) || job.ExchangeOrigin != c.ExchangeOrigin || job.Ticket.Execution != job.Reference.ID || job.Ticket.Deadline.IsZero() || !checksum.MatchString(ticketHash) || timeout < 1 || timeout > 7200 {
		return nil, hostedexecution.ErrConfiguration
	}
	image, digest, found := strings.Cut(job.Image, "@sha256:")
	base := c.Registry + "/" + c.RepositoryPrefix + "/"
	if !found || !strings.HasPrefix(image, base) || !repository.MatchString(strings.TrimPrefix(image, c.Registry+"/")) || !checksum.MatchString(digest) || len(job.Image) > 1024 {
		return nil, hostedexecution.ErrConfiguration
	}
	job.Ticket.Deadline = job.Ticket.Deadline.UTC()
	p := &Plan{config: c, job: job, owner: fingerprint(job.Reference), ticketHash: ticketHash, timeout: timeout}
	p.name = c.NamePrefix + "-" + p.owner[:24]
	p.fingerprint = fingerprint([]any{"azure-worker/v1", c, job.Reference, job.Image, job.Runner, job.Operation, job.Ticket.Deadline, ticketHash, timeout, p.template()})
	return p, nil
}

func validReference(ref execution.Reference, runner, operation string) bool {
	if runner != "go" && runner != "playwright" || !domain.ValidID(ref.ID) || !domain.ValidID(ref.Installation) || !domain.ValidID(ref.Project) || !domain.ValidID(ref.Run) || !domain.ValidID(ref.Test) || !domain.ValidID(ref.Worker) || ref.Attempt < 1 || ref.Attempt > 3 {
		return false
	}
	if operation == "discover" {
		expected := (&hostedexecution.DiscoveryInspector{Name: Location, Runner: runner}).Reference(ref)
		return strings.HasPrefix(ref.Installation, "preparation-") && ref.Test == "catalog" && ref.Attempt == 1 && ref == expected
	}
	return operation == "" && ref == (&hostedexecution.Executor{Name: Location, Runner: runner}).Reference(ref)
}

func (p *Plan) String() string      { return "Azure worker plan: " + p.job.Reference.ID }
func (p *Plan) GoString() string    { return p.String() }
func (p *Plan) Name() string        { return p.name }
func (p *Plan) ResourceID() string  { return p.config.resourceID("Microsoft.App", "jobs", p.name) }
func (p *Plan) Fingerprint() string { return p.fingerprint }

func (p *Plan) tags() map[string]*string {
	operation := "execute"
	if p.job.Operation == "discover" {
		operation = "discover"
	}
	return map[string]*string{
		"teststudio-worker": to.Ptr("isolated-v1"), "teststudio-owner": to.Ptr(p.owner),
		"teststudio-scope": to.Ptr(fingerprint(p.config)), "teststudio-spec": to.Ptr(p.fingerprint),
		"teststudio-execution": to.Ptr(p.job.Reference.ID), "teststudio-installation": to.Ptr(p.job.Reference.Installation),
		"teststudio-project": to.Ptr(p.job.Reference.Project), "teststudio-operation": to.Ptr(operation),
		"teststudio-deadline": to.Ptr(p.job.Ticket.Deadline.Format(time.RFC3339Nano)),
	}
}

func (p *Plan) template() *a.JobTemplate {
	args := []*string{to.Ptr("--runner=" + p.job.Runner), to.Ptr("--exchange-origin=" + p.config.ExchangeOrigin)}
	if p.job.Operation == "discover" {
		args = append(args, to.Ptr("--operation=discover"))
	}
	return &a.JobTemplate{Containers: []*a.Container{{Name: to.Ptr("runner"), Image: to.Ptr(p.job.Image), Command: []*string{to.Ptr("/runner/teststudio-job")}, Args: args, Resources: &a.ContainerResources{CPU: to.Ptr(2.0), Memory: to.Ptr("4Gi")}}}}
}

func (p *Plan) Definition() a.Job {
	return a.Job{
		Location: to.Ptr(p.config.Region), Tags: p.tags(),
		Identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{p.config.PullIdentity: {}}},
		Properties: &a.JobProperties{
			EnvironmentID: to.Ptr(p.config.WorkerEnvironment), WorkloadProfileName: to.Ptr("Consumption"), Template: p.template(),
			Configuration: &a.JobConfiguration{
				TriggerType: to.Ptr(a.TriggerTypeManual), ReplicaRetryLimit: to.Ptr[int32](0), ReplicaTimeout: to.Ptr(p.timeout),
				ManualTriggerConfig: &a.JobConfigurationManualTriggerConfig{Parallelism: to.Ptr[int32](1), ReplicaCompletionCount: to.Ptr[int32](1)},
				Registries:          []*a.RegistryCredentials{{Server: to.Ptr(p.config.Registry), Identity: to.Ptr(p.config.PullIdentity)}},
				IdentitySettings:    []*a.IdentitySettings{{Identity: to.Ptr(p.config.PullIdentity), Lifecycle: to.Ptr(a.IdentitySettingsLifeCycleNone)}},
			},
		},
	}
}

func (p *Plan) Start(now time.Time) (*a.JobsClientBeginStartOptions, error) {
	if !token.MatchString(p.job.Ticket.Token) || !p.job.Ticket.Deadline.After(now) {
		return nil, hostedexecution.ErrRejected
	}
	container := p.template().Containers[0]
	return &a.JobsClientBeginStartOptions{Template: &a.JobExecutionTemplate{Containers: []*a.JobExecutionContainer{{
		Name: container.Name, Image: container.Image, Command: container.Command, Args: container.Args, Resources: container.Resources,
		Env: []*a.EnvironmentVar{{Name: to.Ptr("TESTSTUDIO_JOB_EXECUTION"), Value: to.Ptr(p.job.Reference.ID)}, {Name: to.Ptr("TESTSTUDIO_JOB_TOKEN"), Value: to.Ptr(p.job.Ticket.Token)}},
	}}}}, nil
}
