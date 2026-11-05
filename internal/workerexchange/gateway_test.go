package workerexchange_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	workerhttp "github.com/wtiger001/teststudio/adapters/workerexchange/http"
	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type gatewaySecrets struct{ resolutions atomic.Int32 }

func (s *gatewaySecrets) References(context.Context, string, string) ([]secrets.Reference, error) {
	return nil, secrets.ErrUnavailable
}

func (s *gatewaySecrets) Resolve(_ context.Context, project, reference, purpose string) (secrets.Value, error) {
	if project != "project" || reference != "approved-reader" || purpose != secrets.APICredential {
		return secrets.Value{}, secrets.ErrUnavailable
	}
	s.resolutions.Add(1)
	return secrets.Value{Origins: []string{"https://application.test"}, Bytes: []byte("trusted-application-canary"), Network: networkguard.Public}, nil
}

func gatewayFixture(t *testing.T) (fixture, *gatewaySecrets) {
	t.Helper()
	f := fresh(t)
	f.ref.Gateway = "owned-gateway"
	f.input.AwaitDispatch = true
	policy := &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "reader", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/approved", ReadOnly: true}}, MaxRequests: 2, MaxResponseBytes: 4096}
	f.input.Browser = &browserpolicy.Target{Origin: "https://application.test", Authenticated: true, Requests: policy}
	f.input.Artifact.Browser = &browserpolicy.Spec{Endpoint: "api", Requests: policy}
	f.input.CredentialReference = "approved-reader"
	provider := &gatewaySecrets{}
	f.service.Secrets = provider
	return f, provider
}

