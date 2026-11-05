package azuresandbox

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxRegistryServiceRequiresCompleteReportedInventory(t *testing.T) {
	s := runtimeSettings()
	s.Placement.RegistryConsumers = append([]string{s.Placement.Private.RegistryEndpoint + "-trusted"}, s.Placement.RegistryConsumers...)
	s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
	s.Host.Profile = s.HostProfile()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKRegistryService(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	s.Placement.RegistryConsumers[0] += "-changed"
	if err := reader.Check(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	if len(w.calls) != 4 {
		t.Fatal("registry observation scope changed", len(w.calls))
	}
	values := w.pages[s.Placement.Registry+"/privateEndpointConnections"]["value"].([]any)
	slices.Reverse(values)
	if err := reader.Check(t.Context(), s.Worker); err != nil {
		t.Fatal("inventory order changed authority", err)
	}
}

func TestSandboxRegistryConsumersBindProfilesAndRejectAliasesBeforeCredentials(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	for _, mode := range []string{"worker", "relay", "exchange", "backend", "duplicate", "foreign", "type", "limit", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			endpoint := s.Placement.Private.RegistryEndpoint + "-trusted"
			s.Placement.RegistryConsumers = []string{endpoint}
			switch mode {
			case "worker":
				s.Placement.RegistryConsumers = []string{s.Placement.Private.RegistryEndpoint}
			case "relay":
				s.Placement.RegistryConsumers = []string{s.Placement.Services.RelayNetwork.Registry.Endpoint}
			case "exchange":
				s.Placement.RegistryConsumers = []string{s.Placement.Private.ExchangeEndpoint}
			case "backend":
				s.Placement.RegistryConsumers = []string{s.Placement.Services.BackendEndpoint.Endpoint}
			case "duplicate":
				s.Placement.RegistryConsumers = append(s.Placement.RegistryConsumers, endpoint)
			case "foreign":
				s.Placement.RegistryConsumers[0] = strings.Replace(endpoint, s.Worker.Subscription, "foreign", 1)
			case "type":
				s.Placement.RegistryConsumers[0] = s.Placement.Private.RegistryInterface
			case "limit":
				s.Placement.RegistryConsumers = make([]string, 7)
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid registry consumer constructed credentials")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid consumer admitted", err)
			}
		})
	}
}

func TestSandboxRegistryServiceRejectsAccessConsumerMemberAndReplicaDrift(t *testing.T) {
	for _, mode := range []string{"tier", "task-bypass", "export", "arm-auth", "public-rule", "data-host", "foreign-consumer", "extra-consumer", "missing-consumer", "embedded-missing", "snapshot-mismatch", "pending", "member", "zone", "replica", "missing-policy"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			path := s.Placement.Registry
			props := w.resources[path]["properties"].(map[string]any)
			connections := w.pages[path+"/privateEndpointConnections"]["value"].([]any)
			connection := connections[0].(map[string]any)["properties"].(map[string]any)
			member := w.pages[path+"/privateLinkResources"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "tier":
				w.resources[path]["sku"].(map[string]string)["tier"] = "Standard"
			case "task-bypass":
				props["networkRuleBypassAllowedForTasks"] = true
			case "export":
				props["policies"].(map[string]any)["exportPolicy"].(map[string]string)["status"] = "enabled"
			case "arm-auth":
				props["policies"].(map[string]any)["azureADAuthenticationAsArmPolicy"].(map[string]string)["status"] = "disabled"
			case "public-rule":
				props["networkRuleSet"].(map[string]any)["ipRules"] = []any{map[string]string{"action": "Allow", "iPAddressOrRange": "0.0.0.0/0"}}
			case "data-host":
				props["dataEndpointHostNames"] = []string{"foreign.azurecr.io"}
			case "foreign-consumer":
				connection["privateEndpoint"] = map[string]string{"id": s.Placement.Private.ExchangeEndpoint}
			case "extra-consumer":
				w.pages[path+"/privateEndpointConnections"]["value"] = append(connections, connections[0])
			case "missing-consumer":
				w.pages[path+"/privateEndpointConnections"]["value"] = connections[:1]
				expected = hostedexecution.ErrUnavailable
			case "embedded-missing":
				props["privateEndpointConnections"] = []any{}
				expected = hostedexecution.ErrUnavailable
			case "snapshot-mismatch":
				embedded := objectMap(connections[0])
				embedded["id"] = path + "/privateEndpointConnections/other"
				embedded["name"] = "other"
				props["privateEndpointConnections"] = []any{embedded, connections[1]}
				expected = hostedexecution.ErrUnavailable
			case "pending":
				connection["privateLinkServiceConnectionState"] = map[string]string{"status": "Pending", "actionsRequired": "None"}
				expected = hostedexecution.ErrUnavailable
			case "member":
				member["requiredMembers"] = []string{"registry", "registry_data_eastus2"}
			case "zone":
				member["requiredZoneNames"] = []string{"foreign.example"}
			case "replica":
				w.pages[path+"/replications"]["value"] = []any{map[string]any{"id": path + "/replications/eastus2", "name": "eastus2", "type": "Microsoft.ContainerRegistry/registries/replications", "location": "eastus2", "properties": map[string]any{"provisioningState": "Succeeded", "regionEndpointEnabled": true}}}
			case "missing-policy":
				delete(props, "policies")
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKRegistryService(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("registry drift admitted or misclassified", err)
			}
		})
	}
}

