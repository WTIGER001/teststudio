package azuresandbox

import "strings"

func (w *placementWorld) addRegistryService() {
	s, p := w.s, w.s.Placement
	props := w.resources[p.Registry]["properties"].(map[string]any)
	props["networkRuleBypassAllowedForTasks"] = false
	props["networkRuleSet"] = map[string]any{"defaultAction": "Deny", "ipRules": []any{}}
	props["policies"] = map[string]any{"exportPolicy": map[string]string{"status": "disabled"}, "azureADAuthenticationAsArmPolicy": map[string]string{"status": "enabled"}}
	props["dataEndpointHostNames"] = []string{strings.TrimSuffix(s.Worker.Registry, ".azurecr.io") + "." + s.Worker.Region + ".data.azurecr.io"}
	connections := []any{}
	for i, endpoint := range s.registryBinding().Endpoints {
		connection := w.resources[p.Services.RelayNetwork.Registry.Connection]
		if endpoint != p.Services.RelayNetwork.Registry.Endpoint {
			name := "consumer-" + string(rune('a'+i))
			id := p.Registry + "/privateEndpointConnections/" + name
			connection = map[string]any{"id": id, "name": name, "type": "Microsoft.ContainerRegistry/registries/privateEndpointConnections", "properties": map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": endpoint}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}
			w.resources[id] = connection
		}
		connections = append(connections, connection)
	}
	props["privateEndpointConnections"] = connections
	w.pages[p.Registry+"/privateEndpointConnections"] = map[string]any{"value": connections}
	id := p.Registry + "/privateLinkResources/registry"
	w.pages[p.Registry+"/privateLinkResources"] = map[string]any{"value": []any{map[string]any{"id": id, "name": "registry", "type": "Microsoft.ContainerRegistry/registries/privateLinkResources", "properties": map[string]any{"groupId": "registry", "requiredMembers": []string{"registry", "registry_data_" + s.Worker.Region}, "requiredZoneNames": []string{"privatelink.azurecr.io"}}}}}
	w.pages[p.Registry+"/replications"] = map[string]any{"value": []any{}}
}
