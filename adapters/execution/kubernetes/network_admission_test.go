package kubernetes

import (
	"context"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type networkImageInspector struct{ imageInspector }

func (i networkImageInspector) InspectImage(ctx context.Context, ref string) (hostedexecution.Image, error) {
	image, err := i.imageInspector.InspectImage(ctx, ref)
	image.GatewayTransport, image.APIPolicy = browsergateway.RemoteProtocol, "request-policy/v1"
	image.SupervisorRevision = strings.Repeat("c", 64)
	return image, err
}

func (i networkImageInspector) InspectGateway(_ context.Context, ref string) (hostedexecution.GatewayImage, error) {
	return hostedexecution.GatewayImage{Reference: ref, OS: "linux", Architecture: "arm64", Protocol: browsergateway.RemoteProtocol}, nil
}

type restrictedNetworkRuntime struct {
	networkImageInspector
	transport, policy string
}

func (i restrictedNetworkRuntime) InspectImage(ctx context.Context, ref string) (hostedexecution.Image, error) {
	image, err := i.networkImageInspector.InspectImage(ctx, ref)
	image.GatewayTransport, image.APIPolicy = i.transport, i.policy
	return image, err
}

type automaticGatewayAuthority struct {
	*workerexchange.Service
	fixture *networkFixture
	t       *testing.T
}

func (a automaticGatewayAuthority) BindGateway(ctx context.Context, ref execution.Reference, instance string) error {
	if err := a.Service.BindGateway(ctx, ref, instance); err != nil {
		return err
	}
	a.fixture.initialize(a.t)
	return nil
}

func TestManagedEnvironmentExecutionUsesApprovedGatewayAndSharedResultLifecycle(t *testing.T) {
	n := newNetworkFixture(t, false)
	n.base.mode = "passed"
	n.base.provider.gatewayAuthority = automaticGatewayAuthority{Service: n.base.exchange, fixture: n, t: t}
	driver := &hostedexecution.Executor{Provider: n.base.provider, Exchange: n.base.exchange, Environment: true, Name: "kubernetes", Runner: "playwright", ExchangeOrigin: n.base.provider.config.ExchangeOrigin, ApproveArtifact: n.base.driver.ApproveArtifact}
	profile, err := driver.TimingProfile(t.Context(), n.base.input.Artifact)
	if err != nil || profile == "" {
		t.Fatal("approved environment timing profile unavailable", err)
	}
	if err := driver.Admit(t.Context(), n.base.input.Test, n.base.input.Artifact); err != nil {
		t.Fatal(err)
	}
	report := driver.Execute(t.Context(), n.base.input)
	if report.RecoveryError != nil || report.Result.State != "passed" || report.Result.DurationMs != 1 || len(report.Attachments) != 1 {
		t.Fatal("normal hosted environment lifecycle lost the test observation", report.Result, report.RecoveryError)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.objects) != 0 || n.base.value != nil {
		t.Fatal("managed environment execution retained its resource bundle")
	}
}

func TestEnvironmentAdmissionRejectsRuntimeWithoutGatewayCapability(t *testing.T) {
	n := newNetworkFixture(t, false)
	driver := &hostedexecution.Executor{Provider: n.base.provider, Exchange: n.base.exchange, Environment: true, Name: "kubernetes", Runner: "playwright", ExchangeOrigin: n.base.provider.config.ExchangeOrigin, ApproveArtifact: n.base.driver.ApproveArtifact}
	n.base.provider.inspector = restrictedNetworkRuntime{networkImageInspector: networkImageInspector{imageInspector{"playwright"}}}
	if err := driver.Admit(t.Context(), n.base.input.Test, n.base.input.Artifact); err == nil {
		t.Fatal("old source runtime admitted environment execution")
	}
	if n.base.posts != 0 {
		t.Fatal("failed environment admission dispatched code")
	}
}
