package azuresandbox

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func privateRecord(w *placementWorld, zone string) map[string]any {
	return w.pages[zone+"/ALL"]["value"].([]any)[1].(map[string]any)["properties"].(map[string]any)
}

func TestSandboxPrivateRejectsEndpointInterfaceSubnetAndDNSDrift(t *testing.T) {
	for _, mode := range []string{"service", "group", "approval", "pending", "endpoint-interface", "interface-endpoint", "interface-ip", "interface-public", "interface-forwarding", "extended-location", "missing-data", "subnet-endpoint", "subnet-nat", "endpoint-allow", "dns-address", "dns-ttl", "dns-wildcard", "dns-alias", "dns-autoregister", "dns-link", "dns-link-registration", "dns-fallback", "dns-count"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			p := s.Placement.Private
			w := newPlacementWorld(t, s)
			prop := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			connection := prop(p.ExchangeEndpoint)["privateLinkServiceConnections"].([]any)[0].(map[string]any)["properties"].(map[string]any)
			nic := prop(p.RegistryInterface)
			ip := nic["ipConfigurations"].([]any)[0].(map[string]any)["properties"].(map[string]any)
			link := w.pages[p.ExchangeZone+"/virtualNetworkLinks"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "service":
				connection["privateLinkServiceId"] = s.Placement.Registry
			case "group":
				connection["groupIds"] = []string{"registry"}
			case "approval":
				connection["privateLinkServiceConnectionState"].(map[string]string)["status"] = "Rejected"
			case "pending":
				connection["privateLinkServiceConnectionState"].(map[string]string)["status"] = "Pending"
				expected = hostedexecution.ErrUnavailable
			case "endpoint-interface":
				prop(p.ExchangeEndpoint)["networkInterfaces"] = []any{map[string]string{"id": p.RegistryInterface}}
			case "interface-endpoint":
				nic["privateEndpoint"] = map[string]string{"id": p.ExchangeEndpoint}
			case "interface-ip":
				ip["privateIPAddress"] = p.ExchangeIP
			case "interface-public":
				ip["publicIPAddress"] = map[string]string{"id": "other"}
			case "interface-forwarding":
				nic["enableIPForwarding"] = true
			case "extended-location":
				w.resources[p.RegistryEndpoint]["extendedLocation"] = map[string]string{"name": "other", "type": "EdgeZone"}
			case "missing-data":
				nic["ipConfigurations"] = nic["ipConfigurations"].([]any)[:1]
				expected = hostedexecution.ErrUnavailable
			case "subnet-endpoint":
				prop(p.Subnet)["privateEndpoints"] = []any{map[string]string{"id": p.RegistryEndpoint}, map[string]string{"id": "other"}}
			case "subnet-nat":
				prop(p.Subnet)["natGateway"] = map[string]string{"id": "other"}
			case "endpoint-allow":
				prop(p.SecurityGroup)["securityRules"].([]any)[0].(map[string]any)["properties"].(map[string]any)["sourceAddressPrefix"] = "*"
			case "dns-address":
				privateRecord(w, p.ExchangeZone)["aRecords"] = []any{map[string]string{"ipv4Address": p.RegistryIP}}
			case "dns-ttl":
				privateRecord(w, p.ExchangeZone)["ttl"] = 3600
			case "dns-wildcard":
				record := w.pages[p.ExchangeZone+"/ALL"]["value"].([]any)[1].(map[string]any)
				record["name"], record["id"] = "*", p.ExchangeZone+"/A/*"
			case "dns-alias":
				privateRecord(w, p.ExchangeZone)["cnameRecord"] = map[string]string{"cname": "foreign.example"}
			case "dns-autoregister":
				privateRecord(w, p.ExchangeZone)["isAutoRegistered"] = true
			case "dns-link":
				link["virtualNetwork"] = map[string]string{"id": "other"}
			case "dns-link-registration":
				link["registrationEnabled"] = true
			case "dns-fallback":
				link["resolutionPolicy"] = "NxDomainRedirect"
			case "dns-count":
				prop(p.ExchangeZone)["numberOfRecordSets"] = 3
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKPrivate(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("invalid private binding admitted", err)
			}
		})
	}
}

