package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func topologyCollection(w *placementWorld) map[string]any {
	id := w.s.Placement.Network.Topology.Policy + "/ruleCollectionGroups/" + sandboxFirewallGroup
	return w.resources[id]["properties"].(map[string]any)["ruleCollections"].([]any)[0].(map[string]any)
}

func TestSandboxTopologyDefinitionsHaveNoPlatformOrInternetAllow(t *testing.T) {
	s := runtimeSettings()
	defs, err := s.TopologyDefinitions()
	if err != nil || len(defs) != 5 {
		t.Fatal("missing firewall packaging inputs", err)
	}
	data, _ := json.Marshal(defs[s.Placement.Network.Topology.Policy+"/ruleCollectionGroups/"+sandboxFirewallGroup])
	var group map[string]any
	if json.Unmarshal(data, &group) != nil {
		t.Fatal("invalid firewall group definition")
	}
	collections := group["properties"].(map[string]any)["ruleCollections"].([]any)
	if len(collections) != 1 {
		t.Fatal("sandbox firewall contains another collection")
	}
	collection := collections[0].(map[string]any)
	rules := collection["rules"].([]any)
	if collection["action"].(map[string]any)["type"] != "Deny" || len(rules) != 1 {
		t.Fatal("sandbox firewall permits another traffic class")
	}
	rule := rules[0].(map[string]any)
	wanted := map[string]any{"name": "deny-egress", "ruleType": "NetworkRule", "sourceAddresses": []any{"*"}, "destinationAddresses": []any{"*"}, "destinationPorts": []any{"*"}, "ipProtocols": []any{"Any"}}
	if hash(rule) != hash(wanted) {
		t.Fatal("firewall does not deny all routed traffic")
	}
}

func TestSandboxTopologyRejectsNetworkAndFirewallBypasses(t *testing.T) {
	for _, mode := range []string{"peering", "custom-dns", "address-space", "extra-subnet", "embedded-subnet", "public-nat", "extended-location", "firewall-address", "firewall-policy", "firewall-nat", "firewall-subnet-route", "firewall-subnet-interface", "public-ip-attachment", "public-ip-tier", "parent-policy", "child-policy", "shared-firewall", "dns-proxy", "allow", "partial-deny", "extra-group", "nonempty-default", "missing-group-reference", "pending-firewall"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			p, settings := s.Placement, s.Placement.Network.Topology
			w := newPlacementWorld(t, s)
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			network := props(p.virtualNetwork())
			firewall := props(settings.Firewall)
			policy := props(settings.Policy)
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "peering":
				network["virtualNetworkPeerings"] = []any{map[string]any{"id": "other"}}
			case "custom-dns":
				network["dhcpOptions"] = map[string]any{"dnsServers": []string{"10.48.7.4"}}
			case "address-space":
				network["addressSpace"] = map[string]any{"addressPrefixes": []string{"10.48.0.0/16", "10.49.0.0/16"}}
			case "extra-subnet":
				network["subnets"] = append(network["subnets"].([]any), w.resources[p.Subnet])
			case "embedded-subnet":
				network["subnets"].([]any)[0] = map[string]any{"id": p.Subnet, "name": "sandbox", "type": "Microsoft.Network/virtualNetworks/subnets", "properties": map[string]any{"provisioningState": "Succeeded", "addressPrefix": "10.49.0.0/23"}}
			case "public-nat":
				network["defaultPublicNatGateway"] = map[string]any{"id": "other"}
			case "extended-location":
				w.resources[p.virtualNetwork()]["extendedLocation"] = map[string]string{"name": "other", "type": "EdgeZone"}
			case "firewall-address":
				firewall["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateIPAddress"] = "10.48.4.5"
			case "firewall-policy":
				firewall["firewallPolicy"] = map[string]any{"id": "other"}
			case "firewall-nat":
				firewall["natRuleCollections"] = []any{map[string]any{"name": "escape"}}
			case "firewall-subnet-route":
				props(p.firewallSubnet())["routeTable"] = map[string]any{"id": "other"}
			case "firewall-subnet-interface":
				props(p.firewallSubnet())["ipConfigurations"] = []any{map[string]any{"id": "other"}}
			case "public-ip-attachment":
				props(settings.PublicIP)["ipConfiguration"] = map[string]any{"id": "other"}
			case "public-ip-tier":
				w.resources[settings.PublicIP]["sku"].(map[string]any)["tier"] = "Global"
			case "parent-policy":
				policy["basePolicy"] = map[string]any{"id": "other"}
			case "child-policy":
				policy["childPolicies"] = []any{map[string]any{"id": "other"}}
			case "shared-firewall":
				policy["firewalls"] = []any{map[string]any{"id": settings.Firewall}, map[string]any{"id": "other"}}
			case "dns-proxy":
				policy["dnsSettings"] = map[string]any{"enableProxy": true}
			case "allow":
				topologyCollection(w)["action"] = map[string]any{"type": "Allow"}
			case "partial-deny":
				topologyCollection(w)["rules"].([]any)[0].(map[string]any)["destinationPorts"] = []string{"443"}
			case "extra-group", "nonempty-default":
				id := settings.Policy + "/ruleCollectionGroups/DefaultNetworkRuleCollectionGroup"
				value := map[string]any{"id": id, "name": "DefaultNetworkRuleCollectionGroup", "type": "Microsoft.Network/firewallPolicies/ruleCollectionGroups", "properties": map[string]any{"provisioningState": "Succeeded", "priority": 200, "ruleCollections": []any{topologyCollection(w)}}}
				if mode == "extra-group" {
					value["name"], value["id"] = "escape", settings.Policy+"/ruleCollectionGroups/escape"
				}
				path := settings.Policy + "/ruleCollectionGroups"
				w.pages[path]["value"] = append(w.pages[path]["value"].([]any), value)
				policy["ruleCollectionGroups"] = append(policy["ruleCollectionGroups"].([]any), map[string]any{"id": value["id"]})
			case "missing-group-reference":
				policy["ruleCollectionGroups"] = []any{}
				expected = hostedexecution.ErrUnavailable
			case "pending-firewall":
				firewall["provisioningState"] = "Updating"
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKTopology(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("unsafe or incomplete topology admitted", err)
			}
		})
	}
}

