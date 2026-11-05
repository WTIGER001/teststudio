package azuresandbox

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxHostNetworkBindsTopologyAndModeSpecificDependencyPolicy(t *testing.T) {
	execution, discovery := runtimeSettings(), runtimeSettings("discovery")
	if hash(execution.Worker) != hash(discovery.Worker) {
		t.Fatal("trusted host network fragmented image pool")
	}
	for _, s := range []Settings{execution, discovery} {
		defs, err := s.HostNetworkDefinitions()
		if err != nil || len(defs) != 14 {
			t.Fatal("missing staged network definitions", err, len(defs))
		}
		if !slices.Contains(s.Host.resolverBinding(s).Domains, s.Host.DatabaseHost) {
			t.Fatal("selected database host cannot resolve")
		}
		domains := s.Host.dependencyDomains(s)
		if slices.Contains(domains, "studio.teststudio.example") || !slices.Contains(domains, "management.westus2.azuredevcompute.io") || !slices.Contains(domains, "studiotest.vault.azure.net") || slices.Contains(domains, "studiotest.blob.core.windows.net") != (s.Mode == "execution") {
			t.Fatal("host dependency boundary drifted", domains)
		}
		worker, _ := json.Marshal(s.Placement.networkRules())
		if strings.Contains(string(worker), "Internet") || strings.Contains(string(worker), "MicrosoftContainerRegistry") {
			t.Fatal("trusted network access entered sandbox policy")
		}
		w := newPlacementWorld(t, s)
		reader, err := NewSDKHostNetwork(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		if err := reader.Check(t.Context(), s.Worker); err != nil || len(w.calls) != 17 {
			t.Fatal("fixed controller network rejected", err, len(w.calls))
		}
		if err := reader.CheckRunning(t.Context(), s.Worker); err != nil || w.calls[s.Host.Network.SecurityGroup] != 1 {
			t.Fatal("live observation repeated successful evidence", err)
		}
		request, _ := http.NewRequest("POST", "https://management.azure.com"+s.Host.virtualNetwork()+"/listDnsResolverPolicies?api-version=2025-05-01", strings.NewReader(`{"mutate":true}`))
		if _, err := reader.http.Do(request); err != hostedexecution.ErrOwnership {
			t.Fatal("read action accepted a body", err)
		}
	}
	for _, mode := range []string{"missing", "overlap", "firewall-overlap", "reserved", "alias", "resolver-alias", "stale"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			switch mode {
			case "missing":
				s.Host.Network = HostNetworkSettings{}
			case "overlap":
				s.Host.Network.Topology.Prefix = s.Placement.Network.Topology.Prefix
			case "firewall-overlap":
				s.Host.Network.Topology.FirewallPrefix = s.Host.SubnetPrefix
			case "reserved":
				s.Host.Network.FirewallIP = "10.72.4.2"
			case "alias":
				s.Host.Network.SecurityGroup = s.Host.Registry.SecurityGroup
			case "resolver-alias":
				s.Host.Network.Resolver = s.Placement.Network.Resolver
			case "stale":
				s.Host.Network.Topology.Firewall += "-other"
			}
			if mode != "stale" {
				s.Host.Profile = s.HostProfile()
			}
			if _, err := NewSDKHostNetwork(s, &runtimeCredential{}, nil); err != hostedexecution.ErrConfiguration {
				t.Fatal("unsafe settings admitted", err)
			}
		})
	}
}

