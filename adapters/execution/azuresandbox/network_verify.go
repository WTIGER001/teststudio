package azuresandbox

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func networkProperties(r placementResource, keys ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := properties(r, &fields); err != nil || fields == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	var state string
	if json.Unmarshal(fields["provisioningState"], &state) != nil || provisioned(state) != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	for key := range fields {
		if key != "provisioningState" && !slices.Contains(keys, key) {
			return nil, hostedexecution.ErrOwnership
		}
	}
	return fields, nil
}

func networkSubnet(raw json.RawMessage, subnet string) bool {
	var values []struct {
		ID string `json:"id"`
	}
	return json.Unmarshal(raw, &values) == nil && len(values) == 1 && strings.EqualFold(values[0].ID, subnet)
}

func networkFlag(raw json.RawMessage, wanted bool) bool {
	var flag *bool
	return json.Unmarshal(raw, &flag) == nil && flag != nil && *flag == wanted
}

func (s Settings) verifySecurityGroup(r placementResource) error {
	p := s.Placement
	return verifyNetworkGroup(r, p.SecurityGroup, p.Subnet, s.Worker.Region, p.networkRules())
}

func verifyNetworkGroup(r placementResource, id, subnet, region string, rules []any) error {
	if !namedResource(r, id, "Microsoft.Network/networkSecurityGroups") || !strings.EqualFold(r.Location, region) {
		return hostedexecution.ErrOwnership
	}
	fields, err := networkProperties(r, "flushConnection", "securityRules", "defaultSecurityRules", "networkInterfaces", "subnets", "resourceGuid")
	if err != nil {
		return err
	}
	var interfaces []json.RawMessage
	if !networkFlag(fields["flushConnection"], true) || !networkSubnet(fields["subnets"], subnet) || len(fields["networkInterfaces"]) > 0 && (json.Unmarshal(fields["networkInterfaces"], &interfaces) != nil || len(interfaces) != 0) {
		return hostedexecution.ErrOwnership
	}
	return verifyNetworkChildren(fields["securityRules"], id, "Microsoft.Network/networkSecurityGroups", "securityRules", rules)
}

func (s Settings) verifyRouteTable(r placementResource) error {
	p := s.Placement
	if !namedResource(r, p.RouteTable, "Microsoft.Network/routeTables") || !strings.EqualFold(r.Location, s.Worker.Region) {
		return hostedexecution.ErrOwnership
	}
	fields, err := networkProperties(r, "disableBgpRoutePropagation", "routes", "subnets", "resourceGuid")
	if err != nil {
		return err
	}
	if !networkFlag(fields["disableBgpRoutePropagation"], true) || !networkSubnet(fields["subnets"], p.Subnet) {
		return hostedexecution.ErrOwnership
	}
	wanted := p.networkDefinitions(s.Worker)[p.RouteTable].(map[string]any)["properties"].(map[string]any)["routes"].([]any)
	return verifyNetworkChildren(fields["routes"], p.RouteTable, "Microsoft.Network/routeTables", "routes", wanted)
}

func verifyNetworkChildren(raw json.RawMessage, parent, kind, child string, wanted []any) error {
	var actual []placementResource
	if json.Unmarshal(raw, &actual) != nil || len(actual) != len(wanted) {
		return hostedexecution.ErrOwnership
	}
	expected := map[string]any{}
	for _, entry := range wanted {
		rule := entry.(map[string]any)
		expected[rule["name"].(string)] = rule["properties"]
	}
	for _, value := range actual {
		definition, found := expected[value.Name]
		if !found || value.ID != "" && !strings.EqualFold(value.ID, parent+"/"+child+"/"+value.Name) || value.Type != "" && !strings.EqualFold(value.Type, kind+"/"+child) {
			return hostedexecution.ErrOwnership
		}
		var fields map[string]any
		if err := properties(value, &fields); err != nil {
			return err
		}
		if fields["provisioningState"] != "Succeeded" {
			return hostedexecution.ErrUnavailable
		}
		delete(fields, "provisioningState")
		if child == "securityRules" {
			delete(fields, "description")
			if err := normalizeNetworkAddresses(fields); err != nil {
				return err
			}
		} else if raw, found := fields["hasBgpOverride"]; found {
			if raw != false {
				return hostedexecution.ErrOwnership
			}
			delete(fields, "hasBgpOverride")
		}
		if hash(fields) != hash(definition) {
			return hostedexecution.ErrOwnership
		}
		delete(expected, value.Name)
	}
	return nil
}

func normalizeNetworkAddresses(fields map[string]any) error {
	raw, present := fields["destinationAddressPrefixes"]
	if !present {
		return nil
	}
	values, ok := raw.([]any)
	if !ok || len(values) == 0 || len(values) > 8 {
		return hostedexecution.ErrOwnership
	}
	addresses := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		address, ok := value.(string)
		if !ok || seen[address] {
			return hostedexecution.ErrOwnership
		}
		seen[address] = true
		addresses = append(addresses, address)
	}
	slices.Sort(addresses)
	fields["destinationAddressPrefixes"] = addresses
	return nil
}
