package ecsworker

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/ecsidentity"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type gatewayCloudFixture struct {
	*cloudFixture
	exchange *workerexchange.Service
	image    hostedexecution.GatewayImage
	address  string
	allowed  bool
}

func gatewayLifecycle(t *testing.T, mode string) (*GatewayProvider, *gatewayCloudFixture, hostedexecution.Job, workerexchange.GatewayTicket) {
	t.Helper()
	config, gateway, job, _, _ := gatewayPlanFixture(t)
	database := filepath.Join(t.TempDir(), "ecs-gateway.db")
	db, err := storage.Open(t.Context(), database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	f := &gatewayCloudFixture{cloudFixture: &cloudFixture{t: t, mode: mode, database: database, now: time.Now().UTC()}, address: "10.20.1.7", allowed: true}
	f.exchange = &workerexchange.Service{Store: db, Authorize: func(_ context.Context, ref execution.Reference) (workerstate.Access, error) {
		if !f.allowed || ref != job.Reference {
			return workerstate.Access{}, workerexchange.ErrDenied
		}
		return workerstate.Access{Deadline: job.Ticket.Deadline}, nil
	}}
	input := execution.CodedInput{RunID: job.Reference.Run, Test: catalog.Test{ID: job.Reference.Test, Runner: job.Runner, Selector: "TestApplication"}, Artifact: catalog.Artifact{Image: job.Image, Browser: &browserpolicy.Spec{Endpoint: "website"}, Discovered: map[string]string{job.Reference.Test: "TestApplication"}}, Browser: &browserpolicy.Target{Origin: "https://application.example.test"}, AwaitDispatch: true, TimeoutMs: 1000}
	job.Ticket, err = f.exchange.Prepare(t.Context(), job.Reference, input, job.Ticket.Deadline)
	if err != nil {
		t.Fatal(err)
	}
	initialization, err := f.exchange.PrepareGateway(t.Context(), job.Reference, workerexchange.GatewaySettings{Origin: gateway.Origin, RequireAddress: true})
	if err != nil {
		t.Fatal(err)
	}
	f.plan, err = NewGatewayPlan(config, gateway, job, initialization, f.now)
	if err != nil {
		t.Fatal(err)
	}
	f.image = hostedexecution.GatewayImage{Reference: gateway.Image, OS: "linux", Architecture: config.Architecture, Protocol: browsergateway.RemoteProtocol, CertificateConfiguration: browsergateway.RemoteCertificateProtocol, InstanceProtocol: ecsidentity.Protocol}
	p, err := NewGateway(GatewayOptions{Worker: Options{Config: config, Store: db, ECS: f, Scheduler: f, Inspector: f, Now: func() time.Time { return f.now }}, Gateway: gateway, Authority: f.exchange, Network: f})
	if err != nil {
		t.Fatal(err)
	}
	_, changes, err := (workerstate.Ledger{Store: db}).Begin(job.Reference)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.CommitAll(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	return p, f, job, initialization
}

func (f *gatewayCloudFixture) InspectGateway(context.Context, string) (hostedexecution.GatewayImage, error) {
	return f.image, nil
}

func (f *gatewayCloudFixture) AdmitGateway(ctx context.Context, config Config, _ GatewayConfig) error {
	return f.Admit(ctx, config)
}

func (f *gatewayCloudFixture) VerifyGatewayTask(ctx context.Context, config Config, _ GatewayConfig, task types.Task) (string, error) {
	return f.address, f.VerifyTask(ctx, config, task)
}

func gatewayRecord(t *testing.T, p *GatewayProvider, job hostedexecution.Job) workerexchange.Record {
	t.Helper()
	row, err := p.control.store.Get(t.Context(), workerexchange.Bucket, job.Reference.ID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := storage.Decode[workerexchange.Record](row)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestECSGatewayLifecycleBindsVerifiedAddressBeforeInitializationAndKeepsCleanupCheckpoint(t *testing.T) {
	p, f, job, initialization := gatewayLifecycle(t, "")
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if _, err := p.control.store.Get(t.Context(), Bucket, job.Reference.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("gateway replaced the worker dispatch journal", err)
	}
	if _, err := f.exchange.GatewayInput(t.Context(), job.Reference.ID, initialization.Token, strings.Repeat("a", 32)); !errors.Is(err, workerexchange.ErrUnavailable) {
		t.Fatal("unverified task received private initialization", err)
	}
	f.networkPending = true
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || !status.AdmissionPending || gatewayRecord(t, p, job).Gateway.Instance != "" {
		t.Fatal("unverified network bound gateway authority", status, err)
	}
	f.networkPending = false
	status, err = p.Observe(t.Context(), job.Reference)
	record := gatewayRecord(t, p, job)
	if err != nil || !status.AdmissionPending || record.Gateway.Instance != strings.Repeat("a", 32) || record.Gateway.Settings.Address != f.address {
		t.Fatal("unknown health did not retain the verified initialization binding", status, err)
	}
	if err := f.exchange.Activate(t.Context(), job.Reference); !errors.Is(err, workerexchange.ErrUnavailable) {
		t.Fatal("worker activated before private initialization", err)
	}
	if _, err := f.exchange.GatewayInput(t.Context(), job.Reference.ID, initialization.Token, record.Gateway.Instance); err != nil {
		t.Fatal("verified task could not initialize", err)
	}
	f.task.HealthStatus = types.HealthStatusHealthy
	status, err = p.Observe(t.Context(), job.Reference)
	if err != nil || status.AdmissionPending || status.InfrastructureFailure {
		t.Fatal("ready gateway remained unadmitted", status, err)
	}
	if err := f.exchange.Activate(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	refs, err := p.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile)
	if err != nil || len(refs) != 1 || refs[0] != job.Reference {
		t.Fatal("gateway inventory lost its original execution", refs, err)
	}
	if err := p.Remove(t.Context(), job.Reference); err == nil {
		t.Fatal("uncheckpointed gateway was removed")
	}
	p.control = reopenProvider(t, p.control, f.cloudFixture)
	f.exchange.Store = p.control.store
	revoke(t, p.control, job, false)
	stopObserved(t, p.control, f.cloudFixture, job)
	if err := p.Remove(t.Context(), job.Reference); err == nil || f.deletes != 0 {
		t.Fatal("revocation replaced the result and evidence checkpoint", err)
	}
	revoke(t, p.control, job, true)
	if err := p.Remove(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	refs, err = p.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile)
	if err != nil || len(refs) != 0 || f.registrations != 1 || f.launches != 1 || f.deletes != 1 || f.schedule != nil {
		t.Fatal("gateway cleanup duplicated or retained native resources", refs, err)
	}
	row, err := p.control.store.Get(t.Context(), GatewayBucket, job.Reference.ID)
	if err != nil || strings.Contains(string(row.Data), job.Ticket.Token) || strings.Contains(string(row.Data), initialization.Token) {
		t.Fatal("gateway journal contains raw execution authority", err)
	}
}

func TestECSGatewayLifecycleRecoversLostAcknowledgementsWithoutReplay(t *testing.T) {
	for _, mode := range []string{"lost-register", "lost-launch", "lost-deadline", "launch-rejected"} {
		t.Run(mode, func(t *testing.T) {
			p, f, job, _ := gatewayLifecycle(t, mode)
			if err := p.Ensure(t.Context(), job); err == nil {
				t.Fatal("fixture did not lose its acknowledgement")
			}
			p.control = reopenProvider(t, p.control, f.cloudFixture)
			f.exchange.Store = p.control.store
			_ = p.Ensure(t.Context(), job)
			revoke(t, p.control, job, false)
			if mode == "lost-launch" || mode == "lost-deadline" {
				stopObserved(t, p.control, f.cloudFixture, job)
			}
			revoke(t, p.control, job, true)
			if err := p.Remove(t.Context(), job.Reference); err != nil {
				t.Fatal(err)
			}
			if f.registrations != 1 || f.launches > 1 || f.schedules > 1 || f.deletes != 1 {
				t.Fatal("recovery created another gateway")
			}
		})
	}
}

func TestECSGatewayLifecycleRejectsUnsupportedCapabilitiesAndForeignNetworkBindings(t *testing.T) {
	for _, field := range []string{"transport", "certificate", "instance", "architecture", "network"} {
		t.Run(field, func(t *testing.T) {
			p, f, job, _ := gatewayLifecycle(t, "")
			switch field {
			case "transport":
				f.image.Protocol = ""
			case "certificate":
				f.image.CertificateConfiguration = ""
			case "instance":
				f.image.InstanceProtocol = ""
			case "architecture":
				f.image.Architecture = "amd64"
			case "network":
				f.networkRejected = true
			}
			if err := p.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrRejected) || f.registrations != 0 || f.launches != 0 {
				t.Fatal("unsupported gateway created native resources", err)
			}
		})
	}
	for _, address := range []string{"127.0.0.1", "169.254.170.2", "8.8.8.8", "10.20.1.8"} {
		t.Run(address, func(t *testing.T) {
			p, f, job, _ := gatewayLifecycle(t, "")
			if err := p.Ensure(t.Context(), job); err != nil {
				t.Fatal(err)
			}
			f.address = address
			if _, err := p.Observe(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrOwnership) || gatewayRecord(t, p, job).Gateway.Instance != "" {
				t.Fatal("foreign task address acquired initialization authority", err)
			}
		})
	}
}

func TestECSGatewayLifecycleKeepsAuthorityAndDeadlineAdmissionClosed(t *testing.T) {
	p, f, job, _ := gatewayLifecycle(t, "")
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	f.task.HealthStatus = types.HealthStatusHealthy
	f.allowed = false
	status, err := p.Observe(t.Context(), job.Reference)
	if err != nil || !status.AdmissionPending || gatewayRecord(t, p, job).Gateway.Instance != "" {
		t.Fatal("withdrawn run authority bound a gateway", status, err)
	}
	f.allowed = true
	f.schedule.Target.RoleArn = aws.String(p.control.config.ExecutionRole)
	if _, err := p.Observe(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrOwnership) || gatewayRecord(t, p, job).Gateway.Instance != "" {
		t.Fatal("an unverified independent deadline bound a gateway", err)
	}
}

func TestECSGatewayLifecycleRejectsReplacementAddressAndRoleConfiguration(t *testing.T) {
	p, f, job, _ := gatewayLifecycle(t, "")
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Observe(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	f.address = "10.20.1.8"
	f.task.Containers[0].NetworkInterfaces[0].PrivateIpv4Address = aws.String(f.address)
	if _, err := p.Observe(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrOwnership) || gatewayRecord(t, p, job).Gateway.Settings.Address != "10.20.1.7" {
		t.Fatal("a replacement interface changed the frozen transport", err)
	}
	p.control.gateway.config.KeySecret += "changed"
	if _, err := p.Observe(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("another gateway configuration acquired the durable task", err)
	}
	if err := p.Stop(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrOwnership) || f.stops != 0 {
		t.Fatal("another gateway configuration stopped the durable task", err)
	}
}
