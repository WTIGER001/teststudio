package azuresandbox

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func unrelatedServiceWorkload(s Settings, kind string) map[string]any {
	base := "/subscriptions/" + s.Worker.Subscription + "/resourceGroups/studio-other/providers/Microsoft.App/"
	return map[string]any{"id": base + kind + "/unrelated", "name": "unrelated", "type": "Microsoft.App/" + kind, "identity": map[string]string{"type": "None"}, "properties": map[string]string{"environmentId": base + "managedEnvironments/unrelated"}}
}

func addServiceWorkload(w *placementWorld, kind string, item map[string]any) {
	path := serviceWorkloadPath(w.s, kind)
	w.pages[path]["value"] = append(w.pages[path]["value"].([]any), item)
}

func TestSandboxServiceWorkloadsBindReportedHostsAndFederationAbsence(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	addServiceWorkload(w, "containerApps", unrelatedServiceWorkload(s, "containerApps"))
	addServiceWorkload(w, "jobs", unrelatedServiceWorkload(s, "jobs"))
	reader, err := NewSDKServiceWorkloads(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := reader.Check(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	if len(w.calls) != 10 {
		t.Fatal("service workload observation scope changed", len(w.calls))
	}
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
			t.Fatal("live workload checks repeated fresh inventory")
		}
	}
	w.pages[s.Placement.Services.Backend.Runtime.Resource+"/federatedIdentityCredentials"]["value"] = []any{map[string]string{"id": "unexpected"}}
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh identity check reused safe inventory", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed identity check retained safe inventory", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled inventory used evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed inventory used evidence", err)
	}
}

func TestSandboxServiceWorkloadsRejectSharingOccupantsFederationAndIdentityDrift(t *testing.T) {
	for _, mode := range []string{"relay-occupant", "backend-job", "sharing-app", "sharing-job", "client-alias", "principal-alias", "system-principal", "missing-host", "wrong-environment", "unknown-identity", "client", "lifecycle", "tenant", "missing-client", "federation", "missing-federation", "foreign-resource"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			v := s.Placement.Services
			pin := v.Backend.Runtime
			other := unrelatedServiceWorkload(s, "containerApps")
			kind := "containerApps"
			expected := hostedexecution.ErrOwnership
			identity := map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{pin.Resource: map[string]string{"clientId": pin.Client, "principalId": pin.Principal}}}
			switch mode {
			case "relay-occupant":
				other["properties"] = map[string]string{"environmentId": s.Placement.Private.ExchangeEnvironment}
			case "backend-job":
				kind = "jobs"
				other = unrelatedServiceWorkload(s, kind)
				other["properties"] = map[string]string{"environmentId": v.Backend.Environment}
			case "sharing-app":
				other["identity"] = identity
			case "sharing-job":
				kind = "jobs"
				other = unrelatedServiceWorkload(s, kind)
				other["identity"] = identity
			case "client-alias":
				other["identity"] = map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{pin.Resource + "-other": map[string]string{"clientId": pin.Client, "principalId": v.Relay.Pull.Principal}}}
			case "principal-alias":
				other["identity"] = map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{pin.Resource + "-other": map[string]string{"clientId": v.Relay.Pull.Client, "principalId": pin.Principal}}}
			case "system-principal":
				other["identity"] = map[string]string{"type": "SystemAssigned", "principalId": pin.Principal, "tenantId": s.TenantID}
			case "missing-host":
				w.pages[serviceWorkloadPath(s, kind)]["value"] = []any{w.resources[v.Relay.Application]}
				expected = hostedexecution.ErrUnavailable
			case "wrong-environment":
				w.resources[v.Backend.Application]["properties"].(map[string]any)["environmentId"] = s.Placement.Private.ExchangeEnvironment
			case "unknown-identity":
				delete(other, "identity")
				expected = hostedexecution.ErrUnavailable
			case "client":
				w.resources[v.Backend.Application]["identity"].(map[string]any)["userAssignedIdentities"].(map[string]any)[pin.Resource] = map[string]string{"clientId": v.Relay.Pull.Client, "principalId": pin.Principal}
			case "lifecycle":
				settings := w.resources[v.Backend.Application]["properties"].(map[string]any)["configuration"].(map[string]any)["identitySettings"].([]any)
				for _, raw := range settings {
					item := raw.(map[string]any)
					if item["identity"] == pin.Resource {
						item["lifecycle"] = "All"
					}
				}
			case "tenant":
				w.resources[pin.Resource]["properties"].(map[string]any)["tenantId"] = "other"
			case "missing-client":
				delete(w.resources[pin.Resource]["properties"].(map[string]any), "clientId")
				expected = hostedexecution.ErrUnavailable
			case "federation":
				w.pages[pin.Resource+"/federatedIdentityCredentials"]["value"] = []any{map[string]string{"id": pin.Resource + "/federatedIdentityCredentials/unexpected"}}
			case "missing-federation":
				w.pages[pin.Resource+"/federatedIdentityCredentials"]["value"] = nil
				expected = hostedexecution.ErrUnavailable
			case "foreign-resource":
				other["id"] = strings.Replace(other["id"].(string), s.Worker.Subscription, "87654321-4321-4321-4321-cba987654321", 1)
			}
			addServiceWorkload(w, kind, other)
			reader, err := NewSDKServiceWorkloads(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected {
				t.Fatal("unsafe workload inventory admitted or misclassified", err)
			}
		})
	}
}

