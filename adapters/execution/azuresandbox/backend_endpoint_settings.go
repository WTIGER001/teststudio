package azuresandbox

import (
	"encoding/binary"
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type BackendEndpointSettings struct {
	Endpoint          string `json:"endpoint"`
	Interface         string `json:"interface"`
	Subnet            string `json:"subnet"`
	SubnetPrefix      string `json:"subnetPrefix"`
	RelaySubnetPrefix string `json:"relaySubnetPrefix"`
	SecurityGroup     string `json:"securityGroup"`
}

func (e BackendEndpointSettings) validate(c Config, p PlacementSettings, services ServicesSettings) error {
	prefix, valid := privateIPv4Prefix(e.SubnetPrefix)
	source, sourceValid := privateIPv4Prefix(e.RelaySubnetPrefix)
	worker, _ := netip.ParsePrefix(p.Network.Topology.Prefix)
	address, _ := netip.ParseAddr(services.Backend.Address)
	if !valid || !sourceValid || prefix.Bits() > 29 || source.Bits() > 27 || prefix.Overlaps(source) || prefix.Overlaps(worker) || source.Overlaps(worker) || !prefix.Contains(address) || !serviceSubnet(c, e.Subnet) || !resourceID(c, e.Endpoint, "Microsoft.Network", "privateEndpoints") || !resourceID(c, e.Interface, "Microsoft.Network", "networkInterfaces") || !resourceID(c, e.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") {
		return hostedexecution.ErrConfiguration
	}
	parent := strings.Join(strings.Split(services.Relay.Subnet, "/")[:9], "/")
	if !strings.HasPrefix(e.Subnet, parent+"/subnets/") || strings.EqualFold(e.Subnet, services.Relay.Subnet) {
		return hostedexecution.ErrConfiguration
	}
	for _, pair := range [][3]string{{e.Endpoint, p.Private.ExchangeEndpoint, p.Private.RegistryEndpoint}, {e.Interface, p.Private.ExchangeInterface, p.Private.RegistryInterface}, {e.SecurityGroup, p.SecurityGroup, p.Private.SecurityGroup}} {
		if strings.EqualFold(pair[0], pair[1]) || strings.EqualFold(pair[0], pair[2]) {
			return hostedexecution.ErrConfiguration
		}
	}
	start, host := prefix.Addr().As4(), address.As4()
	offset := binary.BigEndian.Uint32(host[:]) - binary.BigEndian.Uint32(start[:])
	if offset < 4 || offset == (uint32(1)<<(32-prefix.Bits()))-1 {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func (e BackendEndpointSettings) rules(s Settings) []any {
	return []any{
		networkRule("relay-backend-https", "Inbound", "Allow", "Tcp", e.RelaySubnetPrefix, s.Placement.Services.Backend.Address, "443", 100),
		networkRule("endpoint-deny-in", "Inbound", "Deny", "*", "*", "*", "*", 4096),
		networkRule("endpoint-deny-out", "Outbound", "Deny", "*", "*", "*", "*", 4096),
	}
}

func (s Settings) BackendEndpointDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	e := s.Placement.Services.BackendEndpoint
	return map[string]any{
		e.SecurityGroup: map[string]any{"location": s.Worker.Region, "properties": map[string]any{"flushConnection": true, "securityRules": e.rules(s)}},
		e.Subnet:        map[string]any{"properties": map[string]any{"addressPrefix": e.SubnetPrefix, "defaultOutboundAccess": false, "networkSecurityGroup": map[string]string{"id": e.SecurityGroup}, "privateEndpointNetworkPolicies": "Enabled", "privateLinkServiceNetworkPolicies": "Enabled"}},
	}, nil
}
