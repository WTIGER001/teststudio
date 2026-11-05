package azuresandbox

import "strings"

func (w *placementWorld) addBackendEndpoint() {
	w.t.Helper()
	s, v := w.s, w.s.Placement.Services
	e := v.BackendEndpoint
	put := func(id, kind, region string, props map[string]any) map[string]any {
		value := map[string]any{"id": id, "name": id[strings.LastIndex(id, "/")+1:], "type": kind, "properties": props}
		if region != "" {
			value["location"] = region
		}
		w.resources[id] = value
		return value
	}
	member := "backend-service"
	ipID := e.Interface + "/ipConfigurations/backend"
	put(e.Interface, "Microsoft.Network/networkInterfaces", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": e.Endpoint}, "ipConfigurations": []any{map[string]any{"id": ipID, "name": "backend", "properties": map[string]any{"provisioningState": "Succeeded", "privateIPAddress": v.Backend.Address, "privateIPAddressVersion": "IPv4", "privateIPAllocationMethod": "Dynamic", "subnet": map[string]string{"id": e.Subnet}, "privateLinkConnectionProperties": map[string]string{"groupId": "managedEnvironment", "requiredMemberName": member}}}}})
	put(e.Endpoint, "Microsoft.Network/privateEndpoints", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "subnet": map[string]string{"id": e.Subnet}, "networkInterfaces": []any{map[string]string{"id": e.Interface}}, "privateLinkServiceConnections": []any{map[string]any{"name": "backend", "properties": map[string]any{"provisioningState": "Succeeded", "privateLinkServiceId": v.Backend.Environment, "groupIds": []string{"managedEnvironment"}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}}})
	put(e.Subnet, "Microsoft.Network/virtualNetworks/subnets", "", map[string]any{"provisioningState": "Succeeded", "addressPrefix": e.SubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": e.SecurityGroup}, "privateEndpointNetworkPolicies": "Enabled", "privateLinkServiceNetworkPolicies": "Enabled", "privateEndpoints": []any{map[string]string{"id": e.Endpoint}}, "ipConfigurations": []any{map[string]string{"id": ipID}}})
	rules := e.rules(s)
	for _, raw := range rules {
		raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	}
	put(e.SecurityGroup, "Microsoft.Network/networkSecurityGroups", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "flushConnection": true, "subnets": []any{map[string]string{"id": e.Subnet}}, "securityRules": rules})
	put(v.Relay.Subnet, "Microsoft.Network/virtualNetworks/subnets", "", map[string]any{"provisioningState": "Succeeded", "addressPrefix": e.RelaySubnetPrefix, "delegations": []any{map[string]any{"name": "relay-service", "properties": map[string]string{"serviceName": "Microsoft.App/environments", "provisioningState": "Succeeded"}}}})
	connections := []any{map[string]any{"id": v.Backend.Environment + "/privateEndpointConnections/studio-backend", "name": "studio-backend", "type": "Microsoft.App/managedEnvironments/privateEndpointConnections", "properties": map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": e.Endpoint}, "groupIds": []string{"managedEnvironment"}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}}
	w.resources[v.Backend.Environment]["properties"].(map[string]any)["privateEndpointConnections"] = connections
	w.pages[v.Backend.Environment+"/privateEndpointConnections"] = map[string]any{"value": connections}
	w.pages[v.Backend.Environment+"/privateLinkResources"] = map[string]any{"value": []any{map[string]any{"id": v.Backend.Environment + "/privateLinkResources/managedEnvironment", "name": "managedEnvironment", "type": "Microsoft.App/managedEnvironments/privateLinkResources", "properties": map[string]any{"groupId": "managedEnvironment", "requiredMembers": []string{member}, "requiredZoneNames": []string{"privatelink.westus2.azurecontainerapps.io"}}}}}
}
