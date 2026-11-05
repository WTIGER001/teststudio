package azuresandbox

import "strings"

func (w *placementWorld) addHostDatabase() {
	w.t.Helper()
	s, h, d := w.s, w.s.Host, w.s.Host.Database
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
	defs, err := s.HostDatabaseDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	for id, body := range defs {
		kind := "Microsoft.Network/privateDnsZones"
		switch id {
		case d.SecurityGroup:
			kind = "Microsoft.Network/networkSecurityGroups"
		case d.Subnet:
			kind = "Microsoft.Network/virtualNetworks/subnets"
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
	props(d.SecurityGroup)["subnets"] = []any{map[string]string{"id": d.Subnet}}
	for _, raw := range props(d.SecurityGroup)["securityRules"].([]any) {
		raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	}
	ipID := d.Interface + "/ipConfigurations/postgresqlServer"
	put(d.Interface, "Microsoft.Network/networkInterfaces", map[string]any{"location": s.Worker.Region, "properties": map[string]any{"privateEndpoint": map[string]string{"id": d.Endpoint}, "ipConfigurations": []any{map[string]any{"id": ipID, "name": "postgresqlServer", "properties": map[string]any{"provisioningState": "Succeeded", "privateIPAddress": d.Address, "privateIPAddressVersion": "IPv4", "privateIPAllocationMethod": "Dynamic", "subnet": map[string]string{"id": d.Subnet}, "privateLinkConnectionProperties": map[string]string{"groupId": "postgresqlServer", "requiredMemberName": "postgresqlServer"}}}}}})
	put(d.Endpoint, "Microsoft.Network/privateEndpoints", map[string]any{"location": s.Worker.Region, "properties": map[string]any{"subnet": map[string]string{"id": d.Subnet}, "networkInterfaces": []any{map[string]string{"id": d.Interface}}, "privateLinkServiceConnections": []any{map[string]any{"name": "database", "properties": map[string]any{"provisioningState": "Succeeded", "privateLinkServiceId": d.Server, "groupIds": []string{"postgresqlServer"}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}}}})
	props(d.Subnet)["privateEndpoints"], props(d.Subnet)["ipConfigurations"] = []any{map[string]string{"id": d.Endpoint}}, []any{map[string]string{"id": ipID}}
	connections := []any{}
	for i, endpoint := range d.consumers() {
		id := d.Server + "/privateEndpointConnections/consumer-" + string(rune('a'+i)) + ".11111111-1111-1111-1111-111111111111"
		connections = append(connections, put(id, "Microsoft.DBforPostgreSQL/flexibleServers/privateEndpointConnections", map[string]any{"properties": map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": endpoint}, "groupIds": []string{"postgresqlServer"}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}))
	}
	put(d.Server, "Microsoft.DBforPostgreSQL/flexibleServers", map[string]any{"location": s.Worker.Region, "properties": map[string]any{"state": "Ready", "fullyQualifiedDomainName": h.DatabaseHost, "network": map[string]string{"publicNetworkAccess": "Disabled"}, "authConfig": map[string]string{"passwordAuth": "Enabled"}, "privateEndpointConnections": connections}})
	w.pages[d.Server+"/privateEndpointConnections"] = map[string]any{"value": connections}
	link := put(d.Server+"/privateLinkResources/postgresqlServer", "Microsoft.DBforPostgreSQL/flexibleServers/privateLinkResources", map[string]any{"properties": map[string]any{"groupId": "postgresqlServer", "requiredMembers": []string{"postgresqlServer"}, "requiredZoneNames": []string{postgresPrivateZone}}})
	w.pages[d.Server+"/privateLinkResources"] = map[string]any{"value": []any{link}}
	props(d.Zone)["numberOfRecordSets"], props(d.Zone)["numberOfVirtualNetworkLinks"], props(d.Zone)["numberOfVirtualNetworkLinksWithRegistration"] = 2, 1, 0
	zoneLink := w.resources[d.Zone+"/virtualNetworkLinks/controller-database"]
	zoneLink["properties"].(map[string]any)["virtualNetworkLinkState"] = "Completed"
	w.pages[d.Zone+"/virtualNetworkLinks"] = map[string]any{"value": []any{zoneLink}}
	w.pages[d.Zone+"/ALL"] = map[string]any{"value": []any{map[string]any{"id": d.Zone + "/SOA/@", "name": "@", "type": "Microsoft.Network/privateDnsZones/SOA", "properties": map[string]any{"ttl": 60, "soaRecord": map[string]any{"host": "dns.teststudio.example", "email": "operator.teststudio.example", "serialNumber": 1, "refreshTime": 60, "retryTime": 60, "expireTime": 3600, "minimumTtl": 60}}}, w.resources[d.Zone+"/A/"+d.name()]}}
	root := "/subscriptions/" + s.Worker.Subscription + "/providers/Microsoft.Network/privateDnsZones"
	w.pages[root]["value"] = append(w.pages[root]["value"].([]any), w.resources[d.Zone])
}
