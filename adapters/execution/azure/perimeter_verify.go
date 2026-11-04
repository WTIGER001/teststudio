package azureworker

import (
	"slices"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func perimeterIdentity(id, name, kind, expected, expectedKind, location, region string) bool {
	_, expectedName := resourceParts(expected)
	return strings.EqualFold(id, expected) && name == expectedName && strings.EqualFold(kind, expectedKind) && location == region
}

func soleSubnet(subnets []*n.Subnet, id string) bool {
	return len(subnets) == 1 && subnets[0] != nil && strings.EqualFold(value(subnets[0].ID), id)
}

func (g *PerimeterGuard) verifyGroup(actual n.SecurityGroup) error {
	expectedID := g.settings.Topology.SecurityGroup
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), expectedID, "Microsoft.Network/networkSecurityGroups", value(actual.Location), g.worker.Region) || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if !value(p.FlushConnection) || len(p.NetworkInterfaces) != 0 || !soleSubnet(p.Subnets, g.settings.Topology.WorkerSubnet) {
		return hostedexecution.ErrOwnership
	}
	wanted := g.settings.rules()
	if len(p.SecurityRules) != len(wanted) {
		return hostedexecution.ErrOwnership
	}
	expected := map[string]*n.SecurityRule{}
	for _, rule := range wanted {
		expected[value(rule.Name)] = rule
	}
	for _, actual := range p.SecurityRules {
		if actual == nil || actual.Properties == nil {
			return hostedexecution.ErrOwnership
		}
		name := value(actual.Name)
		rule, ok := expected[name]
		if !ok || actual.ID != nil && !strings.EqualFold(value(actual.ID), expectedID+"/securityRules/"+name) || actual.Type != nil && !strings.EqualFold(value(actual.Type), "Microsoft.Network/networkSecurityGroups/securityRules") {
			return hostedexecution.ErrOwnership
		}
		if value(actual.Properties.ProvisioningState) != n.ProvisioningStateSucceeded {
			return hostedexecution.ErrUnavailable
		}
		fields := document(actual.Properties)
		if raw, ok := fields["destinationAddressPrefixes"]; ok {
			entries, ok := raw.([]any)
			if !ok {
				return hostedexecution.ErrOwnership
			}
			addresses := make([]string, 0, len(entries))
			seen := map[string]bool{}
			for _, entry := range entries {
				address, ok := entry.(string)
				if !ok || seen[address] {
					return hostedexecution.ErrOwnership
				}
				seen[address] = true
				addresses = append(addresses, address)
			}
			slices.Sort(addresses)
			fields["destinationAddressPrefixes"] = addresses
		}
		delete(fields, "provisioningState")
		delete(fields, "description")
		if fingerprint(fields) != fingerprint(document(rule.Properties)) {
			return hostedexecution.ErrOwnership
		}
		delete(expected, name)
	}
	return nil
}

func (g *PerimeterGuard) verifyRoutes(actual n.RouteTable) error {
	expectedID := g.settings.Topology.RouteTable
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), expectedID, "Microsoft.Network/routeTables", value(actual.Location), g.worker.Region) || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if !value(p.DisableBgpRoutePropagation) || !soleSubnet(p.Subnets, g.settings.Topology.WorkerSubnet) || len(p.Routes) != 1 || p.Routes[0] == nil || p.Routes[0].Properties == nil {
		return hostedexecution.ErrOwnership
	}
	route := p.Routes[0]
	if value(route.Name) != "worker-egress" || route.ID != nil && !strings.EqualFold(value(route.ID), expectedID+"/routes/worker-egress") || route.Type != nil && !strings.EqualFold(value(route.Type), "Microsoft.Network/routeTables/routes") || value(route.Properties.HasBgpOverride) {
		return hostedexecution.ErrOwnership
	}
	if value(route.Properties.ProvisioningState) != n.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	expected, _ := g.settings.RouteTableDefinition(g.worker)
	fields := document(route.Properties)
	delete(fields, "provisioningState")
	delete(fields, "hasBgpOverride")
	if fingerprint(fields) != fingerprint(document(expected.Properties.Routes[0].Properties)) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
