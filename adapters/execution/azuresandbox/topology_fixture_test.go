package azuresandbox

import (
	"encoding/json"
	"strings"
)

func (w *placementWorld) addTopology() {
	w.t.Helper()
	s, p := w.s, w.s.Placement
	t := p.Network.Topology
	defs, err := s.TopologyDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	for id, definition := range defs {
		data, _ := json.Marshal(definition)
		var value map[string]any
		if json.Unmarshal(data, &value) != nil {
			w.t.Fatal("invalid topology resource body")
		}
		value["id"], value["name"] = id, id[strings.LastIndex(id, "/")+1:]
		kind := "Microsoft.Network/"
		switch id {
		case t.Firewall:
			kind += "azureFirewalls"
		case t.Policy:
			kind += "firewallPolicies"
		case t.PublicIP:
			kind += "publicIPAddresses"
		case p.firewallSubnet():
			kind += "virtualNetworks/subnets"
		default:
			kind += "firewallPolicies/ruleCollectionGroups"
		}
		value["type"] = kind
		props := value["properties"].(map[string]any)
		props["provisioningState"] = "Succeeded"
		w.resources[id] = value
	}
	configuration := t.Firewall + "/azureFirewallIpConfigurations/egress"
	firewall := w.resources[t.Firewall]["properties"].(map[string]any)
	ip := firewall["ipConfigurations"].([]any)[0].(map[string]any)
	ip["id"] = configuration
	ip["properties"].(map[string]any)["provisioningState"] = "Succeeded"
	ip["properties"].(map[string]any)["privateIPAddress"] = p.Network.FirewallIP
	public := w.resources[t.PublicIP]["properties"].(map[string]any)
	public["ipAddress"], public["ipConfiguration"] = "203.0.113.20", map[string]string{"id": configuration}
	subnet := w.resources[p.firewallSubnet()]["properties"].(map[string]any)
	subnet["ipConfigurations"] = []any{map[string]string{"id": configuration}}
	policy := w.resources[t.Policy]["properties"].(map[string]any)
	policy["firewalls"] = []any{map[string]string{"id": t.Firewall}}
	groupID := t.Policy + "/ruleCollectionGroups/" + sandboxFirewallGroup
	policy["ruleCollectionGroups"] = []any{map[string]string{"id": groupID}}
	w.pages[t.Policy+"/ruleCollectionGroups"] = map[string]any{"value": []any{w.resources[groupID]}}
	id := p.virtualNetwork()
	w.resources[id] = map[string]any{"id": id, "name": id[strings.LastIndex(id, "/")+1:], "type": "Microsoft.Network/virtualNetworks", "location": s.Worker.Region, "properties": map[string]any{"provisioningState": "Succeeded", "addressSpace": map[string]any{"addressPrefixes": []string{t.Prefix}}, "subnets": []any{w.resources[p.Subnet], w.resources[p.Private.Subnet], w.resources[p.firewallSubnet()]}}}
}