func TestSandboxTopologySettingsBindFirewallAndAddressSpaceBeforeCredentials(t *testing.T) {
	for _, mode := range []string{"missing", "public-vnet", "outside-vnet", "overlap", "small-firewall", "reserved-firewall", "wrong-address", "foreign-firewall", "wrong-policy-type", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			topology := &s.Placement.Network.Topology
			switch mode {
			case "missing":
				*topology = TopologySettings{}
			case "public-vnet":
				topology.Prefix = "198.51.100.0/24"
			case "outside-vnet":
				topology.Prefix = "10.49.0.0/16"
			case "overlap":
				topology.FirewallPrefix = s.Placement.Private.SubnetPrefix
			case "small-firewall":
				topology.FirewallPrefix = "10.48.4.0/27"
			case "reserved-firewall":
				s.Placement.Network.FirewallIP = "10.48.4.3"
			case "wrong-address":
				s.Placement.Network.FirewallIP = "10.48.5.4"
			case "foreign-firewall":
				topology.Firewall = strings.Replace(topology.Firewall, s.Worker.Subscription, "foreign", 1)
			case "wrong-policy-type":
				topology.Policy = topology.Firewall
			case "profile":
				topology.Firewall += "-other"
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			f := newControllerFixture(t, "go", "")
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid topology constructed credentials")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid topology settings accepted", err)
			}
		})
	}
}

func TestSandboxTopologyAllowsOnlyEmptyServiceDefaultsAcrossFourBoundedPages(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	policy := s.Placement.Network.Topology.Policy
	path := policy + "/ruleCollectionGroups"
	values := w.pages[path]["value"].([]any)
	refs := w.resources[policy]["properties"].(map[string]any)["ruleCollectionGroups"].([]any)
	for i, name := range []string{"DefaultDnatRuleCollectionGroup", "DefaultNetworkRuleCollectionGroup", "DefaultApplicationRuleCollectionGroup"} {
		id := path + "/" + name
		values = append(values, map[string]any{"id": id, "name": name, "type": "Microsoft.Network/firewallPolicies/ruleCollectionGroups", "properties": map[string]any{"provisioningState": "Succeeded", "priority": (i + 1) * 100, "ruleCollections": []any{}}})
		refs = append(refs, map[string]string{"id": id})
	}
	w.resources[policy]["properties"].(map[string]any)["ruleCollectionGroups"] = refs
	calls := 0
	transport := runtimeTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != path {
			return w.Do(r)
		}
		if r.Method != "GET" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2025-03-01" || r.Header.Get("Authorization") != "Bearer placement-metadata-canary" {
			t.Error("rule group inventory escaped its selected request")
		}
		if calls > 0 && r.URL.Query().Get("$skipToken") != strconv.Itoa(calls) || calls >= 4 {
			t.Error("rule group continuation changed or repeated")
			return response(503, `{}`), nil
		}
		page := map[string]any{"value": []any{values[calls]}}
		calls++
		if calls < 4 {
			page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-03-01&$skipToken=" + strconv.Itoa(calls)
		}
		return jsonResponse(page), nil
	})
	reader, err := NewSDKTopology(s, &runtimeCredential{token: "placement-metadata-canary"}, transport)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := reader.Check(t.Context(), s.Worker); err != nil || calls != 4 {
		t.Fatal("complete bounded default group inventory failed", err)
	}
}

func TestSandboxTopologyRefreshSharesEvidenceAndKeepsFailedPolicyVisible(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKTopology(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
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
		t.Fatal("topology scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("concurrent topology checks repeated reads")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("missing topology became worker absence", err)
	}
	w.status = 0
	topologyCollection(w)["action"] = map[string]string{"type": "Allow"}
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh check used cached safe firewall", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed current check retained successful evidence", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled check returned topology evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed reader returned topology evidence", err)
	}
}

func TestSandboxRuntimeTopologyCannotBeOmittedForAnyAdmission(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings()
	s.Mode = "discovery"
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
	topologyCollection(w)["action"] = map[string]string{"type": "Allow"}
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("changed firewall reached image import", err)
	}
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("changed firewall reached sandbox creation", err)
	}
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("changed firewall reached live command admission", err)
	}
	r.Close()
	if err := r.topology.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left topology client open", err)
	}
}
