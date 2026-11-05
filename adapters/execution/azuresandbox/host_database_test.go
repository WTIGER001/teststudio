package azuresandbox

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxHostDatabaseBindsBothModesConsumersAndExactNetworkSettings(t *testing.T) {
	for _, mode := range []string{"execution", "discovery"} {
		s := runtimeSettings(mode)
		s.Host.Database.Consumers = []string{s.Host.Database.Endpoint + "-api"}
		s.Host.Profile = s.HostProfile()
		w := newPlacementWorld(t, s)
		defs, err := s.HostDatabaseDefinitions()
		if err != nil || len(defs) != 5 {
			t.Fatal("missing database network bodies", err, len(defs))
		}
		reader, err := NewSDKHostDatabase(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		s.Host.Database.Consumers[0] += "-changed"
		if err := reader.Check(t.Context(), s.Worker); err != nil || len(w.calls) != 10 {
			t.Fatal("bound database path rejected", err, len(w.calls))
		}
		if err := reader.CheckRunning(t.Context(), s.Worker); err != nil || w.calls[s.Host.Database.Server] != 1 {
			t.Fatal("successful evidence not shared", err)
		}
		for _, raw := range w.pages[s.Host.Database.Server+"/privateEndpointConnections"]["value"].([]any) {
			raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Ready"
		}
		if err := reader.Check(t.Context(), s.Worker); err != nil {
			t.Fatal("documented Ready connection state rejected", err)
		}
	}
	for _, mode := range []string{"missing", "hostname", "public-address", "reserved-address", "overlap", "foreign", "alias", "consumer-alias", "consumer-duplicate", "registry-consumer", "zone"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			switch mode {
			case "missing":
				s.Host.Database = HostDatabaseSettings{}
			case "hostname":
				s.Host.DatabaseHost = "foreign.postgres.database.azure.com"
			case "public-address":
				s.Host.Database.Address = "203.0.113.44"
			case "reserved-address":
				s.Host.Database.Address = "10.72.6.2"
			case "overlap":
				s.Host.Database.SubnetPrefix = s.Host.SubnetPrefix
			case "foreign":
				s.Host.Database.Server = strings.Replace(s.Host.Database.Server, s.Worker.Subscription, "foreign", 1)
			case "alias":
				s.Host.Database.Endpoint = s.Host.Registry.Endpoint
			case "consumer-alias":
				s.Host.Database.Consumers = []string{s.Host.Registry.Endpoint}
			case "consumer-duplicate":
				s.Host.Database.Consumers = []string{s.Host.Database.Endpoint}
			case "registry-consumer":
				s.Placement.RegistryConsumers = append(s.Placement.RegistryConsumers, s.Host.Database.Endpoint)
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			case "zone":
				s.Host.Database.Zone = s.Host.Registry.Zone
			}
			s.Host.Profile = s.HostProfile()
			if _, err := NewSDKHostDatabase(s, &runtimeCredential{}, nil); err != hostedexecution.ErrConfiguration {
				t.Fatal("unsafe database settings admitted", err)
			}
		})
	}
}

func TestSandboxHostDatabaseRejectsServiceEndpointPolicyDNSAndInventoryDrift(t *testing.T) {
	for _, mode := range []string{"hostname", "public", "authentication", "state", "endpoint", "address", "member", "subnet", "source", "port", "consumer", "missing-consumer", "duplicate-consumer", "pending", "unknown-state", "approval-group", "embedded", "member-list", "zone-list", "dns", "link", "continuation"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			d := s.Host.Database
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			connections := w.pages[d.Server+"/privateEndpointConnections"]["value"].([]any)
			connection := connections[0].(map[string]any)["properties"].(map[string]any)
			switch mode {
			case "hostname":
				props(d.Server)["fullyQualifiedDomainName"] = "foreign.postgres.database.azure.com"
			case "public":
				props(d.Server)["network"].(map[string]any)["publicNetworkAccess"] = "Enabled"
			case "authentication":
				props(d.Server)["authConfig"].(map[string]any)["passwordAuth"] = "Disabled"
			case "state":
				props(d.Server)["state"] = "Updating"
			case "endpoint":
				props(d.Endpoint)["privateLinkServiceConnections"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceId"] = "foreign"
			case "address":
				props(d.Interface)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateIPAddress"] = "10.72.6.9"
			case "member":
				props(d.Interface)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkConnectionProperties"].(map[string]any)["requiredMemberName"] = "foreign"
			case "subnet":
				props(d.Subnet)["privateEndpoints"] = []any{}
			case "source":
				props(d.SecurityGroup)["securityRules"].([]any)[0].(map[string]any)["properties"].(map[string]any)["sourceAddressPrefix"] = "*"
			case "port":
				props(d.SecurityGroup)["securityRules"].([]any)[0].(map[string]any)["properties"].(map[string]any)["destinationPortRange"] = "*"
			case "consumer":
				connection["privateEndpoint"] = map[string]string{"id": s.Host.Registry.Endpoint}
			case "missing-consumer":
				w.pages[d.Server+"/privateEndpointConnections"]["value"] = []any{}
			case "duplicate-consumer":
				w.pages[d.Server+"/privateEndpointConnections"]["value"] = append(connections, connections[0])
			case "pending":
				connection["privateLinkServiceConnectionState"].(map[string]any)["status"] = "Pending"
			case "unknown-state":
				connection["provisioningState"] = "Unknown"
			case "approval-group":
				connection["groupIds"] = []string{"foreign"}
			case "embedded":
				props(d.Server)["privateEndpointConnections"] = []any{}
			case "member-list":
				w.pages[d.Server+"/privateLinkResources"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["requiredMembers"] = []string{"foreign"}
			case "zone-list":
				w.pages[d.Server+"/privateLinkResources"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["requiredZoneNames"] = []string{"foreign"}
			case "dns":
				w.pages[d.Zone+"/ALL"]["value"].([]any)[1].(map[string]any)["properties"].(map[string]any)["aRecords"] = []any{map[string]string{"ipv4Address": "10.72.6.9"}}
			case "link":
				w.pages[d.Zone+"/virtualNetworkLinks"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["virtualNetwork"] = map[string]string{"id": s.Placement.virtualNetwork()}
			case "continuation":
				w.pages[d.Server+"/privateEndpointConnections"]["nextLink"] = "https://foreign.example/private-canary"
			}
			reader, err := NewSDKHostDatabase(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			err = reader.Check(t.Context(), s.Worker)
			if err != hostedexecution.ErrOwnership && err != hostedexecution.ErrUnavailable {
				t.Fatal("unsafe database path admitted", err)
			}
			if strings.Contains(err.Error(), "canary") {
				t.Fatal("private provider diagnostics disclosed")
			}
		})
	}
}

func TestSandboxRuntimeHostDatabaseFreshFailureFencesAllAdmissionAndClosesReader(t *testing.T) {
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
	w.resources[s.Host.Database.Server]["properties"].(map[string]any)["network"].(map[string]any)["publicNetworkAccess"] = "Enabled"
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || p.imports.Load() != 0 {
		t.Fatal("database drift reached import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || p.creates.Load() != 1 {
		t.Fatal("database drift reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || p.verifies.Load() != 0 {
		t.Fatal("fresh database failure retained live evidence", err)
	}
	runtime.Close()
	if err := runtime.hostDatabase.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("database reader remained open", err)
	}
}
