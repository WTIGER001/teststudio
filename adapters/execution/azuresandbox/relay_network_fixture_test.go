package azuresandbox

import "strings"

func runtimeRelayNetworkSettings(c Config, p PlacementSettings, v ServicesSettings) RelayNetworkSettings {
	base := "/subscriptions/" + c.Subscription + "/resourceGroups/studio-relay/providers/Microsoft.Network/"
	return RelayNetworkSettings{
		Topology:   TopologySettings{Prefix: "10.60.0.0/16", Firewall: base + "azureFirewalls/studio-relay-egress", Policy: base + "firewallPolicies/studio-relay-egress", PublicIP: base + "publicIPAddresses/studio-relay-egress", FirewallPrefix: "10.60.4.0/26"},
		FirewallIP: "10.60.4.4", SecurityGroup: base + "networkSecurityGroups/studio-relay", RouteTable: base + "routeTables/studio-relay",
		Resolver: ResolverSettings{Policy: base + "dnsResolverPolicies/studio-relay", AllowList: base + "dnsResolverDomainLists/studio-relay-allow", BlockList: base + "dnsResolverDomainLists/studio-relay-block"},
		Registry: RelayRegistrySettings{Endpoint: base + "privateEndpoints/studio-relay-registry", Interface: base + "networkInterfaces/studio-relay-registry", Subnet: strings.TrimSuffix(v.Relay.Subnet, "services") + "registry", SubnetPrefix: "10.60.3.0/24", SecurityGroup: base + "networkSecurityGroups/studio-relay-registry", LoginIP: "10.60.3.4", DataIP: "10.60.3.5", Zone: base + "privateDnsZones/privatelink.azurecr.io", PullAssignment: p.Registry + "/providers/Microsoft.Authorization/roleAssignments/87777777-7777-7777-7777-777777777777", Connection: p.Registry + "/privateEndpointConnections/studio-relay"},
	}
}

