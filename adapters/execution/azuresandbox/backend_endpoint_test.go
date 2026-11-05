package azuresandbox

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func backendIP(w *placementWorld) map[string]any {
	e := w.s.Placement.Services.BackendEndpoint
	return w.resources[e.Interface]["properties"].(map[string]any)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)
}

func TestSandboxBackendEndpointDefinitionsAndSettingsBindRelaySourceBeforeCredentials(t *testing.T) {
	s := runtimeSettings()
	defs, err := s.BackendEndpointDefinitions()
	if err != nil || len(defs) != 2 {
		t.Fatal("missing backend network definitions", err)
	}
	e := s.Placement.Services.BackendEndpoint
	rules := defs[e.SecurityGroup].(map[string]any)["properties"].(map[string]any)["securityRules"].([]any)
	allow := rules[0].(map[string]any)["properties"].(map[string]any)
	if len(rules) != 3 || allow["sourceAddressPrefix"] != "10.60.0.0/23" || allow["destinationAddressPrefix"] != "10.60.2.4" || allow["destinationPortRange"] != "443" || allow["protocol"] != "Tcp" || allow["direction"] != "Inbound" {
		t.Fatal("backend network policy widened relay access")
	}
	for _, mode := range []string{"missing", "worker-alias", "subnet-overlap", "source-overlap", "foreign-network", "reserved-address", "wrong-address", "security-alias", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			e := &s.Placement.Services.BackendEndpoint
			switch mode {
			case "missing":
				*e = BackendEndpointSettings{}
			case "worker-alias":
				e.Endpoint = s.Placement.Private.ExchangeEndpoint
			case "subnet-overlap":
				e.SubnetPrefix = e.RelaySubnetPrefix
			case "source-overlap":
				e.RelaySubnetPrefix = s.Placement.SubnetPrefix
			case "foreign-network":
				e.Subnet = s.Placement.Private.Subnet
			case "reserved-address":
				s.Placement.Services.Backend.Address = "10.60.2.3"
			case "wrong-address":
				s.Placement.Services.Backend.Address = "10.60.3.4"
			case "security-alias":
				e.SecurityGroup = s.Placement.Private.SecurityGroup
			case "profile":
				e.Endpoint += "-other"
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			f := newControllerFixture(t, "go", "")
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid backend endpoint constructed a credential")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid backend endpoint accepted", err)
			}
		})
	}
}

func TestSandboxBackendEndpointRejectsAddressServiceMemberApprovalAndNetworkDrift(t *testing.T) {
	for _, mode := range []string{"address", "service", "member", "service-member", "approval", "embedded-approval", "public-environment", "public-ip", "subnet-security", "subnet-extra", "source-prefix", "source-delegation", "security-source", "security-port", "pending-approval", "missing-links"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			v, e := s.Placement.Services, s.Placement.Services.BackendEndpoint
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "address":
				backendIP(w)["privateIPAddress"] = "10.60.2.5"
			case "service":
				props(e.Endpoint)["privateLinkServiceConnections"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceId"] = s.Placement.Private.ExchangeEnvironment
			case "member":
				backendIP(w)["privateLinkConnectionProperties"].(map[string]string)["requiredMemberName"] = "other-member"
			case "service-member":
				w.pages[v.Backend.Environment+"/privateLinkResources"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["requiredMembers"] = []string{"other-member"}
			case "approval":
				w.pages[v.Backend.Environment+"/privateEndpointConnections"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceConnectionState"].(map[string]string)["status"] = "Rejected"
			case "embedded-approval":
				props(v.Backend.Environment)["privateEndpointConnections"] = []any{}
				expected = hostedexecution.ErrUnavailable
			case "public-environment":
				props(v.Backend.Environment)["publicNetworkAccess"] = "Enabled"
			case "public-ip":
				backendIP(w)["publicIPAddress"] = map[string]string{"id": "other"}
			case "subnet-security":
				props(e.Subnet)["networkSecurityGroup"] = map[string]string{"id": "other"}
			case "subnet-extra":
				props(e.Subnet)["privateEndpoints"] = append(props(e.Subnet)["privateEndpoints"].([]any), map[string]string{"id": "other"})
			case "source-prefix":
				props(v.Relay.Subnet)["addressPrefix"] = "10.61.0.0/23"
			case "source-delegation":
				props(v.Relay.Subnet)["delegations"] = []any{}
			case "security-source":
				props(e.SecurityGroup)["securityRules"].([]any)[0].(map[string]any)["properties"].(map[string]any)["sourceAddressPrefix"] = "*"
			case "security-port":
				props(e.SecurityGroup)["securityRules"].([]any)[0].(map[string]any)["properties"].(map[string]any)["destinationPortRange"] = "*"
			case "pending-approval":
				w.pages[v.Backend.Environment+"/privateEndpointConnections"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceConnectionState"].(map[string]string)["status"] = "Pending"
				expected = hostedexecution.ErrUnavailable
			case "missing-links":
				w.pages[v.Backend.Environment+"/privateLinkResources"]["value"] = []any{}
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKBackendEndpoint(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("unsafe backend endpoint admitted", err)
			}
		})
	}
}

func TestSandboxBackendEndpointSharesEvidenceAndFreshFailureInvalidatesIt(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKBackendEndpoint(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	now := time.Now()
	reader.now = func() time.Time { return now }
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			if err := reader.CheckRunning(t.Context(), s.Worker); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if len(w.calls) != 8 {
		t.Fatal("backend endpoint scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("concurrent endpoint checks repeated reads")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable || strings.Contains(err.Error(), "canary") {
		t.Fatal("missing endpoint evidence became worker absence or exposed diagnostics", err)
	}
	w.status = 0
	backendIP(w)["privateIPAddress"] = "10.60.2.5"
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh endpoint admission used cached address", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("current failed endpoint evidence retained earlier success", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled endpoint check returned evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed endpoint check returned evidence", err)
	}
}

func TestSandboxRuntimeBackendEndpointGateBlocksAllAdmissionPaths(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings("discovery")
	w := newPlacementWorld(t, s)
	protection := &placementProtection{}
	r, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: protection, ImageAdmission: protection, ARMTransport: w, Credentials: func(string) (azcore.TokenCredential, error) {
		return &runtimeCredential{token: "placement-metadata-canary"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	backendIP(w)["privateIPAddress"] = "10.60.2.5"
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("changed endpoint reached image import", err)
	}
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("changed endpoint reached sandbox creation", err)
	}
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("changed endpoint reached live command admission", err)
	}
	r.Close()
	if err := r.backendEndpoint.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left endpoint reader open", err)
	}
}
