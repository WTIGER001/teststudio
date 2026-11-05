package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func resolverRule(w *placementWorld, name string) map[string]any {
	return w.resources[w.s.Placement.Network.Resolver.Policy+"/dnsSecurityRules/sandbox-"+name]["properties"].(map[string]any)
}

func TestSandboxResolverDefinitionsAndSettingsBindFiveDomainsBeforeCredentials(t *testing.T) {
	s := runtimeSettings()
	defs, err := s.ResolverDefinitions()
	if err != nil || len(defs) != 6 {
		t.Fatal("missing resolver packaging inputs", err)
	}
	r := s.Placement.Network.Resolver
	data, _ := json.Marshal(defs[r.AllowList])
	var list struct {
		Properties struct{ Domains []string } `json:"properties"`
	}
	if json.Unmarshal(data, &list) != nil || hash(list.Properties.Domains) != hash([]string{"exchange.studio.example", "studiotest.azurecr.io", "studiotest.privatelink.azurecr.io", "studiotest.westus2.data.azurecr.io", "studiotest.westus2.data.privatelink.azurecr.io"}) {
		t.Fatal("resolver inherited Jobs platform or unrelated domains")
	}
	for _, mode := range []string{"missing", "same-list", "foreign-policy", "wrong-list-type", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			r := &s.Placement.Network.Resolver
			switch mode {
			case "missing":
				*r = ResolverSettings{}
			case "same-list":
				r.BlockList = r.AllowList
			case "foreign-policy":
				r.Policy = strings.Replace(r.Policy, s.Worker.Subscription, "foreign", 1)
			case "wrong-list-type":
				r.AllowList = r.Policy
			case "profile":
				r.Policy += "-other"
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			f := newControllerFixture(t, "go", "")
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid resolver bindings constructed credentials")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid resolver bindings accepted", err)
			}
		})
	}
}

func TestSandboxResolverRejectsDomainRuleAndAssociationDrift(t *testing.T) {
	for _, mode := range []string{"extra-domain", "duplicate-domain", "remote-domain-list", "block-all", "rule-alert", "rule-disabled", "rule-priority", "extra-rule", "wrong-vnet", "extra-link", "wrong-policy", "missing-attachment", "partial-attachment", "pending-policy"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			r := s.Placement.Network.Resolver
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "extra-domain":
				props(r.AllowList)["domains"] = append(props(r.AllowList)["domains"].([]any), "foreign.example")
			case "duplicate-domain":
				props(r.AllowList)["domains"] = append(props(r.AllowList)["domains"].([]any), "exchange.studio.example")
			case "remote-domain-list":
				props(r.AllowList)["domainsUrl"] = "https://foreign.example/domains"
			case "block-all":
				props(r.BlockList)["domains"] = []string{"foreign.example"}
			case "rule-alert":
				resolverRule(w, "default-block")["action"] = map[string]string{"actionType": "Alert"}
			case "rule-disabled":
				resolverRule(w, "default-block")["dnsSecurityRuleState"] = "Disabled"
			case "rule-priority":
				resolverRule(w, "default-block")["priority"] = 99
			case "extra-rule":
				path := r.Policy + "/dnsSecurityRules"
				w.pages[path]["value"] = append(w.pages[path]["value"].([]any), w.pages[path]["value"].([]any)[0])
			case "wrong-vnet":
				props(r.Policy + "/virtualNetworkLinks/sandbox-network")["virtualNetwork"] = map[string]string{"id": s.Placement.virtualNetwork() + "-other"}
			case "extra-link":
				path := r.Policy + "/virtualNetworkLinks"
				w.pages[path]["value"] = append(w.pages[path]["value"].([]any), w.pages[path]["value"].([]any)[0])
			case "wrong-policy":
				w.pages[s.Placement.virtualNetwork()+"/listDnsResolverPolicies"]["value"] = []any{map[string]string{"id": r.Policy + "-other"}}
			case "missing-attachment":
				w.pages[s.Placement.virtualNetwork()+"/listDnsResolverPolicies"]["value"] = nil
				expected = hostedexecution.ErrUnavailable
			case "partial-attachment":
				w.pages[s.Placement.virtualNetwork()+"/listDnsResolverPolicies"]["nextLink"] = "https://foreign.example"
				expected = hostedexecution.ErrUnavailable
			case "pending-policy":
				props(r.Policy)["provisioningState"] = "Updating"
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKResolver(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("unsafe resolver policy admitted", err)
			}
		})
	}
}

