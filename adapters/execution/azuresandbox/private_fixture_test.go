package azuresandbox

import (
	"encoding/json"
	"slices"
	"strings"
)

func (w *placementWorld) addPrivate() {
	w.t.Helper()
	w.pages = map[string]map[string]any{}
	put := func(id, kind, region string, props map[string]any) map[string]any {
		r := map[string]any{"id": id, "name": id[strings.LastIndex(id, "/")+1:], "type": kind, "properties": props}
		if region != "" {
			r["location"] = region
		}
		w.resources[id] = r
		return r
	}
	s := w.s
	p := s.Placement.Private
	ipReferences, endpoints := []any{}, []any{}
	for _, b := range p.endpointBindings(s) {
		members := []string{}
		for member := range b.Members {
			members = append(members, member)
		}
		slices.Sort(members)
		configs := []any{}
		for _, member := range members {
			id := b.Interface + "/ipConfigurations/" + member
			configs = append(configs, map[string]any{"id": id, "name": member, "properties": map[string]any{"provisioningState": "Succeeded", "privateIPAddress": b.Members[member], "privateIPAddressVersion": "IPv4", "privateIPAllocationMethod": "Dynamic", "subnet": map[string]string{"id": p.Subnet}, "privateLinkConnectionProperties": map[string]string{"groupId": b.Group, "requiredMemberName": member}}})
			ipReferences = append(ipReferences, map[string]string{"id": id})
		}
		put(b.Interface, "Microsoft.Network/networkInterfaces", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": b.Endpoint}, "ipConfigurations": configs})
		put(b.Endpoint, "Microsoft.Network/privateEndpoints", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "subnet": map[string]string{"id": p.Subnet}, "networkInterfaces": []any{map[string]string{"id": b.Interface}}, "privateLinkServiceConnections": []any{map[string]any{"name": "studio-link", "properties": map[string]any{"provisioningState": "Succeeded", "privateLinkServiceId": b.Service, "groupIds": []string{b.Group}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}}})
		endpoints = append(endpoints, map[string]string{"id": b.Endpoint})
	}
	put(p.Subnet, "Microsoft.Network/virtualNetworks/subnets", "", map[string]any{"provisioningState": "Succeeded", "addressPrefix": p.SubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": p.SecurityGroup}, "privateEndpointNetworkPolicies": "Enabled", "privateLinkServiceNetworkPolicies": "Enabled", "privateEndpoints": endpoints, "ipConfigurations": ipReferences})
	rules := p.rules(s)
	for _, raw := range rules {
		raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	}
	put(p.SecurityGroup, "Microsoft.Network/networkSecurityGroups", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "flushConnection": true, "subnets": []any{map[string]string{"id": p.Subnet}}, "securityRules": rules})
	for _, b := range p.zoneBindings(s) {
		put(b.ID, "Microsoft.Network/privateDnsZones", "global", map[string]any{"provisioningState": "Succeeded", "numberOfRecordSets": len(b.Records) + 1, "numberOfVirtualNetworkLinks": 1, "numberOfVirtualNetworkLinksWithRegistration": 0})
		link := b.LinkDefinition()
		encoded, _ := json.Marshal(link)
		var fields map[string]any
		if json.Unmarshal(encoded, &fields) != nil {
			w.t.Fatal("invalid link definition")
		}
		props := fields["properties"].(map[string]any)
		props["provisioningState"], props["virtualNetworkLinkState"] = "Succeeded", "Completed"
		id := b.ID + "/virtualNetworkLinks/" + b.LinkName
		w.pages[b.ID+"/virtualNetworkLinks"] = map[string]any{"value": []any{put(id, "Microsoft.Network/privateDnsZones/virtualNetworkLinks", "global", props)}}
		records := []any{map[string]any{"id": b.ID + "/SOA/@", "name": "@", "type": "Microsoft.Network/privateDnsZones/SOA", "properties": map[string]any{"ttl": 60, "soaRecord": map[string]any{"host": "dns.teststudio.example", "email": "operator.teststudio.example", "serialNumber": 1, "refreshTime": 60, "retryTime": 60, "expireTime": 3600, "minimumTtl": 60}}}}
		names := []string{}
		for name := range b.Records {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			records = append(records, map[string]any{"id": b.ID + "/A/" + name, "name": name, "type": "Microsoft.Network/privateDnsZones/A", "properties": map[string]any{"ttl": 60, "aRecords": []any{map[string]string{"ipv4Address": b.Records[name]}}}})
		}
		w.pages[b.ID+"/ALL"] = map[string]any{"value": records}
	}
}
