package azureworker

import (
	"net/netip"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type FirewallSettings struct {
	Perimeter    PerimeterSettings `json:"perimeter"`
	Firewall     string            `json:"firewall"`
	Policy       string            `json:"policy"`
	PublicIP     string            `json:"publicIp"`
	SubnetPrefix string            `json:"subnetPrefix"`
}

func (s FirewallSettings) subnet() string {
	return s.Perimeter.Topology.WorkerVNet + "/subnets/AzureFirewallSubnet"
}
func (s FirewallSettings) group() string { return s.Policy + "/ruleCollectionGroups/worker-platform" }
func (s FirewallSettings) configuration() string {
	return s.Firewall + "/azureFirewallIpConfigurations/egress"
}

func (s FirewallSettings) canonical(c Config) (FirewallSettings, error) {
	var err error
	s.Perimeter, err = s.Perimeter.canonical(c)
	p, valid := privatePrefix(s.SubnetPrefix)
	vnet, _ := netip.ParsePrefix(s.Perimeter.Topology.WorkerVNetPrefix)
	worker, _ := netip.ParsePrefix(s.Perimeter.Topology.WorkerSubnetPrefix)
	ip, _ := netip.ParseAddr(s.Perimeter.FirewallIP)
	if err != nil || !valid || p.Bits() > 26 || p.Bits() < vnet.Bits() || !vnet.Contains(p.Addr()) || p.Overlaps(worker) || !p.Contains(ip) || !c.ownsResource(s.Firewall, "Microsoft.Network", "azureFirewalls") || !c.ownsResource(s.Policy, "Microsoft.Network", "firewallPolicies") || !c.ownsResource(s.PublicIP, "Microsoft.Network", "publicIPAddresses") {
		return s, hostedexecution.ErrConfiguration
	}
	for _, raw := range s.Perimeter.PrivateEndpointIPs {
		address, _ := netip.ParseAddr(raw)
		if p.Contains(address) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	return s, nil
}

func platformDomains() []string {
	return []string{"acs-mirror.azureedge.net", "mcr.microsoft.com", "packages.aks.azure.com", "*.data.mcr.microsoft.com", "*.identity.azure.net", "login.microsoftonline.com", "*.login.microsoftonline.com", "login.microsoft.com", "*.login.microsoft.com"}
}
