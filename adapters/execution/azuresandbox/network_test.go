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

func networkEntries(w *placementWorld, id, field string) []any {
	return w.resources[id]["properties"].(map[string]any)[field].([]any)
}

func TestSandboxNetworkPolicyRestrictsPrivateConnectionWithoutJobPlatformRules(t *testing.T) {
	s := runtimeSettings()
	defs, err := s.NetworkDefinitions()
	if err != nil || len(defs) != 2 {
		t.Fatal("missing network definitions", err)
	}
	props := defs[s.Placement.SecurityGroup].(map[string]any)["properties"].(map[string]any)
	rules := props["securityRules"].([]any)
	if len(rules) != 6 || props["flushConnection"] != true {
		t.Fatal("policy does not replace default traffic decisions")
	}
	allows := 0
	for _, raw := range rules {
		rule := raw.(map[string]any)
		p := rule["properties"].(map[string]any)
		if p["access"] != "Allow" {
			continue
		}
		allows++
		if p["direction"] != "Outbound" || p["sourceAddressPrefix"] != "10.48.0.0/23" || p["sourcePortRange"] != "*" {
			t.Fatal("private connection has inbound or unscoped allow")
		}
		switch rule["name"] {
		case "sandbox-dns-udp", "sandbox-dns-tcp":
			if p["destinationAddressPrefix"] != "AzurePlatformDNS" || p["destinationPortRange"] != "53" {
				t.Fatal("DNS is not constrained to the selected platform resolver")
			}
		case "sandbox-private-https":
			if p["protocol"] != "Tcp" || p["destinationPortRange"] != "443" || hash(p["destinationAddressPrefixes"]) != hash([]string{"10.48.6.10", "10.48.6.11", "10.48.6.12"}) {
				t.Fatal("private HTTPS destinations widened")
			}
		default:
			t.Fatal("ordinary Job platform access entered sandbox connection policy")
		}
	}
	if allows != 3 {
		t.Fatal("policy lost required constrained access")
	}
	routes := defs[s.Placement.RouteTable].(map[string]any)["properties"].(map[string]any)
	if routes["disableBgpRoutePropagation"] != true || len(routes["routes"].([]any)) != 1 {
		t.Fatal("routing can inherit a gateway or bypass")
	}
	route := routes["routes"].([]any)[0].(map[string]any)["properties"].(map[string]any)
	if route["addressPrefix"] != "0.0.0.0/0" || route["nextHopType"] != "VirtualAppliance" || route["nextHopIpAddress"] != "10.48.4.4" {
		t.Fatal("default route escapes the selected firewall")
	}
}

func TestSandboxNetworkRejectsChangedRulesRoutesAndAttachments(t *testing.T) {
	for _, mode := range []string{"rule-port", "rule-source", "rule-destination", "rule-priority", "missing-deny", "extra-allow", "duplicate-rule", "duplicate-address", "unknown-rule-field", "foreign-rule", "flush", "shared-subnet", "nic", "unknown-group-field", "route-internet", "route-firewall", "route-bgp", "route-override", "extra-route", "foreign-route", "unknown-route-field", "region", "pending-rule", "pending-group"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			group := w.resources[s.Placement.SecurityGroup]["properties"].(map[string]any)
			table := w.resources[s.Placement.RouteTable]["properties"].(map[string]any)
			rules := networkEntries(w, s.Placement.SecurityGroup, "securityRules")
			private := rules[4].(map[string]any)["properties"].(map[string]any)
			routes := networkEntries(w, s.Placement.RouteTable, "routes")
			route := routes[0].(map[string]any)["properties"].(map[string]any)
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "rule-port":
				private["destinationPortRange"] = "*"
			case "rule-source":
				private["sourceAddressPrefix"] = "*"
			case "rule-destination":
				private["destinationAddressPrefixes"] = []string{"10.48.6.12"}
			case "rule-priority":
				private["priority"] = 99
			case "missing-deny":
				group["securityRules"] = rules[:len(rules)-1]
			case "extra-allow":
				group["securityRules"] = append(rules, networkRule("escape", "Outbound", "Allow", "*", "*", "*", "*", 101))
			case "duplicate-rule":
				rules[0] = rules[1]
			case "duplicate-address":
				private["destinationAddressPrefixes"] = []string{"10.48.6.10", "10.48.6.10"}
			case "unknown-rule-field":
				private["destinationApplicationSecurityGroups"] = []any{map[string]string{"id": "other"}}
			case "foreign-rule":
				rules[0].(map[string]any)["id"] = s.Placement.SecurityGroup + "/securityRules/other"
			case "flush":
				group["flushConnection"] = false
			case "shared-subnet":
				group["subnets"] = []any{map[string]string{"id": s.Placement.Subnet}, map[string]string{"id": "other"}}
			case "nic":
				group["networkInterfaces"] = []any{map[string]string{"id": "other"}}
			case "unknown-group-field":
				group["futurePolicy"] = "escape"
			case "route-internet":
				route["nextHopType"] = "Internet"
			case "route-firewall":
				route["nextHopIpAddress"] = "10.48.4.5"
			case "route-bgp":
				table["disableBgpRoutePropagation"] = false
			case "route-override":
				route["hasBgpOverride"] = true
			case "extra-route":
				table["routes"] = append(routes, routes[0])
			case "foreign-route":
				routes[0].(map[string]any)["type"] = "Microsoft.Network/other"
			case "unknown-route-field":
				route["futureOverride"] = true
			case "region":
				w.resources[s.Placement.RouteTable]["location"] = "elsewhere"
			case "pending-rule":
				private["provisioningState"] = "Updating"
				expected = hostedexecution.ErrUnavailable
			case "pending-group":
				group["provisioningState"] = "Updating"
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKNetwork(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("unsafe or incomplete connection admitted", err)
			}
		})
	}
}

