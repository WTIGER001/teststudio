package azuresandbox

import (
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestSandboxLaunchRequiresTheVersionedCommandCapability(t *testing.T) {
	p := fixturePlan(t, "go", "")
	job := hostedexecution.Job{Reference: p.proof.Reference, Image: p.proof.Image, Runner: p.proof.Runner, ExchangeOrigin: p.config.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: p.proof.Reference.ID, Token: p.token, Deadline: p.proof.Deadline}}
	disk := DiskImage{ID: p.proof.DiskImage, Base: p.proof.Image, State: "Ready", Entrypoint: []string{"/runner/teststudio-job"}}
	inspected := hostedexecution.Image{Reference: p.proof.Image, Runner: "go", Runtime: "fixture-runtime", Project: p.proof.Reference.Project, OS: "linux", Architecture: "amd64", Supervisor: "worker-exchange/v1", TaskStorage: hostedexecution.NonRootTaskStorage, SourceDigest: p.proof.SourceDigest}
	if _, err := NewPlan(p.config, job, disk, inspected, time.Now()); err != hostedexecution.ErrConfiguration {
		t.Fatal("older supervisor image accepted the new command contract", err)
	}
	inspected.Sandbox = hostedexecution.SandboxSupervisor
	if _, err := NewPlan(p.config, job, disk, inspected, time.Now()); err != nil {
		t.Fatal(err)
	}
}
