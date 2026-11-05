package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxRelayNetworkJoinsPrivatePath(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKRelayNetwork(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := reader.Check(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	if len(w.calls) != 28 {
		t.Fatal("unexpected relay observation scope", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("relay metadata was retried")
		}
	}
	request, _ := http.NewRequest("POST", "https://management.azure.com"+s.Placement.Services.RelayNetwork.virtualNetwork(s)+"/listDnsResolverPolicies?api-version=2025-05-01", strings.NewReader(`{"write":true}`))
	if _, err := reader.http.Do(request); err != hostedexecution.ErrOwnership {
		t.Fatal("relay read action admitted a payload", err)
	}
}

func TestSandboxRelayNetworkDefinitionsSeparateTrustedPlatformFromSandbox(t *testing.T) {
	s := runtimeSettings()
	defs, err := s.RelayNetworkDefinitions()
	if err != nil || len(defs) != 21 {
		t.Fatal("missing relay packaging inputs", err, len(defs))
	}
	r := s.Placement.Services.RelayNetwork
	rules := defs[r.SecurityGroup].(map[string]any)["properties"].(map[string]any)["securityRules"].([]any)
	var destinations []any
	for _, raw := range rules {
		rule := raw.(map[string]any)
		if rule["name"] == "relay-private-endpoints" {
			destinations = rule["properties"].(map[string]any)["destinationAddressPrefixes"].([]any)
		}
	}
	if hash(destinations) != hash([]string{"10.60.2.4", "10.60.3.4", "10.60.3.5"}) {
		t.Fatal("relay private destinations widened", destinations)
	}
	domains := r.resolverBinding(s).Domains
	if len(domains) != 13 || slices.Contains(domains, strings.TrimPrefix(s.Placement.Services.Backend.Origin, "https://")) || !slices.Contains(domains, "mcr.microsoft.com") || !slices.Contains(domains, "studiotest.westus2.data.azurecr.io") {
		t.Fatal("relay DNS does not match platform and private registry needs")
	}
	worker, _ := json.Marshal(s.Placement.networkRules())
	if strings.Contains(string(worker), "MicrosoftContainerRegistry") || strings.Contains(string(worker), "AzureActiveDirectory") || strings.Contains(string(worker), "AzureFrontDoor") {
		t.Fatal("trusted platform access entered sandbox policy")
	}
	group, _ := json.Marshal(defs[r.Topology.Policy+"/ruleCollectionGroups/relay-platform"])
	if !strings.Contains(string(group), "relay-deny-other-applications") || !strings.Contains(string(group), "mcr.microsoft.com") {
		t.Fatal("relay firewall lost fixed platform boundary")
	}
}

func TestSandboxRelayNetworkInvalidSettingsRejectBeforeCredentials(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	for _, mode := range []string{"missing", "overlap", "registry-reserved", "firewall-reserved", "registry-same-ip", "subnet-alias", "firewall-subnet-alias", "security-alias", "resolver-alias", "zone-alias", "grant-alias", "foreign-grant", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			r := &s.Placement.Services.RelayNetwork
			switch mode {
			case "missing":
				*r = RelayNetworkSettings{}
			case "overlap":
				r.Topology.Prefix = s.Placement.Network.Topology.Prefix
			case "registry-reserved":
				r.Registry.LoginIP = "10.60.3.3"
			case "firewall-reserved":
				r.FirewallIP = "10.60.4.63"
			case "registry-same-ip":
				r.Registry.DataIP = r.Registry.LoginIP
			case "subnet-alias":
				r.Registry.Subnet = s.Placement.Services.BackendEndpoint.Subnet
			case "firewall-subnet-alias":
				r.Registry.Subnet = r.virtualNetwork(s) + "/subnets/AzureFirewallSubnet"
			case "security-alias":
				r.SecurityGroup = s.Placement.SecurityGroup
			case "resolver-alias":
				r.Resolver = s.Placement.Network.Resolver
			case "zone-alias":
				r.Registry.Zone = s.Placement.Private.RegistryZone
			case "grant-alias":
				r.Registry.PullAssignment = s.Placement.PullAssignment
			case "foreign-grant":
				r.Registry.PullAssignment = strings.Replace(r.Registry.PullAssignment, s.Worker.Subscription, "foreign", 1)
			case "profile":
				r.FirewallIP = "10.60.4.5"
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid relay network constructed credentials")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid relay network admitted", err)
			}
		})
	}
}

