package azuresandbox

import (
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type Plan struct {
	config Config
	proof  Proof
	token  string
}

func NewPlan(c Config, job hostedexecution.Job, disk DiskImage, inspected hostedexecution.Image, now time.Time) (*Plan, error) {
	if !ticketToken.MatchString(job.Ticket.Token) || job.Ticket.Execution != job.Reference.ID || !job.Ticket.Deadline.After(now) || job.Ticket.Deadline.Sub(now) > 2*time.Hour || job.ExchangeOrigin != c.ExchangeOrigin {
		return nil, hostedexecution.ErrConfiguration
	}
	if inspected.Reference != job.Image || inspected.Runner != job.Runner || inspected.Project != job.Reference.Project || inspected.OS != "linux" || inspected.Architecture != "amd64" || inspected.Supervisor != "worker-exchange/v1" || inspected.Sandbox != hostedexecution.SandboxSupervisor || inspected.Runtime == "" || inspected.TaskStorage != hostedexecution.NonRootTaskStorage || !validSourceDigest(inspected.SourceDigest) || job.Operation == "discover" && inspected.Discovery != "worker-exchange/v1" {
		return nil, hostedexecution.ErrConfiguration
	}
	if disk.Base != job.Image || !identifier.MatchString(disk.ID) || !slices.Equal(disk.Entrypoint, []string{"/runner/teststudio-job"}) || len(disk.Command) != 0 {
		return nil, hostedexecution.ErrOwnership
	}
	if !readyImage(disk.State) {
		return nil, hostedexecution.ErrUnavailable
	}
	proof := Proof{Reference: job.Reference, Image: job.Image, SourceDigest: inspected.SourceDigest, Sandbox: inspected.Sandbox, DiskImage: disk.ID, Runner: job.Runner, Operation: job.Operation, Deadline: job.Ticket.Deadline.UTC(), TicketHash: hash(job.Ticket.Token), EnvironmentHash: hash(environment(job.Reference.ID, job.Ticket.Token))}
	p, err := newPlan(c, proof)
	if err != nil {
		return nil, err
	}
	p.token = job.Ticket.Token
	return p, nil
}

func readyImage(state string) bool { return state == "Ready" || state == "Succeeded" }

func newPlan(c Config, proof Proof) (*Plan, error) {
	if Validate(c) != nil || !validReference(proof.Reference, proof.Runner, proof.Operation) || !c.validImage(proof.Image) || !validSourceDigest(proof.SourceDigest) || proof.Sandbox != hostedexecution.SandboxSupervisor || !identifier.MatchString(proof.DiskImage) || !checksum.MatchString(proof.TicketHash) || !checksum.MatchString(proof.EnvironmentHash) || proof.Deadline.IsZero() || proof.Deadline.Year() < 2000 || proof.Deadline.Year() > 2200 {
		return nil, hostedexecution.ErrConfiguration
	}
	proof.Deadline = proof.Deadline.UTC()
	proof.Fingerprint = ""
	p := &Plan{config: c, proof: proof}
	p.proof.Fingerprint = hash([]any{"azure-sandbox/v3", c, APIVersion, TokenScope, proof, p.definition(), p.commandBody()})
	return p, nil
}

func validSourceDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && checksum.MatchString(strings.TrimPrefix(value, "sha256:"))
}

func Restore(c Config, proof Proof) (*Plan, error) {
	p, err := newPlan(c, proof)
	if err != nil || p.proof.Fingerprint != proof.Fingerprint {
		return nil, hostedexecution.ErrOwnership
	}
	return p, nil
}

func validReference(ref execution.Reference, runner, operation string) bool {
	if runner != "go" && runner != "playwright" || !domain.ValidID(ref.ID) || !domain.ValidID(ref.Installation) || !domain.ValidID(ref.Project) || !domain.ValidID(ref.Run) || !domain.ValidID(ref.Test) || !domain.ValidID(ref.Worker) || ref.Attempt < 1 || ref.Attempt > 3 {
		return false
	}
	if operation == "discover" {
		return strings.HasPrefix(ref.Installation, "preparation-") && ref.Test == "catalog" && ref.Attempt == 1 && ref == (&hostedexecution.DiscoveryInspector{Name: Location, Runner: runner}).Reference(ref)
	}
	return operation == "" && ref == (&hostedexecution.Executor{Name: Location, Runner: runner}).Reference(ref)
}

func environment(executionID, token string) map[string]string {
	return map[string]string{"TESTSTUDIO_JOB_EXECUTION": executionID, "TESTSTUDIO_JOB_TOKEN": token}
}

func (p *Plan) Proof() Proof                { return p.proof }
func (p Plan) String() string               { return "Azure sandbox plan: " + p.proof.Reference.ID }
func (p Plan) GoString() string             { return p.String() }
func (p Plan) MarshalJSON() ([]byte, error) { return []byte(`"Azure sandbox launch plan"`), nil }

func (p *Plan) definition() map[string]any {
	u, _ := url.Parse(p.config.ExchangeOrigin)
	command := []string{"/runner/teststudio-job", "--sandbox-idle-until=" + p.proof.Deadline.Format(time.RFC3339Nano)}
	return map[string]any{
		"sourcesRef": map[string]any{"diskImage": map[string]any{"id": p.proof.DiskImage}},
		"resources":  map[string]string{"cpu": "2000m", "memory": "4096Mi", "disk": "40Gi"},
		"lifecycle":  map[string]any{"autoSuspendPolicy": map[string]any{"enabled": true, "interval": 7200, "mode": "Disk"}},
		"labels":     map[string]string{"teststudio-owner": hash(p.proof.Reference), "teststudio-spec": p.proof.Fingerprint, "teststudio-scope": hash(p.config), "teststudio-execution": p.proof.Reference.ID, "teststudio-project": p.proof.Reference.Project, "teststudio-installation": p.proof.Reference.Installation, "teststudio-deadline": p.proof.Deadline.Format(time.RFC3339Nano)},
		"entrypoint": command, "cmd": []string{}, "skipEgressProxy": false,
		"customerVnetConnectionName": p.config.VNetConnection,
		"vmmType":                    p.config.VMMType,
		"egressPolicy":               map[string]any{"defaultAction": "Deny", "trafficInspection": "Full", "rules": []any{map[string]any{"name": "teststudio-exchange", "match": map[string]any{"host": u.Hostname()}, "action": map[string]string{"type": "Allow"}}}},
	}
}

func (p *Plan) launch(now time.Time) (map[string]any, error) {
	if !ticketToken.MatchString(p.token) || hash(p.token) != p.proof.TicketHash || hash(environment(p.proof.Reference.ID, p.token)) != p.proof.EnvironmentHash || !p.proof.Deadline.After(now) {
		return nil, hostedexecution.ErrRejected
	}
	body := p.definition()
	body["environment"] = environment(p.proof.Reference.ID, p.token)
	return body, nil
}

func (p *Plan) Verify(snapshot Snapshot) error {
	if !identifier.MatchString(snapshot.ID) {
		return hostedexecution.ErrOwnership
	}
	want := p.definition()
	want["environment"] = p.proof.EnvironmentHash
	if hash(normalize(want)) != snapshot.DefinitionHash {
		return hostedexecution.ErrOwnership
	}
	return nil
}
