package azuresandbox

import (
	"encoding/binary"
	"net/netip"
	"net/url"
	"slices"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PrivateSettings struct {
	Subnet              string `json:"subnet"`
	SubnetPrefix        string `json:"subnetPrefix"`
	SecurityGroup       string `json:"securityGroup"`
	RegistryEndpoint    string `json:"registryEndpoint"`
	RegistryInterface   string `json:"registryInterface"`
	ExchangeEndpoint    string `json:"exchangeEndpoint"`
	ExchangeInterface   string `json:"exchangeInterface"`
	ExchangeEnvironment string `json:"exchangeEnvironment"`
	RegistryZone        string `json:"registryZone"`
	ExchangeZone        string `json:"exchangeZone"`
	RegistryIP          string `json:"registryIp"`
	RegistryDataIP      string `json:"registryDataIp"`
	ExchangeIP          string `json:"exchangeIp"`
}

func zoneID(c Config, id, host string) bool {
	parts := strings.Split(id, "/")
	return len(parts) == 9 && parts[8] == host && resourceID(c, strings.Join(parts[:8], "/")+"/zone", "Microsoft.Network", "privateDnsZones")
}

func (p PrivateSettings) validate(c Config, placement PlacementSettings) error {
	worker, _ := netip.ParsePrefix(placement.SubnetPrefix)
	prefix, private := privateIPv4Prefix(p.SubnetPrefix)
	vnet := strings.TrimSuffix(placement.Subnet, "/subnets/"+strings.Split(placement.Subnet, "/")[10])
	subnet := strings.TrimPrefix(p.Subnet, vnet+"/subnets/")
	u, _ := url.Parse(c.ExchangeOrigin)
	if !private || prefix.Bits() > 29 || prefix.Overlaps(worker) || !identifier.MatchString(subnet) || p.Subnet != vnet+"/subnets/"+subnet || p.Subnet == placement.Subnet || !resourceID(c, p.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") || p.SecurityGroup == placement.SecurityGroup || !resourceID(c, p.ExchangeEnvironment, "Microsoft.App", "managedEnvironments") || !zoneID(c, p.RegistryZone, "privatelink.azurecr.io") || !zoneID(c, p.ExchangeZone, u.Hostname()) || strings.HasSuffix(u.Hostname(), ".azurecr.io") {
		return hostedexecution.ErrConfiguration
	}
	for _, pair := range [][3]string{{p.RegistryEndpoint, p.ExchangeEndpoint, "privateEndpoints"}, {p.RegistryInterface, p.ExchangeInterface, "networkInterfaces"}} {
		if !resourceID(c, pair[0], "Microsoft.Network", pair[2]) || !resourceID(c, pair[1], "Microsoft.Network", pair[2]) || pair[0] == pair[1] {
			return hostedexecution.ErrConfiguration
		}
	}
	ips := []string{p.RegistryIP, p.RegistryDataIP, p.ExchangeIP}
	start := prefix.Addr().As4()
	base := binary.BigEndian.Uint32(start[:])
	for _, raw := range ips {
		ip, err := netip.ParseAddr(raw)
		if err != nil || !ip.Is4() || !prefix.Contains(ip) {
			return hostedexecution.ErrConfiguration
		}
		bytes := ip.As4()
		offset := binary.BigEndian.Uint32(bytes[:]) - base
		if offset < 4 || offset == (uint32(1)<<(32-prefix.Bits()))-1 {
			return hostedexecution.ErrConfiguration
		}
	}
	slices.Sort(ips)
	if !slices.Equal(ips, placement.Network.PrivateEndpointIPs) {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func (p PrivateSettings) endpointBindings(s Settings) []azureworker.PrivateEndpointBinding {
	return []azureworker.PrivateEndpointBinding{
		{Endpoint: p.RegistryEndpoint, Interface: p.RegistryInterface, Service: s.Placement.Registry, Group: "registry", Subnet: p.Subnet, Region: s.Worker.Region, Members: map[string]string{"registry": p.RegistryIP, "registry_data_" + s.Worker.Region: p.RegistryDataIP}},
		{Endpoint: p.ExchangeEndpoint, Interface: p.ExchangeInterface, Service: p.ExchangeEnvironment, Group: "managedEnvironment", Subnet: p.Subnet, Region: s.Worker.Region, Members: map[string]string{"managedEnvironment": p.ExchangeIP}},
	}
}

func (p PrivateSettings) zoneBindings(s Settings) []azureworker.PrivateZoneBinding {
	vnet := strings.Join(strings.Split(p.Subnet, "/")[:9], "/")
	name := strings.TrimSuffix(s.Worker.Registry, ".azurecr.io")
	return []azureworker.PrivateZoneBinding{
		{ID: p.ExchangeZone, VirtualNetwork: vnet, LinkName: "sandbox-network", Records: map[string]string{"@": p.ExchangeIP}},
		{ID: p.RegistryZone, VirtualNetwork: vnet, LinkName: "sandbox-network", ResolutionPolicy: true, Records: map[string]string{name: p.RegistryIP, name + "." + s.Worker.Region + ".data": p.RegistryDataIP}},
	}
}
