package ecsworker

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestECSEnvironmentWaitsForInitializedHealthyGatewayAndDispatchesOneWorker(t *testing.T) {
	p, f, job := environmentLifecycle(t)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	value, _, _, err := p.worker.load(t.Context(), job.Reference)
	if err != nil || value.Proof.Environment == nil || value.Proof.Gateway != nil {
		t.Fatal("worker did not retain its distinct environment role", err)
	}
	restored, err := Restore(p.worker.config, value.Proof)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restored.Launch(p.worker.config.ecsPrefix() + "task-definition/" + restored.family + ":1"); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("recovery proof supplied worker launch authority", err)
	}
	if f.launches != 0 || f.gateway.launches != 1 {
		t.Fatal("worker started while the gateway was cold")
	}
	f.gateway.task.HealthStatus = types.HealthStatusHealthy
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || !status.AdmissionPending || f.launches != 0 {
		t.Fatal("health replaced private initialization", status, err)
	}
	readyEnvironmentGateway(t, p, f, job)
	f.gateway.task.HealthStatus = types.HealthStatusUnknown
	status, err = p.Observe(t.Context(), job.Reference)
	if err != nil || !status.AdmissionPending || f.launches != 0 {
		t.Fatal("initialization replaced readiness", status, err)
	}
	f.gateway.task.HealthStatus = types.HealthStatusHealthy
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() { _, _ = p.Observe(t.Context(), job.Reference) })
	}
	group.Wait()
	status, err = p.Observe(t.Context(), job.Reference)
	if err != nil || status.AdmissionPending || status.Image != job.Image || f.registrations != 1 || f.launches != 1 || f.gateway.launches != 1 || f.schedules != 1 || f.gateway.schedules != 1 {
		t.Fatal("ready execution duplicated native work or retained pending admission", status, err)
	}
	if err := f.gateway.exchange.Activate(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	packet, err := f.gateway.exchange.Input(t.Context(), job.Reference.ID, job.Ticket.Token, strings.Repeat("a", 32))
	if err != nil || len(packet.Input) == 0 {
		t.Fatal("ready worker could not claim its original input", err)
	}
	reopenEnvironment(t, p, f)
	f.allowedForObservation(t, p, job)
	refs, err := p.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile)
	if err != nil || len(refs) != 1 || refs[0] != job.Reference {
		t.Fatal("two roles produced duplicate inventory references", refs, err)
	}
	f.task.LastStatus, f.task.Containers[0].ExitCode = aws.String("STOPPED"), aws.Int32(0)
	status, err = p.Observe(t.Context(), job.Reference)
	if err != nil || status.Phase != "succeeded" || status.AdmissionPending || status.InfrastructureFailure {
		t.Fatal("worker completion did not preserve a healthy bundle observation", status, err)
	}
	if err := p.Remove(t.Context(), job.Reference); err == nil {
		t.Fatal("bundle removed before result and evidence checkpoints")
	}
	revoke(t, p.worker, job, true)
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	finishEnvironment(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	refs, err = p.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile)
	if err != nil || len(refs) != 0 || f.deletes != 1 || f.gateway.deletes != 1 || f.schedule != nil || f.gateway.schedule != nil {
		t.Fatal("bundle cleanup left owned resources", refs, err)
	}
	if err := p.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrRejected) || f.launches != 1 || f.gateway.launches != 1 {
		t.Fatal("removed bundle started another task", err)
	}
}

func (f *environmentCloud) allowedForObservation(t *testing.T, p *EnvironmentProvider, job hostedexecution.Job) {
	t.Helper()
	f.gateway.allowed = false
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || status.AdmissionPending || f.launches != 1 {
		t.Fatal("recovery changed a previously bound running execution", status, err)
	}
}

func finishEnvironment(t *testing.T, p *EnvironmentProvider, f *environmentCloud, job hostedexecution.Job) {
	t.Helper()
	for _, role := range []*cloudFixture{f.cloudFixture, f.gateway.cloudFixture} {
		if role.task != nil && aws.ToString(role.task.LastStatus) != "STOPPED" {
			role.task.LastStatus, role.task.Containers[0].ExitCode = aws.String("STOPPED"), aws.Int32(137)
		}
	}
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || !terminal(status) || status.AdmissionPending {
		t.Fatal("bundle did not retain confirmed shutdown", status, err)
	}
}