func TestGatewayInitializationSeparatesWorkerTransportAndApplicationAuthority(t *testing.T) {
	f, provider := gatewayFixture(t)
	worker := f.prepare(t)
	settings := workerexchange.GatewaySettings{Origin: "https://owned-gateway.test:8044"}
	gateway, err := f.service.PrepareGateway(t.Context(), f.ref, settings)
	if err != nil {
		t.Fatal(err)
	}
	if gateway.Token == worker.Token || strings.Contains(fmt.Sprintf("%+v %#v", gateway, gateway), gateway.Token) {
		t.Fatal("gateway initialization authority is shared or disclosed")
	}
	if _, err := f.service.GatewayInput(t.Context(), f.ref.ID, gateway.Token, "gateway-pod"); !errors.Is(err, workerexchange.ErrUnavailable) || provider.resolutions.Load() != 0 {
		t.Fatal("unverified gateway received credentials", err)
	}
	if err := f.service.Activate(t.Context(), f.ref); !errors.Is(err, workerexchange.ErrUnavailable) {
		t.Fatal("worker activated before gateway identity and initialization", err)
	}
	if err := f.service.BindGateway(t.Context(), f.ref, "gateway-pod"); err != nil {
		t.Fatal(err)
	}
	if err := f.service.BindGateway(t.Context(), f.ref, "replacement-pod"); !errors.Is(err, workerexchange.ErrConflict) {
		t.Fatal("gateway instance was replaced", err)
	}
	if err := f.service.Activate(t.Context(), f.ref); !errors.Is(err, workerexchange.ErrUnavailable) {
		t.Fatal("worker activated before gateway configuration", err)
	}
	server := httptest.NewTLSServer(workerhttp.Handler(f.service))
	defer server.Close()
	newClient := func(token, instance string) *workerclient.Client {
		t.Helper()
		client, err := workerclient.New(server.URL, f.ref.ID, instance, token, server.Client().Transport)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(client.Close)
		return client
	}
	workerClient := newClient(worker.Token, "runner-pod")
	gatewayClient := newClient(gateway.Token, "gateway-pod")
	if _, err := workerClient.GatewayInput(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("worker fetched application credential", err)
	}
	if _, err := gatewayClient.Input(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("gateway fetched uploaded code", err)
	}
	if _, err := newClient(gateway.Token, "replacement-pod").GatewayInput(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("wrong gateway instance fetched credential", err)
	}
	configuration, err := gatewayClient.GatewayInput(t.Context())
	if err != nil || configuration.Credential == nil || configuration.Credential.Value != "Bearer trusted-application-canary" || configuration.Remote == nil {
		t.Fatal("trusted gateway initialization failed", err)
	}
	if err := f.service.Activate(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
	packet, err := workerClient.Input(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	connection, err := workerClient.GatewayConnection(t.Context())
	if err != nil || connection.Origin != settings.Origin || connection.Session.Token != configuration.Remote.Token || connection.Session.Token == worker.Token || connection.Session.Token == gateway.Token {
		t.Fatal("bridge received wrong execution-scoped transport authority", err)
	}
	for _, private := range []string{"trusted-application-canary", "approved-reader", worker.Token, gateway.Token, connection.Session.Token} {
		if strings.Contains(string(packet.Input), private) {
			t.Fatal("private gateway data entered uploaded runner input")
		}
	}
	bridgeJSON, _ := json.Marshal(connection)
	if strings.Contains(string(bridgeJSON), "trusted-application-canary") {
		t.Fatal("bridge connection contains application credential")
	}
	if _, err := gatewayClient.GatewayConnection(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("gateway initialization ticket controls runner bridge", err)
	}
	transportClient := newClient(connection.Session.Token, "gateway-pod")
	if _, err := transportClient.GatewayInput(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("proxy transport capability fetched application credential", err)
	}
	if _, err := transportClient.Input(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("proxy transport capability fetched uploaded code", err)
	}
	if _, err := newClient(worker.Token, "another-runner").GatewayConnection(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("bridge crossed runner instance", err)
	}
	row, err := f.store.Get(t.Context(), workerexchange.Bucket, f.ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := storage.Decode[workerexchange.Record](row)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(record)
	for _, private := range []string{"trusted-application-canary", worker.Token, gateway.Token, connection.Session.Token} {
		if strings.Contains(string(data), private) {
			t.Fatal("durable exchange contains private credential or token")
		}
	}
	if err := f.service.Revoke(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
	if _, err := gatewayClient.GatewayInput(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("revoked gateway fetched application credential", err)
	}
	if _, err := workerClient.GatewayConnection(t.Context()); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("revoked worker fetched bridge connection", err)
	}
}

func TestGatewayRestartPreservesFrozenBindingAndWithdrawnAuthority(t *testing.T) {
	f, provider := gatewayFixture(t)
	worker := f.prepare(t)
	settings := workerexchange.GatewaySettings{Origin: "https://owned-gateway.test:8044"}
	gateway, err := f.service.PrepareGateway(t.Context(), f.ref, settings)
	if err != nil {
		t.Fatal(err)
	}
	restarted := &workerexchange.Service{Store: f.store, Blobs: f.blobs, Authorize: f.service.Authorize, Secrets: provider}
	again, err := restarted.PrepareGateway(t.Context(), f.ref, settings)
	if err != nil || again.Token != gateway.Token {
		t.Fatal("restart changed gateway initialization authority", err)
	}
	changed := f.input
	changed.CredentialReference = "different-reader"
	if _, err := restarted.Prepare(t.Context(), f.ref, changed, f.deadline); !errors.Is(err, workerexchange.ErrConflict) {
		t.Fatal("retry changed frozen application credential binding", err)
	}
	settings.Origin = "https://different-gateway.test:8044"
	if _, err := restarted.PrepareGateway(t.Context(), f.ref, settings); !errors.Is(err, workerexchange.ErrConflict) {
		t.Fatal("retry changed gateway origin", err)
	}
	if err := restarted.BindGateway(t.Context(), f.ref, "gateway-pod"); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.GatewayInput(t.Context(), f.ref.ID, gateway.Token, "gateway-pod"); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Activate(t.Context(), f.ref); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Input(t.Context(), f.ref.ID, worker.Token, "runner-pod"); err != nil {
		t.Fatal(err)
	}
	f.revokeAuthority(t)
	before := provider.resolutions.Load()
	if _, err := restarted.GatewayInput(t.Context(), f.ref.ID, gateway.Token, "gateway-pod"); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("withdrawn run authority delivered a credential", err)
	}
	if _, err := restarted.GatewayConnection(t.Context(), f.ref.ID, worker.Token, "runner-pod"); !errors.Is(err, workerexchange.ErrDenied) {
		t.Fatal("withdrawn run authority delivered a bridge", err)
	}
	if provider.resolutions.Load() != before {
		t.Fatal("withdrawn authority reached credential resolver")
	}
}