func TestSandboxRelayNetworkRejectsTopologyRoutingRegistryAndAuthorityDrift(t *testing.T) {
	for _, mode := range []string{"vnet-prefix", "peering", "custom-dns", "embedded-subnet", "relay-outbound", "relay-security", "route-address", "security-destination", "firewall-address", "firewall-extra-group", "endpoint-service", "registry-ip", "endpoint-subnet", "registry-public", "connection-rejected", "embedded-connection-missing", "zone-address", "resolver-disabled", "pull-tenant", "grant-principal", "grant-condition", "role-authority"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			r, v := s.Placement.Services.RelayNetwork, s.Placement.Services
			p := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "vnet-prefix":
				p(r.virtualNetwork(s))["addressSpace"] = map[string]any{"addressPrefixes": []string{"10.61.0.0/16"}}
			case "peering":
				p(r.virtualNetwork(s))["virtualNetworkPeerings"] = []any{map[string]string{"id": "foreign"}}
			case "custom-dns":
				p(r.virtualNetwork(s))["dhcpOptions"] = map[string]any{"dnsServers": []string{"10.60.9.4"}}
			case "embedded-subnet":
				p(r.virtualNetwork(s))["subnets"] = []any{w.resources[v.Relay.Subnet]}
			case "relay-outbound":
				p(v.Relay.Subnet)["defaultOutboundAccess"] = true
			case "relay-security":
				p(v.Relay.Subnet)["networkSecurityGroup"] = map[string]string{"id": s.Placement.SecurityGroup}
			case "route-address":
				p(r.RouteTable)["routes"].([]any)[0].(map[string]any)["properties"].(map[string]any)["nextHopIpAddress"] = "10.60.4.5"
			case "security-destination":
				for _, raw := range p(r.SecurityGroup)["securityRules"].([]any) {
					rule := raw.(map[string]any)
					if rule["name"] == "relay-private-endpoints" {
						rule["properties"].(map[string]any)["destinationAddressPrefixes"] = []string{"*"}
					}
				}
			case "firewall-address":
				p(r.Topology.Firewall)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateIPAddress"] = "10.60.4.5"
			case "firewall-extra-group":
				w.pages[r.Topology.Policy+"/ruleCollectionGroups"]["value"] = append(w.pages[r.Topology.Policy+"/ruleCollectionGroups"]["value"].([]any), w.resources[r.Topology.Policy+"/ruleCollectionGroups/relay-platform"])
			case "endpoint-service":
				p(r.Registry.Endpoint)["privateLinkServiceConnections"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceId"] = "foreign"
			case "registry-ip":
				p(r.Registry.Interface)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateIPAddress"] = "10.60.3.9"
			case "endpoint-subnet":
				p(r.Registry.Subnet)["routeTable"] = map[string]string{"id": r.RouteTable}
			case "registry-public":
				p(s.Placement.Registry)["publicNetworkAccess"] = "Enabled"
			case "connection-rejected":
				p(r.Registry.Connection)["privateLinkServiceConnectionState"].(map[string]string)["status"] = "Rejected"
			case "embedded-connection-missing":
				p(s.Placement.Registry)["privateEndpointConnections"] = []any{}
				expected = hostedexecution.ErrUnavailable
			case "zone-address":
				p(r.Registry.Zone + "/A/studiotest")["aRecords"] = []any{map[string]string{"ipv4Address": "10.60.3.9"}}
			case "resolver-disabled":
				p(r.Resolver.Policy + "/dnsSecurityRules/relay-default-block")["dnsSecurityRuleState"] = "Disabled"
			case "pull-tenant":
				p(v.Relay.Pull.Resource)["tenantId"] = "foreign"
			case "grant-principal":
				p(r.Registry.PullAssignment)["principalId"] = s.Authority.Control.PrincipalID
			case "grant-condition":
				p(r.Registry.PullAssignment)["condition"] = "unexpected"
			case "role-authority":
				p("/subscriptions/" + s.Worker.Subscription + pullRole)["permissions"] = []any{map[string]any{"actions": []string{"*"}}}
			}
			reader, err := NewSDKRelayNetwork(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("relay drift admitted or misclassified", err)
			}
		})
	}
}

func TestSandboxRelayNetworkSharesEvidenceAndFreshFailureInvalidatesIt(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKRelayNetwork(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
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
	if len(w.calls) != 28 {
		t.Fatal("relay evidence scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("relay evidence was not shared")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable || strings.Contains(err.Error(), "canary") {
		t.Fatal("metadata gap became absence or leaked diagnostics", err)
	}
	w.status = 0
	subnet := s.Placement.Services.Relay.Subnet
	w.resources[subnet]["properties"].(map[string]any)["defaultOutboundAccess"] = true
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh relay check used old evidence", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed relay check retained safe evidence", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled check used evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed relay reader used evidence", err)
	}
}

func TestSandboxRuntimeRelayNetworkBlocksCreationImportAndLiveCommands(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings("discovery")
	w := newPlacementWorld(t, s)
	protection := &placementProtection{}
	runtime, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: protection, ImageAdmission: protection, ARMTransport: w, Credentials: func(string) (azcore.TokenCredential, error) {
		return &runtimeCredential{token: "placement-metadata-canary"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	w.resources[s.Placement.Services.Relay.Subnet]["properties"].(map[string]any)["defaultOutboundAccess"] = true
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("relay drift reached image import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("relay drift reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("relay drift reached live commands", err)
	}
	runtime.Close()
	if err := runtime.relayNetwork.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left relay network reader open", err)
	}
}