func TestSandboxServiceWorkloadInventoriesRequireCompleteScopedPages(t *testing.T) {
	for _, mode := range []string{"paged", "foreign-host", "foreign-path", "duplicate", "missing", "truncated", "federation-continuation", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			path := serviceWorkloadPath(s, "containerApps")
			values := w.pages[path]["value"].([]any)
			calls := 0
			if mode == "federation-continuation" {
				fic := s.Placement.Services.Backend.Runtime.Resource + "/federatedIdentityCredentials"
				w.pages[fic]["nextLink"] = "https://management.azure.com" + fic + "?api-version=2024-11-30&$skipToken=second"
			}
			transport := runtimeTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != path {
					return w.Do(r)
				}
				calls++
				page := map[string]any{"value": values[:1], "nextLink": "https://management.azure.com" + path + "?api-version=2025-07-01&$skipToken=second"}
				if calls > 1 {
					page = map[string]any{"value": values[1:]}
				}
				switch mode {
				case "foreign-host":
					page["nextLink"] = "https://foreign.example" + path + "?api-version=2025-07-01&$skipToken=second"
				case "foreign-path":
					page["nextLink"] = "https://management.azure.com" + serviceWorkloadPath(s, "jobs") + "?api-version=2025-07-01&$skipToken=second"
				case "duplicate":
					page = map[string]any{"value": []any{values[0], values[0]}}
				case "missing":
					page = map[string]any{}
				case "truncated":
					page = map[string]any{"value": []any{}, "nextLink": "https://management.azure.com" + path + "?api-version=2025-07-01&$skipToken=" + strings.Repeat("x", calls)}
				case "unavailable":
					return response(403, `{"error":{"message":"private-canary"}}`), nil
				}
				return jsonResponse(page), nil
			})
			reader, err := NewSDKServiceWorkloads(s, &runtimeCredential{token: "placement-metadata-canary"}, transport)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			err = reader.Check(t.Context(), s.Worker)
			if mode == "paged" {
				if err != nil || calls != 2 {
					t.Fatal("complete workload pages failed", err, calls)
				}
				return
			}
			if err == nil || strings.Contains(err.Error(), "canary") || calls > 16 {
				t.Fatal("incomplete inventory admitted", err, calls)
			}
			if (mode == "foreign-host" || mode == "foreign-path") && calls != 1 {
				t.Fatal("unsafe continuation reached transport")
			}
		})
	}
}

func TestSandboxRuntimeServiceWorkloadsBlockAllAdmissionPaths(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings()
	s.Mode = "discovery"
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
	other := unrelatedServiceWorkload(s, "jobs")
	other["properties"] = map[string]string{"environmentId": s.Placement.Services.Backend.Environment}
	addServiceWorkload(w, "jobs", other)
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("untrusted occupant reached import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("untrusted occupant reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("untrusted occupant reached live commands", err)
	}
	runtime.Close()
	if err := runtime.serviceWorkloads.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left workload reader open", err)
	}
}