func TestSandboxPrivateInventoryRejectsIncompleteOrEscapingPagesBeforeCredentials(t *testing.T) {
	for _, mode := range []string{"paged", "foreign-host", "foreign-path", "version", "filter", "duplicate-token", "duplicate-record", "missing-value", "null-value", "oversize", "truncated", "missing-metadata"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			path := s.Placement.Private.RegistryZone + "/ALL"
			w := newPlacementWorld(t, s)
			values := w.pages[path]["value"].([]any)
			credential := &runtimeCredential{token: "placement-metadata-canary"}
			calls := 0
			transport := runtimeTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "management.azure.com" || r.URL.Path != path || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer placement-metadata-canary" {
					t.Error("inventory request escaped its scope")
				}
				page := map[string]any{"value": []any{values[0]}, "nextLink": "https://management.azure.com" + path + "?api-version=2024-06-01&$skipToken=second"}
				if calls > 1 {
					page = map[string]any{"value": values[1:]}
				}
				switch mode {
				case "foreign-host":
					page["nextLink"] = "https://foreign.example" + path + "?api-version=2024-06-01&$skipToken=second"
				case "foreign-path":
					page["nextLink"] = "https://management.azure.com" + s.Placement.Private.ExchangeZone + "/ALL?api-version=2024-06-01&$skipToken=second"
				case "version":
					page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-01-01&$skipToken=second"
				case "filter":
					page["nextLink"] = "https://management.azure.com" + path + "?api-version=2024-06-01&$skipToken=second&$filter=other"
				case "duplicate-token":
					page["nextLink"] = "https://management.azure.com" + path + "?api-version=2024-06-01&$skipToken=second"
				case "duplicate-record":
					page = map[string]any{"value": []any{values[0], values[0]}}
				case "missing-value":
					page = map[string]any{}
				case "null-value":
					page = map[string]any{"value": nil}
				case "oversize":
					page = map[string]any{"value": append(values, values[0])}
				case "truncated":
					page = map[string]any{"value": []any{}, "nextLink": "https://management.azure.com" + path + "?api-version=2024-06-01&$skipToken=" + strings.Repeat("x", calls)}
				case "missing-metadata":
					return response(404, `{"error":{"message":"sensitive-provider-canary"}}`), nil
				}
				return jsonResponse(page), nil
			})
			reader, err := NewSDKPrivate(s, credential, transport)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			result, err := reader.inventory(t.Context(), path, 3)
			if mode == "paged" {
				if err != nil || len(result) != 3 || calls != 2 {
					t.Fatal("complete paged inventory failed", err)
				}
			} else if err == nil || result != nil || strings.Contains(err.Error(), "canary") {
				t.Fatal("incomplete inventory returned partial evidence", err)
			}
			if mode == "foreign-host" || mode == "foreign-path" || mode == "version" || mode == "filter" {
				if calls != 1 || len(credential.scopes) != 1 {
					t.Fatal("unsafe continuation acquired another credential")
				}
			}
		})
	}
}

func TestSandboxPrivateBindingsRejectInvalidSettingsBeforeCredentials(t *testing.T) {
	for _, mode := range []string{"missing", "wrong-ips", "reserved-address", "foreign-endpoint", "shared-interface", "shared-subnet", "overlap", "public-prefix", "broad-prefix", "zone-host", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			p := &s.Placement.Private
			switch mode {
			case "missing":
				*p = PrivateSettings{}
			case "wrong-ips":
				p.RegistryIP = "10.48.6.13"
			case "reserved-address":
				p.RegistryIP = "10.48.6.3"
				s.Placement.Network.PrivateEndpointIPs = []string{"10.48.6.11", "10.48.6.12", "10.48.6.3"}
			case "foreign-endpoint":
				p.ExchangeEndpoint = strings.Replace(p.ExchangeEndpoint, s.Worker.Subscription, "foreign", 1)
			case "shared-interface":
				p.ExchangeInterface = p.RegistryInterface
			case "shared-subnet":
				p.Subnet = s.Placement.Subnet
			case "overlap":
				p.SubnetPrefix = s.Placement.SubnetPrefix
			case "public-prefix":
				p.SubnetPrefix = "198.51.100.0/24"
			case "broad-prefix":
				p.SubnetPrefix = "10.0.0.0/7"
			case "zone-host":
				p.ExchangeZone = p.RegistryZone
			case "profile":
				p.ExchangeEndpoint += "-other"
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			f := newControllerFixture(t, "go", "")
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid private bindings constructed a credential")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid private configuration accepted", err)
			}
		})
	}
}

func TestSandboxPrivateRunningEvidenceIsBoundedAndFreshFailureInvalidatesIt(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKPrivate(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
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
	if len(w.calls) != 12 {
		t.Fatal("private scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("concurrent private observations repeated reads")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("missing private metadata became worker absence", err)
	}
	w.status = 0
	privateRecord(w, s.Placement.Private.ExchangeZone)["ttl"] = 3600
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh observation used old DNS evidence", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed full observation retained successful evidence", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled private check returned evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed private reader returned evidence", err)
	}
}

func TestSandboxRuntimePrivateGateBindsReviewedDNSAndBlocksAllAdmissions(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings()
	s.Mode = "discovery"
	defs, err := s.PrivateDNSDefinitions()
	if err != nil || len(defs) != 7 {
		t.Fatal("missing reviewable DNS definitions", err)
	}
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
	privateRecord(w, s.Placement.Private.ExchangeZone)["aRecords"] = []any{map[string]string{"ipv4Address": "10.48.6.13"}}
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("private DNS drift reached import", err)
	}
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("private DNS drift reached creation", err)
	}
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("private DNS drift reached live command admission", err)
	}
	r.Close()
	if err := r.private.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left private reader open", err)
	}
}
