package azuresandbox

import "strings"

func (w *placementWorld) addHostRegistry() {
	w.t.Helper()
	s, h, r := w.s, w.s.Host, w.s.Host.Registry
	put := func(id, kind string, body any) map[string]any {
		value := objectMap(body)
		value["id"], value["name"], value["type"] = id, id[strings.LastIndex(id, "/")+1:], kind
		props, ok := value["properties"].(map[string]any)
		if !ok {
			props = map[string]any{}
			value["properties"] = props
		}
		props["provisioningState"] = "Succeeded"
		w.resources[id] = value
		return value
	}
	defs, err := s.HostRegistryDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	for id, body := range defs {
		kind := "Microsoft.Network/privateDnsZones"
		switch id {
		case r.SecurityGroup:
			kind = "Microsoft.Network/networkSecurityGroups"
		case r.Subnet:
			kind = "Microsoft.Network/virtualNetworks/subnets"
		case r.PullAssignment:
			kind = "Microsoft.Authorization/roleAssignments"
		default:
			if strings.Contains(id, "/A/") {
				kind += "/A"
			}
			if strings.Contains(id, "/virtualNetworkLinks/") {
				kind += "/virtualNetworkLinks"
			}
		}
		put(id, kind, body)
	}
	props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
	props(r.SecurityGroup)["subnets"] = []any{map[string]string{"id": r.Subnet}}
	for _, raw := range props(r.SecurityGroup)["securityRules"].([]any) {
		raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	}
	put(h.Subnet, "Microsoft.Network/virtualNetworks/subnets", map[string]any{"properties": map[string]any{"addressPrefix": h.SubnetPrefix, "delegations": []any{map[string]any{"properties": map[string]string{"serviceName": "Microsoft.App/environments", "provisioningState": "Succeeded"}}}}})
	ips, refs := []any{}, []any{}
	for _, member := range [][2]string{{"registry", r.LoginIP}, {"registry_data_" + s.Worker.Region, r.DataIP}} {
		id := r.Interface + "/ipConfigurations/" + member[0]
		refs = append(refs, map[string]string{"id": id})
		ips = append(ips, map[string]any{"id": id, "name": member[0], "properties": map[string]any{"provisioningState": "Succeeded", "privateIPAddress": member[1], "privateIPAddressVersion": "IPv4", "privateIPAllocationMethod": "Dynamic", "subnet": map[string]string{"id": r.Subnet}, "privateLinkConnectionProperties": map[string]string{"groupId": "registry", "requiredMemberName": member[0]}}})
	}
	put(r.Interface, "Microsoft.Network/networkInterfaces", map[string]any{"location": s.Worker.Region, "properties": map[string]any{"privateEndpoint": map[string]string{"id": r.Endpoint}, "ipConfigurations": ips}})
	put(r.Endpoint, "Microsoft.Network/privateEndpoints", map[string]any{"location": s.Worker.Region, "properties": map[string]any{"subnet": map[string]string{"id": r.Subnet}, "networkInterfaces": []any{map[string]string{"id": r.Interface}}, "privateLinkServiceConnections": []any{map[string]any{"name": "registry", "properties": map[string]any{"provisioningState": "Succeeded", "privateLinkServiceId": s.Placement.Registry, "groupIds": []string{"registry"}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}}}})
	props(r.Subnet)["privateEndpoints"], props(r.Subnet)["ipConfigurations"] = []any{map[string]string{"id": r.Endpoint}}, refs
	put(r.Connection, "Microsoft.ContainerRegistry/registries/privateEndpointConnections", map[string]any{"properties": map[string]any{"privateEndpoint": map[string]string{"id": r.Endpoint}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}})
	props(r.Zone)["numberOfRecordSets"], props(r.Zone)["numberOfVirtualNetworkLinks"], props(r.Zone)["numberOfVirtualNetworkLinksWithRegistration"] = 3, 1, 0
	link := w.resources[r.Zone+"/virtualNetworkLinks/controller-network"]
	link["properties"].(map[string]any)["virtualNetworkLinkState"] = "Completed"
	w.pages[r.Zone+"/virtualNetworkLinks"] = map[string]any{"value": []any{link}}
	records := []any{map[string]any{"id": r.Zone + "/SOA/@", "name": "@", "type": "Microsoft.Network/privateDnsZones/SOA", "properties": map[string]any{"ttl": 60, "soaRecord": map[string]any{"host": "dns.teststudio.example", "email": "operator.teststudio.example", "serialNumber": 1, "refreshTime": 60, "retryTime": 60, "expireTime": 3600, "minimumTtl": 60}}}}
	for name := range h.registryZone(s).Records {
		records = append(records, w.resources[r.Zone+"/A/"+name])
	}
	w.pages[r.Zone+"/ALL"] = map[string]any{"value": records}
	root := "/subscriptions/" + s.Worker.Subscription + "/providers/Microsoft.Network/privateDnsZones"
	w.pages[root]["value"] = append(w.pages[root]["value"].([]any), w.resources[r.Zone])
}
