package azuresandbox

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxHostRegistryBindsBothModesAndRejectsSettingsDrift(t *testing.T) {
	for _, mode := range []string{"execution", "discovery"} {
		s := runtimeSettings(mode)
		w := newPlacementWorld(t, s)
		reader, err := NewSDKHostRegistry(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		if err := reader.Check(t.Context(), s.Worker); err != nil || len(w.calls) != 13 {
			t.Fatal("private controller registry rejected", err, len(w.calls))
		}
		if err := reader.CheckRunning(t.Context(), s.Worker); err != nil || len(w.calls) != 13 || w.calls[s.Host.Registry.Endpoint] != 1 {
			t.Fatal("successful live evidence not shared", err)
		}
	}
	for _, mode := range []string{"missing", "undeclared", "foreign", "alias", "subnet", "overlap", "reserved-ip", "same-ip", "zone", "assignment"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			switch mode {
			case "missing":
				s.Host.Registry = RelayRegistrySettings{}
			case "undeclared":
				s.Placement.RegistryConsumers = nil
			case "foreign":
				s.Host.Registry.Interface = strings.Replace(s.Host.Registry.Interface, s.Worker.Subscription, "foreign", 1)
			case "alias":
				s.Host.Registry.SecurityGroup = s.Placement.Services.RelayNetwork.Registry.SecurityGroup
			case "subnet":
				s.Host.Registry.Subnet = s.Host.Subnet
			case "overlap":
				s.Host.SubnetPrefix = s.Host.Registry.SubnetPrefix
			case "reserved-ip":
				s.Host.Registry.LoginIP = "10.72.3.2"
			case "same-ip":
				s.Host.Registry.DataIP = s.Host.Registry.LoginIP
			case "zone":
				s.Host.Registry.Zone = s.Placement.Private.RegistryZone
			case "assignment":
				s.Host.Registry.PullAssignment = s.Placement.PullAssignment
			}
			s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			s.Host.Profile = s.HostProfile()
			if _, err := NewSDKHostRegistry(s, &runtimeCredential{}, nil); err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid registry settings admitted", err)
			}
		})
	}
}

func TestSandboxHostRegistryRejectsEndpointDNSGrantAndSourceDrift(t *testing.T) {
	for _, mode := range []string{"source", "address", "member", "service", "subnet", "security", "approval", "embedded", "grant", "identity", "dns", "link", "page"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			h, r := s.Host, s.Host.Registry
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			switch mode {
			case "source":
				props(h.Subnet)["addressPrefix"] = "10.74.0.0/23"
			case "address":
				props(r.Interface)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateIPAddress"] = "10.72.3.6"
			case "member":
				props(r.Interface)["ipConfigurations"].([]any)[1].(map[string]any)["properties"].(map[string]any)["privateLinkConnectionProperties"].(map[string]any)["requiredMemberName"] = "foreign"
			case "service":
				props(r.Endpoint)["privateLinkServiceConnections"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceId"] = s.Placement.Registry + "-foreign"
			case "subnet":
				props(r.Subnet)["privateEndpoints"] = []any{}
			case "security":
				props(r.SecurityGroup)["securityRules"].([]any)[0].(map[string]any)["properties"].(map[string]any)["sourceAddressPrefix"] = "*"
			case "approval":
				props(r.Connection)["privateEndpoint"] = map[string]string{"id": s.Placement.Private.RegistryEndpoint}
			case "embedded":
				props(s.Placement.Registry)["privateEndpointConnections"] = []any{}
			case "grant":
				props(r.PullAssignment)["principalId"] = s.Authority.Control.PrincipalID
			case "identity":
				props(h.Pull.Resource)["tenantId"] = "foreign"
			case "dns":
				w.pages[r.Zone+"/ALL"]["value"].([]any)[1].(map[string]any)["properties"].(map[string]any)["aRecords"] = []any{map[string]string{"ipv4Address": "10.72.3.9"}}
			case "link":
				w.pages[r.Zone+"/virtualNetworkLinks"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["virtualNetwork"] = map[string]string{"id": s.Placement.virtualNetwork()}
			case "page":
				w.pages[r.Zone+"/ALL"]["nextLink"] = "https://foreign.example/private-canary"
			}
			reader, err := NewSDKHostRegistry(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			err = reader.Check(t.Context(), s.Worker)
			if err != hostedexecution.ErrOwnership && err != hostedexecution.ErrUnavailable {
				t.Fatal("unsafe private registry admitted", err)
			}
			if strings.Contains(err.Error(), "canary") {
				t.Fatal("diagnostics leaked")
			}
		})
	}
}

func TestSandboxRuntimeHostRegistryFreshFailureFencesAllAdmissions(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings("discovery")
	w := newPlacementWorld(t, s)
	p := &placementProtection{}
	runtime, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: p, ImageAdmission: p, ARMTransport: w, Credentials: func(string) (azcore.TokenCredential, error) {
		return &runtimeCredential{token: "placement-metadata-canary"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	w.resources[s.Host.Registry.PullAssignment]["properties"].(map[string]any)["principalId"] = s.Authority.Control.PrincipalID
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || p.imports.Load() != 0 {
		t.Fatal("grant drift reached import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || p.creates.Load() != 1 {
		t.Fatal("grant drift reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || p.verifies.Load() != 0 {
		t.Fatal("fresh failure retained live evidence", err)
	}
	runtime.Close()
	if err := runtime.hostRegistry.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("registry reader remained open", err)
	}
}
