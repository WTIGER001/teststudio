package azuresandbox

import "strings"

func (w *placementWorld) addServiceWorkloads() {
	s, v := w.s, w.s.Placement.Services
	for _, pin := range v.identities() {
		w.resources[pin.Resource] = map[string]any{"id": pin.Resource, "name": pin.Resource[strings.LastIndex(pin.Resource, "/")+1:], "type": "Microsoft.ManagedIdentity/userAssignedIdentities", "location": s.Worker.Region, "properties": map[string]any{"clientId": pin.Client, "principalId": pin.Principal, "tenantId": s.TenantID}}
		w.pages[pin.Resource+"/federatedIdentityCredentials"] = map[string]any{"value": []any{}}
	}
	w.pages[serviceWorkloadPath(s, "containerApps")] = map[string]any{"value": []any{w.resources[v.Relay.Application], w.resources[v.Backend.Application]}}
	w.pages[serviceWorkloadPath(s, "jobs")] = map[string]any{"value": []any{}}
}