func TestSandboxHostNetworkRejectsTopologyRouteFirewallAndResolverDrift(t *testing.T) {
	for _, mode := range []string{"peering", "dns-server", "extra-subnet", "embedded", "nat", "source-group", "route", "extra-rule", "source-access", "database-access", "firewall-ip", "firewall-extra-interface", "firewall-policy", "firewall-domains", "extra-group", "resolver-domains", "resolver-association", "resolver-page", "pending"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			h, r := s.Host, s.Host.Network
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			group := props(r.Topology.Policy + "/ruleCollectionGroups/controller-platform")
			switch mode {
			case "peering":
				props(h.virtualNetwork())["virtualNetworkPeerings"] = []any{map[string]string{"id": "foreign"}}
			case "dns-server":
				props(h.virtualNetwork())["dhcpOptions"] = map[string]any{"dnsServers": []string{"8.8.8.8"}}
			case "extra-subnet":
				props(h.virtualNetwork())["subnets"] = append(props(h.virtualNetwork())["subnets"].([]any), w.resources[h.Subnet])
			case "embedded":
				clone := objectMap(w.resources[h.Subnet])
				clone["properties"].(map[string]any)["addressPrefix"] = "10.73.0.0/23"
				props(h.virtualNetwork())["subnets"].([]any)[0] = clone
			case "nat":
				props(h.Subnet)["natGateway"] = map[string]string{"id": "foreign"}
			case "source-group":
				props(h.Subnet)["networkSecurityGroup"] = map[string]string{"id": h.Registry.SecurityGroup}
			case "route":
				props(r.RouteTable)["routes"].([]any)[0].(map[string]any)["properties"].(map[string]any)["nextHopIpAddress"] = "10.72.4.5"
			case "extra-rule":
				props(r.SecurityGroup)["securityRules"] = append(props(r.SecurityGroup)["securityRules"].([]any), props(r.SecurityGroup)["securityRules"].([]any)[0])
			case "source-access":
				props(r.SecurityGroup)["securityRules"].([]any)[0].(map[string]any)["properties"].(map[string]any)["sourceAddressPrefix"] = "*"
			case "database-access":
				for _, raw := range props(r.SecurityGroup)["securityRules"].([]any) {
					rule := raw.(map[string]any)
					if rule["name"] == "controller-database" {
						rule["properties"].(map[string]any)["destinationAddressPrefix"] = "10.72.6.9"
					}
				}
			case "firewall-ip":
				props(r.Topology.Firewall)["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateIPAddress"] = "10.72.4.5"
			case "firewall-extra-interface":
				props(r.Topology.Firewall)["ipConfigurations"] = append(props(r.Topology.Firewall)["ipConfigurations"].([]any), props(r.Topology.Firewall)["ipConfigurations"].([]any)[0])
			case "firewall-policy":
				props(r.Topology.Policy)["basePolicy"] = map[string]string{"id": "foreign"}
			case "firewall-domains":
				group["ruleCollections"].([]any)[0].(map[string]any)["rules"].([]any)[0].(map[string]any)["targetFqdns"] = []string{"*"}
			case "extra-group":
				path := r.Topology.Policy + "/ruleCollectionGroups"
				w.pages[path]["value"] = append(w.pages[path]["value"].([]any), w.pages[path]["value"].([]any)[0])
			case "resolver-domains":
				props(r.Resolver.AllowList)["domains"] = append(props(r.Resolver.AllowList)["domains"].([]any), "foreign.example")
			case "resolver-association":
				w.pages[h.virtualNetwork()+"/listDnsResolverPolicies"]["value"] = []any{map[string]string{"id": s.Placement.Network.Resolver.Policy}}
			case "resolver-page":
				w.pages[r.Resolver.Policy+"/dnsSecurityRules"]["nextLink"] = "https://foreign.example/private-canary"
			case "pending":
				props(r.Topology.Firewall)["provisioningState"] = "Updating"
			}
			reader, err := NewSDKHostNetwork(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			err = reader.Check(t.Context(), s.Worker)
			if err != hostedexecution.ErrOwnership && err != hostedexecution.ErrUnavailable {
				t.Fatal("unsafe network admitted", err)
			}
			if strings.Contains(err.Error(), "canary") {
				t.Fatal("provider diagnostics leaked")
			}
		})
	}
}

func TestSandboxRuntimeHostNetworkFreshFailureBlocksEveryAdmission(t *testing.T) {
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
	w.resources[s.Host.Network.RouteTable]["properties"].(map[string]any)["disableBgpRoutePropagation"] = false
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || p.imports.Load() != 0 {
		t.Fatal("routing drift reached import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || p.creates.Load() != 1 {
		t.Fatal("routing drift reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || p.verifies.Load() != 0 {
		t.Fatal("fresh failure retained live evidence", err)
	}
	runtime.Close()
	if err := runtime.hostNetwork.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("host network reader remained open", err)
	}
}