func TestSandboxRegistryInventoryRequiresCompleteBoundedScopedPages(t *testing.T) {
	for _, mode := range []string{"paged", "foreign-host", "foreign-path", "duplicate", "endless", "missing", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			path := s.Placement.Registry + "/privateEndpointConnections"
			values := w.pages[path]["value"].([]any)
			calls := 0
			transport := runtimeTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != path {
					return w.Do(r)
				}
				calls++
				page := map[string]any{"value": values[:1], "nextLink": "https://management.azure.com" + path + "?api-version=2025-11-01&$skipToken=second"}
				if calls > 1 {
					page = map[string]any{"value": values[1:]}
				}
				switch mode {
				case "foreign-host":
					page["nextLink"] = "https://foreign.example" + path + "?api-version=2025-11-01&$skipToken=second"
				case "foreign-path":
					page["nextLink"] = "https://management.azure.com" + s.Placement.Registry + "/replications?api-version=2025-11-01&$skipToken=second"
				case "duplicate":
					page = map[string]any{"value": []any{values[0], values[0]}}
				case "endless":
					page = map[string]any{"value": []any{}, "nextLink": "https://management.azure.com" + path + "?api-version=2025-11-01&$skipToken=" + strings.Repeat("x", calls)}
				case "missing":
					page = map[string]any{}
				case "unavailable":
					return response(404, `{"error":{"message":"private-canary"}}`), nil
				}
				return jsonResponse(page), nil
			})
			reader, err := NewSDKRegistryService(s, &runtimeCredential{token: "placement-metadata-canary"}, transport)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			err = reader.Check(t.Context(), s.Worker)
			if mode == "paged" {
				if err != nil || calls != 2 {
					t.Fatal("complete inventory failed", err, calls)
				}
				return
			}
			if err == nil || strings.Contains(err.Error(), "canary") || calls > 3 {
				t.Fatal("unsafe or incomplete inventory admitted", err, calls)
			}
			if (mode == "foreign-host" || mode == "foreign-path") && calls != 1 {
				t.Fatal("escaped continuation reached transport")
			}
		})
	}
}

func TestSandboxRegistryServiceSharesEvidenceAndInvalidatesFreshFailure(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKRegistryService(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
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
	if len(w.calls) != 4 {
		t.Fatal("registry evidence scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("registry evidence was not shared")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("registry gap became absence", err)
	}
	w.status = 0
	w.resources[s.Placement.Registry]["properties"].(map[string]any)["networkRuleBypassAllowedForTasks"] = true
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh registry check used old evidence", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed registry check retained evidence", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled registry check returned evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed registry check returned evidence", err)
	}
}

func TestSandboxRuntimeRegistryServiceBlocksCreationImportAndLiveCommands(t *testing.T) {
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
	w.resources[s.Placement.Registry]["properties"].(map[string]any)["networkRuleBypassAllowedForTasks"] = true
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("registry drift reached import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("registry drift reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("registry drift reached live commands", err)
	}
	runtime.Close()
	if err := runtime.registryService.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left registry reader open", err)
	}
}
