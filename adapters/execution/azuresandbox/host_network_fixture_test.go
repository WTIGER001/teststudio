package azuresandbox

import "strings"

func (w *placementWorld) addHostNetwork() {
	w.t.Helper()
	s, h, r := w.s, w.s.Host, w.s.Host.Network
	defs, err := s.HostNetworkDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	for id, body := range defs {
		value := objectMap(body)
		parts := strings.Split(id, "/")
		kind := parts[6] + "/" + parts[7]
		for i := 9; i < len(parts); i += 2 {
			kind += "/" + parts[i]
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
	props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
	for _, id := range []string{r.SecurityGroup, r.RouteTable} {
		props(id)["subnets"] = []any{map[string]string{"id": h.Subnet}}
		child := "securityRules"
		if id == r.RouteTable {
			child = "routes"
		}
		for _, raw := range props(id)[child].([]any) {
			raw.(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
		}
	}
	props(h.Subnet)["delegations"].([]any)[0].(map[string]any)["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	configuration := r.Topology.Firewall + "/azureFirewallIpConfigurations/egress"
	ip := props(r.Topology.Firewall)["ipConfigurations"].([]any)[0].(map[string]any)
	ip["id"] = configuration
	ip["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	ip["properties"].(map[string]any)["privateIPAddress"] = r.FirewallIP
	props(r.Topology.PublicIP)["ipAddress"], props(r.Topology.PublicIP)["ipConfiguration"] = "203.0.113.40", map[string]string{"id": configuration}
	firewallSubnet := h.firewallBinding().Subnet
	props(firewallSubnet)["ipConfigurations"] = []any{map[string]string{"id": configuration}}
	groupID := r.Topology.Policy + "/ruleCollectionGroups/controller-platform"
	props(r.Topology.Policy)["firewalls"] = []any{map[string]string{"id": r.Topology.Firewall}}
	props(r.Topology.Policy)["ruleCollectionGroups"] = []any{map[string]string{"id": groupID}}
	w.pages[r.Topology.Policy+"/ruleCollectionGroups"] = map[string]any{"value": []any{w.resources[groupID]}}
	vnet := h.virtualNetwork()
	w.resources[vnet] = map[string]any{"id": vnet, "name": "studio-control", "type": "Microsoft.Network/virtualNetworks", "location": s.Worker.Region, "properties": map[string]any{"provisioningState": "Succeeded", "addressSpace": map[string]any{"addressPrefixes": []string{r.Topology.Prefix}}, "subnets": []any{w.resources[h.Subnet], w.resources[h.Registry.Subnet], w.resources[firewallSubnet], w.resources[h.Database.Subnet]}}}
	w.pages[r.Resolver.Policy+"/dnsSecurityRules"] = map[string]any{"value": []any{w.resources[r.Resolver.Policy+"/dnsSecurityRules/controller-allow-approved"], w.resources[r.Resolver.Policy+"/dnsSecurityRules/controller-default-block"]}}
	link := w.resources[r.Resolver.Policy+"/virtualNetworkLinks/controller-network"]
	link["properties"].(map[string]any)["virtualNetworkLinkState"] = "Completed"
	w.pages[r.Resolver.Policy+"/virtualNetworkLinks"] = map[string]any{"value": []any{link}}
	w.pages[vnet+"/listDnsResolverPolicies"] = map[string]any{"value": []any{map[string]string{"id": r.Resolver.Policy}}}
}