func TestSandboxNetworkFreshAdmissionInvalidatesBoundedRunningEvidence(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	private := networkEntries(w, s.Placement.SecurityGroup, "securityRules")[4].(map[string]any)["properties"].(map[string]any)
	private["destinationAddressPrefixes"] = []string{"10.48.6.12", "10.48.6.11", "10.48.6.10"}
	reader, err := NewSDKNetwork(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	now := time.Now()
	reader.now = func() time.Time { return now }
	s.Placement.Network.PrivateEndpointIPs[0] = "10.48.6.12"
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			if err := reader.CheckRunning(t.Context(), s.Worker); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("concurrent polling repeated network reads")
		}
	}
	private["destinationPortRange"] = "*"
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh dispatch used cached network evidence", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed fresh observation retained earlier success", err)
	}
	private["destinationPortRange"] = "443"
	if err := reader.CheckRunning(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable || strings.Contains(err.Error(), "canary") {
		t.Fatal("missing policy became worker absence or exposed diagnostics", err)
	}
	w.status = 0
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled read returned evidence", err)
	}
	if _, err := reader.get(t.Context(), s.Placement.Subnet); err != hostedexecution.ErrOwnership {
		t.Fatal("network reader read unselected resource", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed network reader returned evidence", err)
	}
}

func TestSandboxNetworkSettingsBindPolicyBeforeCredentials(t *testing.T) {
	for _, mode := range []string{"missing", "public-firewall", "firewall-in-subnet", "empty-endpoints", "public-endpoint", "duplicate", "firewall-reuse", "unsorted", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			switch mode {
			case "missing":
				s.Placement.Network = NetworkSettings{}
			case "public-firewall":
				s.Placement.Network.FirewallIP = "198.51.100.4"
			case "firewall-in-subnet":
				s.Placement.Network.FirewallIP = "10.48.0.4"
			case "empty-endpoints":
				s.Placement.Network.PrivateEndpointIPs = nil
			case "public-endpoint":
				s.Placement.Network.PrivateEndpointIPs = []string{"198.51.100.4"}
			case "duplicate":
				s.Placement.Network.PrivateEndpointIPs = []string{"10.48.6.10", "10.48.6.10"}
			case "firewall-reuse":
				s.Placement.Network.PrivateEndpointIPs = []string{s.Placement.Network.FirewallIP}
			case "unsorted":
				s.Placement.Network.PrivateEndpointIPs = []string{"10.48.6.11", "10.48.6.10"}
			case "profile":
				s.Placement.Network.FirewallIP = "10.48.4.5"
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			f := newControllerFixture(t, "go", "")
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid network policy constructed a credential")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid or stale network settings accepted", err)
			}
		})
	}
}

func TestSandboxRuntimeNetworkGateAppliesToCreationImportsAndLiveCommands(t *testing.T) {
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
	private := networkEntries(w, s.Placement.SecurityGroup, "securityRules")[4].(map[string]any)["properties"].(map[string]any)
	private["destinationPortRange"] = "*"
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("network drift reached creation protection", err)
	}
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("network drift reached image import", err)
	}
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("network drift reached live command protection", err)
	}
	r.Close()
	if err := r.network.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left network client open", err)
	}
}
