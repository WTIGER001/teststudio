package azuresandbox

import (
	"encoding/json"
	"slices"
	"strings"
)

func (w *placementWorld) addResolver() {
	w.t.Helper()
	s, r := w.s, w.s.Placement.Network.Resolver
	defs, err := s.ResolverDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	rules, links := []any{}, []any{}
	ids := []string{}
	for id := range defs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		data, _ := json.Marshal(defs[id])
		var value map[string]any
		if json.Unmarshal(data, &value) != nil {
			w.t.Fatal("invalid resolver definition")
		}
		props, ok := value["properties"].(map[string]any)
		if !ok {
			props = map[string]any{}
			value["properties"] = props
		}
		props["provisioningState"] = "Succeeded"
		value["id"], value["name"] = id, id[strings.LastIndex(id, "/")+1:]
		kind := "Microsoft.Network/dnsResolverDomainLists"
		switch {
		case id == r.Policy:
			kind = "Microsoft.Network/dnsResolverPolicies"
		case strings.Contains(id, "/dnsSecurityRules/"):
			kind = "Microsoft.Network/dnsResolverPolicies/dnsSecurityRules"
			rules = append(rules, value)
		case strings.Contains(id, "/virtualNetworkLinks/"):
			kind = "Microsoft.Network/dnsResolverPolicies/virtualNetworkLinks"
			links = append(links, value)
		}
		value["type"] = kind
		w.resources[id] = value
	}
	w.pages[r.Policy+"/dnsSecurityRules"] = map[string]any{"value": rules}
	w.pages[r.Policy+"/virtualNetworkLinks"] = map[string]any{"value": links}
	w.pages[s.Placement.virtualNetwork()+"/listDnsResolverPolicies"] = map[string]any{"value": []any{map[string]string{"id": r.Policy}}}
	w.pages["/subscriptions/"+s.Worker.Subscription+"/providers/Microsoft.Network/privateDnsZones"] = map[string]any{"value": []any{w.resources[s.Placement.Private.ExchangeZone], w.resources[s.Placement.Private.RegistryZone]}}
}