func TestSandboxResolverRejectsAdditionalReportedZoneLinksWithoutBlockingUnrelatedNetworks(t *testing.T) {
	for _, mode := range []string{"unrelated", "unexpected-link", "missing-zone", "foreign-zone", "zone-count-gap", "missing-page"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			root := "/subscriptions/" + s.Worker.Subscription + "/providers/Microsoft.Network/privateDnsZones"
			zoneID := strings.TrimSuffix(s.Placement.Private.RegistryZone, "privatelink.azurecr.io") + "unrelated.example"
			linkID := zoneID + "/virtualNetworkLinks/unrelated-network"
			zone := map[string]any{"id": zoneID, "name": "unrelated.example", "type": "Microsoft.Network/privateDnsZones", "location": "global", "properties": map[string]any{"provisioningState": "Succeeded", "numberOfVirtualNetworkLinks": 1, "numberOfVirtualNetworkLinksWithRegistration": 0}}
			link := map[string]any{"id": linkID, "name": "unrelated-network", "type": "Microsoft.Network/privateDnsZones/virtualNetworkLinks", "location": "global", "properties": map[string]any{"provisioningState": "Succeeded", "virtualNetworkLinkState": "Completed", "registrationEnabled": false, "virtualNetwork": map[string]string{"id": s.Placement.virtualNetwork() + "-other"}}}
			w.resources[zoneID] = zone
			w.pages[zoneID+"/virtualNetworkLinks"] = map[string]any{"value": []any{link}}
			w.pages[root]["value"] = append(w.pages[root]["value"].([]any), zone)
			var expected error
			switch mode {
			case "unexpected-link":
				link["properties"].(map[string]any)["virtualNetwork"] = map[string]string{"id": s.Placement.virtualNetwork()}
				expected = hostedexecution.ErrOwnership
			case "missing-zone":
				w.pages[root]["value"] = []any{w.resources[s.Placement.Private.RegistryZone]}
				expected = hostedexecution.ErrUnavailable
			case "foreign-zone":
				zone["id"] = strings.Replace(zoneID, s.Worker.Subscription, "87654321-4321-4321-4321-cba987654321", 1)
				expected = hostedexecution.ErrOwnership
			case "zone-count-gap":
				zone["properties"].(map[string]any)["numberOfVirtualNetworkLinks"] = 2
				expected = hostedexecution.ErrUnavailable
			case "missing-page":
				w.pages[root]["value"] = nil
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKResolver(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("subscription link coverage was misclassified", err)
			}
		})
	}
}

func TestSandboxResolverOnlyPermitsSelectedEmptyReadAction(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	credential := &runtimeCredential{token: "placement-metadata-canary"}
	reader, err := NewSDKResolver(s, credential, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	path := s.Placement.virtualNetwork() + "/listDnsResolverPolicies"
	for _, mode := range []string{"get-action", "write", "payload", "length", "foreign-action", "foreign-host", "filtered"} {
		request, _ := http.NewRequest("POST", "https://management.azure.com"+path+"?api-version=2025-05-01", nil)
		switch mode {
		case "get-action":
			request.Method = "GET"
		case "write":
			request.Method = "PUT"
		case "payload":
			request, _ = http.NewRequest("POST", request.URL.String(), strings.NewReader(`{"write":true}`))
		case "length":
			request.ContentLength = 1
		case "foreign-action":
			request.URL.Path = s.Placement.virtualNetwork() + "/otherAction"
		case "foreign-host":
			request.URL.Host = "foreign.example"
		case "filtered":
			request.URL.RawQuery += "&$filter=other"
		}
		if _, err := reader.http.Do(request); err != hostedexecution.ErrOwnership || len(credential.scopes) != 0 {
			t.Fatal("read action admitted a mutation or foreign scope", mode, err)
		}
	}
	var attached any
	if err := reader.documentID(t.Context(), path, &attached); err != nil || len(w.calls) != 1 {
		t.Fatal("selected empty read action failed", err)
	}
}

func TestSandboxResolverSharesBoundedEvidenceAndInvalidatesFailedFreshPolicy(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKResolver(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
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
	if len(w.calls) != 9 {
		t.Fatal("resolver observation scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("concurrent resolver checks repeated reads")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable || strings.Contains(err.Error(), "canary") {
		t.Fatal("missing resolver evidence became worker absence or exposed diagnostics", err)
	}
	w.status = 0
	resolverRule(w, "default-block")["dnsSecurityRuleState"] = "Disabled"
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh policy used cached safe evidence", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed policy retained earlier success", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled resolver check returned evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed resolver check returned evidence", err)
	}
}

func TestSandboxRuntimeResolverCannotBeOmittedFromAnyAdmission(t *testing.T) {
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
	resolverRule(w, "default-block")["action"] = map[string]string{"actionType": "Alert"}
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("changed resolver policy reached image import", err)
	}
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("changed resolver policy reached creation", err)
	}
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("changed resolver policy reached live command admission", err)
	}
	r.Close()
	if err := r.resolver.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left resolver client open", err)
	}
}
