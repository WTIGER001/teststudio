package azuresandbox

import (
	"net/netip"
	"slices"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (h HostSettings) virtualNetwork() string {
	return strings.Join(strings.Split(h.Subnet, "/")[:9], "/")
}

func (h HostSettings) validateRegistry(s Settings) error {
	r, p, c := h.Registry, s.Placement, s.Worker
	source, sourceOK := privateIPv4Prefix(h.SubnetPrefix)
	target, targetOK := privateIPv4Prefix(r.SubnetPrefix)
	if !sourceOK || !targetOK || source.Bits() > 27 || target.Bits() > 29 || source.Overlaps(target) || !usablePrivateAddress(target, r.LoginIP) || !usablePrivateAddress(target, r.DataIP) || r.LoginIP == r.DataIP || !serviceSubnet(c, r.Subnet) || !strings.HasPrefix(r.Subnet, h.virtualNetwork()+"/subnets/") || !slices.Contains(p.RegistryConsumers, r.Endpoint) || !zoneID(c, r.Zone, "privatelink.azurecr.io") {
		return hostedexecution.ErrConfiguration
	}
	for _, raw := range []string{p.Network.Topology.Prefix, p.Services.RelayNetwork.Topology.Prefix} {
		prefix, _ := netip.ParsePrefix(raw)
		if source.Overlaps(prefix) || target.Overlaps(prefix) {
			return hostedexecution.ErrConfiguration
		}
	}
	seen := map[string]bool{}
	for _, id := range []string{h.Subnet, p.Subnet, p.Private.Subnet, p.Private.SecurityGroup, p.Private.RegistryEndpoint, p.Private.RegistryInterface, p.Private.ExchangeEndpoint, p.Private.ExchangeInterface, p.Private.RegistryZone, p.Private.ExchangeZone, p.SecurityGroup, p.Services.BackendEndpoint.Endpoint, p.Services.BackendEndpoint.Interface, p.Services.BackendEndpoint.SecurityGroup, p.Services.RelayNetwork.Registry.Endpoint, p.Services.RelayNetwork.Registry.Interface, p.Services.RelayNetwork.Registry.SecurityGroup, p.Services.RelayNetwork.Registry.Zone} {
		seen[strings.ToLower(id)] = true
	}
	for _, entry := range [][2]string{{r.Endpoint, "privateEndpoints"}, {r.Interface, "networkInterfaces"}, {r.SecurityGroup, "networkSecurityGroups"}} {
		if !resourceID(c, entry[0], "Microsoft.Network", entry[1]) {
			return hostedexecution.ErrConfiguration
		}
	}
	for _, id := range []string{r.Endpoint, r.Interface, r.SecurityGroup, r.Subnet, r.Zone} {
		if seen[strings.ToLower(id)] {
			return hostedexecution.ErrConfiguration
		}
		seen[strings.ToLower(id)] = true
	}
	for _, entry := range [][2]string{{r.PullAssignment, "providers/Microsoft.Authorization/roleAssignments/"}, {r.Connection, "privateEndpointConnections/"}} {
		prefix := p.Registry + "/" + entry[1]
		name := strings.TrimPrefix(entry[0], prefix)
		if !strings.HasPrefix(entry[0], prefix) || !identifier.MatchString(name) || strings.Contains(entry[1], "roleAssignments") && (!nonzeroGUID(name) || strings.EqualFold(entry[0], p.PullAssignment) || strings.EqualFold(entry[0], p.Services.RelayNetwork.Registry.PullAssignment)) || entry[0] == p.Services.RelayNetwork.Registry.Connection {
			return hostedexecution.ErrConfiguration
		}
	}
	return nil
}

func (h HostSettings) registryZone(s Settings) azureworker.PrivateZoneBinding {
	name := strings.TrimSuffix(s.Worker.Registry, ".azurecr.io")
	return azureworker.PrivateZoneBinding{ID: h.Registry.Zone, VirtualNetwork: h.virtualNetwork(), LinkName: "controller-network", ResolutionPolicy: true, Records: map[string]string{name: h.Registry.LoginIP, name + "." + s.Worker.Region + ".data": h.Registry.DataIP}}
}

func (h HostSettings) registryRules() []any {
	r := h.Registry
	allow := networkRule("controller-registry-https", "Inbound", "Allow", "Tcp", h.SubnetPrefix, "", "443", 100)
	props := allow["properties"].(map[string]any)
	delete(props, "destinationAddressPrefix")
	ips := []string{r.LoginIP, r.DataIP}
	slices.Sort(ips)
	props["destinationAddressPrefixes"] = ips
	return []any{allow, networkRule("endpoint-deny-in", "Inbound", "Deny", "*", "*", "*", "*", 4096), networkRule("endpoint-deny-out", "Outbound", "Deny", "*", "*", "*", "*", 4096)}
}

func (s Settings) HostRegistryDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	h, r := s.Host, s.Host.Registry
	zone := h.registryZone(s)
	result := map[string]any{
		r.SecurityGroup:  map[string]any{"location": s.Worker.Region, "properties": map[string]any{"flushConnection": true, "securityRules": h.registryRules()}},
		r.Subnet:         map[string]any{"properties": map[string]any{"addressPrefix": r.SubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": r.SecurityGroup}, "privateEndpointNetworkPolicies": "Enabled", "privateLinkServiceNetworkPolicies": "Enabled"}},
		r.PullAssignment: map[string]any{"properties": map[string]any{"principalId": h.Pull.Principal, "principalType": "ServicePrincipal", "scope": s.Placement.Registry, "roleDefinitionId": "/subscriptions/" + s.Worker.Subscription + pullRole}},
		zone.ID:          map[string]any{"location": "global"},
		zone.ID + "/virtualNetworkLinks/" + zone.LinkName: zone.LinkDefinition(),
	}
	for name, record := range zone.RecordDefinitions() {
		result[zone.ID+"/A/"+name] = record
	}
	return result, nil
}