func (w *placementWorld) addRelayNetwork() {
	w.t.Helper()
	s, v := w.s, w.s.Placement.Services
	r := v.RelayNetwork
	defs, err := s.RelayNetworkDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	for id, body := range defs {
		value := objectMap(body)
		parts := strings.Split(id, "/")
		kind := parts[6] + "/" + parts[7]
		for i := 9; i < len(parts); i += 2 {
			if parts[i] == "providers" {
				kind = parts[i+1] + "/" + parts[i+2]
				i += 3
			} else {
				kind += "/" + parts[i]
			}
		}
		value["id"], value["name"], value["type"] = id, parts[len(parts)-1], kind
		props, ok := value["properties"].(map[string]any)
		if !ok {
			props = map[string]any{}
			value["properties"] = props
		}
		props["provisioningState"] = "Succeeded"
		w.resources[id] = value
	}
	prop := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
	for _, item := range [][2]string{{r.SecurityGroup, v.Relay.Subnet}, {r.Registry.SecurityGroup, r.Registry.Subnet}, {r.RouteTable, v.Relay.Subnet}} {
		props := prop(item[0])
		props["subnets"] = []any{map[string]string{"id": item[1]}}
		child := "securityRules"
		if item[0] == r.RouteTable {
			child = "routes"
		}
		for _, raw := range props[child].([]any) {
			raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
		}
	}
	for _, raw := range prop(v.Relay.Subnet)["delegations"].([]any) {
		raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	}
	configuration := r.Topology.Firewall + "/azureFirewallIpConfigurations/egress"
	ip := prop(r.Topology.Firewall)["ipConfigurations"].([]any)[0].(map[string]any)
	ip["id"] = configuration
	ip["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	ip["properties"].(map[string]any)["privateIPAddress"] = r.FirewallIP
	prop(r.Topology.PublicIP)["ipAddress"], prop(r.Topology.PublicIP)["ipConfiguration"] = "203.0.113.30", map[string]string{"id": configuration}
	firewallSubnet := r.virtualNetwork(s) + "/subnets/AzureFirewallSubnet"
	prop(firewallSubnet)["ipConfigurations"] = []any{map[string]string{"id": configuration}}
	groupID := r.Topology.Policy + "/ruleCollectionGroups/relay-platform"
	prop(r.Topology.Policy)["firewalls"] = []any{map[string]string{"id": r.Topology.Firewall}}
	prop(r.Topology.Policy)["ruleCollectionGroups"] = []any{map[string]string{"id": groupID}}
	w.pages[r.Topology.Policy+"/ruleCollectionGroups"] = map[string]any{"value": []any{w.resources[groupID]}}
	put := func(id, kind, region string, props map[string]any) map[string]any {
		value := map[string]any{"id": id, "name": id[strings.LastIndex(id, "/")+1:], "type": kind, "properties": props}
		if region != "" {
			value["location"] = region
		}
		w.resources[id] = value
		return value
	}
	ips, refs := []any{}, []any{}
	for _, member := range []struct{ name, address string }{{"registry", r.Registry.LoginIP}, {"registry_data_" + s.Worker.Region, r.Registry.DataIP}} {
		id := r.Registry.Interface + "/ipConfigurations/" + member.name
		ips = append(ips, map[string]any{"id": id, "name": member.name, "properties": map[string]any{"provisioningState": "Succeeded", "privateIPAddress": member.address, "privateIPAddressVersion": "IPv4", "privateIPAllocationMethod": "Dynamic", "subnet": map[string]string{"id": r.Registry.Subnet}, "privateLinkConnectionProperties": map[string]string{"groupId": "registry", "requiredMemberName": member.name}}})
		refs = append(refs, map[string]string{"id": id})
	}
	put(r.Registry.Interface, "Microsoft.Network/networkInterfaces", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": r.Registry.Endpoint}, "ipConfigurations": ips})
	put(r.Registry.Endpoint, "Microsoft.Network/privateEndpoints", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "subnet": map[string]string{"id": r.Registry.Subnet}, "networkInterfaces": []any{map[string]string{"id": r.Registry.Interface}}, "privateLinkServiceConnections": []any{map[string]any{"name": "registry", "properties": map[string]any{"provisioningState": "Succeeded", "privateLinkServiceId": s.Placement.Registry, "groupIds": []string{"registry"}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}}})
	prop(r.Registry.Subnet)["privateEndpoints"] = []any{map[string]string{"id": r.Registry.Endpoint}}
	prop(r.Registry.Subnet)["ipConfigurations"] = refs
	put(r.virtualNetwork(s), "Microsoft.Network/virtualNetworks", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "addressSpace": map[string]any{"addressPrefixes": []string{r.Topology.Prefix}}, "subnets": []any{w.resources[v.Relay.Subnet], w.resources[v.BackendEndpoint.Subnet], w.resources[r.Registry.Subnet], w.resources[firewallSubnet]}})
	connection := put(r.Registry.Connection, "Microsoft.ContainerRegistry/registries/privateEndpointConnections", "", map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": r.Registry.Endpoint}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}})
	prop(s.Placement.Registry)["privateEndpointConnections"] = []any{connection}
	put(v.Relay.Pull.Resource, "Microsoft.ManagedIdentity/userAssignedIdentities", s.Worker.Region, map[string]any{"clientId": v.Relay.Pull.Client, "principalId": v.Relay.Pull.Principal, "tenantId": s.TenantID})
	rules := []any{w.resources[r.Resolver.Policy+"/dnsSecurityRules/relay-allow-approved"], w.resources[r.Resolver.Policy+"/dnsSecurityRules/relay-default-block"]}
	w.pages[r.Resolver.Policy+"/dnsSecurityRules"] = map[string]any{"value": rules}
	w.pages[r.Resolver.Policy+"/virtualNetworkLinks"] = map[string]any{"value": []any{w.resources[r.Resolver.Policy+"/virtualNetworkLinks/relay-network"]}}
	w.pages[r.virtualNetwork(s)+"/listDnsResolverPolicies"] = map[string]any{"value": []any{map[string]string{"id": r.Resolver.Policy}}}
	w.addRelayRegistryZone()
}

func (w *placementWorld) addRelayRegistryZone() {
	r := w.s.Placement.Services.RelayNetwork
	zone := w.resources[r.Registry.Zone]
	props := zone["properties"].(map[string]any)
	props["numberOfRecordSets"], props["numberOfVirtualNetworkLinks"], props["numberOfVirtualNetworkLinksWithRegistration"] = 3, 1, 0
	link := w.resources[r.Registry.Zone+"/virtualNetworkLinks/relay-network"]
	link["properties"].(map[string]any)["virtualNetworkLinkState"] = "Completed"
	w.pages[r.Registry.Zone+"/virtualNetworkLinks"] = map[string]any{"value": []any{link}}
	records := []any{map[string]any{"id": r.Registry.Zone + "/SOA/@", "name": "@", "type": "Microsoft.Network/privateDnsZones/SOA", "properties": map[string]any{"ttl": 60, "soaRecord": map[string]any{"host": "dns.teststudio.example", "email": "operator.teststudio.example", "serialNumber": 1, "refreshTime": 60, "retryTime": 60, "expireTime": 3600, "minimumTtl": 60}}}}
	for name := range r.zoneBinding(w.s).Records {
		records = append(records, w.resources[r.Registry.Zone+"/A/"+name])
	}
	w.pages[r.Registry.Zone+"/ALL"] = map[string]any{"value": records}
	root := "/subscriptions/" + w.s.Worker.Subscription + "/providers/Microsoft.Network/privateDnsZones"
	w.pages[root]["value"] = append(w.pages[root]["value"].([]any), zone)
}