func TestECSEnvironmentCancelsBeforeWorkerCreationAndDoesNotResumeRevokedWork(t *testing.T) {
	p, f, job := environmentLifecycle(t)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	readyEnvironmentGateway(t, p, f, job)
	f.gateway.allowed = false
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || !status.AdmissionPending || f.launches != 0 {
		t.Fatal("withdrawn authority dispatched uploaded code", status, err)
	}
	reopenEnvironment(t, p, f)
	revoke(t, p.worker, job, true)
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	finishEnvironment(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); err != nil || f.launches != 0 || f.gateway.launches != 1 || f.gateway.deletes != 1 {
		t.Fatal("partial bundle recovery launched a worker or lost gateway cleanup", err)
	}
}

func TestECSEnvironmentLostWorkerLaunchDoesNotReplayAfterRestart(t *testing.T) {
	p, f, job := environmentLifecycle(t)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	readyEnvironmentGateway(t, p, f, job)
	f.mode = "lost-launch"
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || !status.AdmissionPending || f.launches != 1 {
		t.Fatal("uncertain worker launch did not remain pending", status, err)
	}
	reopenEnvironment(t, p, f)
	for range 2 {
		_, _ = p.Observe(t.Context(), job.Reference)
	}
	revoke(t, p.worker, job, true)
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	finishEnvironment(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); err != nil || f.launches != 1 || f.gateway.launches != 1 {
		t.Fatal("lost launch replayed during bundle recovery", err)
	}
}

func TestECSEnvironmentStopsGatewayWhenWorkerObservationFails(t *testing.T) {
	p, f, job := environmentLifecycle(t)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	readyEnvironmentGateway(t, p, f, job)
	if _, err := p.Observe(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	f.mode = "task-outage"
	revoke(t, p.worker, job, true)
	if err := p.Stop(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrUnavailable) || f.gateway.stops != 1 {
		t.Fatal("worker read failure prevented independent gateway shutdown", err)
	}
	f.mode = ""
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	finishEnvironment(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestECSEnvironmentGatewayFailureCannotConfirmWorkerPass(t *testing.T) {
	p, f, job := environmentLifecycle(t)
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	readyEnvironmentGateway(t, p, f, job)
	if _, err := p.Observe(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	f.task.LastStatus, f.task.Containers[0].ExitCode = aws.String("STOPPED"), aws.Int32(0)
	f.gateway.task.HealthStatus = types.HealthStatusUnhealthy
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || !status.InfrastructureFailure || status.Phase == "succeeded" {
		t.Fatal("gateway infrastructure loss retained a passing bundle", status, err)
	}
	revoke(t, p.worker, job, true)
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	f.gateway.task.LastStatus, f.gateway.task.Containers[0].ExitCode = aws.String("STOPPED"), aws.Int32(137)
	reopenEnvironment(t, p, f)
	status, err = p.Observe(t.Context(), job.Reference)
	if err != nil || !status.InfrastructureFailure || status.Phase != "failed" || status.ExitCode != nil {
		t.Fatal("restart lost the infrastructure failure", status, err)
	}
}

func TestECSEnvironmentRejectsLegacyAddressCapabilityBeforeNativeCreation(t *testing.T) {
	p, f, job := environmentLifecycle(t)
	f.image.GatewayAddress = ""
	if err := p.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrRejected) || f.launches != 0 || f.gateway.launches != 0 {
		t.Fatal("legacy runtime started an ECS bundle", err)
	}
	if _, err := p.worker.store.Get(t.Context(), Bucket, job.Reference.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("rejected runtime reserved an execution", err)
	}
	if _, err := f.gateway.exchange.GatewayInput(t.Context(), job.Reference.ID, strings.Repeat("a", 43), "unbound"); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("rejected runtime delivered initialization", err)
	}
}

func TestECSEnvironmentUnknownGatewayLaunchRetainsCleanupAndDoesNotDispatchWorker(t *testing.T) {
	p, f, job := environmentLifecycle(t)
	f.gateway.mode = "hidden-launch"
	if err := p.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("unknown gateway launch was admitted", err)
	}
	reopenEnvironment(t, p, f)
	_ = p.Ensure(t.Context(), job)
	revoke(t, p.worker, job, true)
	if err := p.Remove(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrUnavailable) || f.gateway.deletes != 0 || f.launches != 0 || f.gateway.launches != 1 {
		t.Fatal("unknown gateway launch replayed or disappeared during cleanup", err)
	}
	f.gateway.makeTask()
	if err := p.Stop(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	finishEnvironment(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); err != nil || f.launches != 0 || f.gateway.launches != 1 {
		t.Fatal("late gateway receipt could not clean the original workload", err)
	}
}
