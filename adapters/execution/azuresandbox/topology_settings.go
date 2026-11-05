package azuresandbox

import (
	"encoding/binary"
	"net/netip"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type TopologySettings struct {
	Prefix         string `json:"prefix"`
	Firewall       string `json:"firewall"`
	Policy         string `json:"policy"`
	PublicIP       string `json:"publicIp"`
	FirewallPrefix string `json:"firewallPrefix"`
}

func (p PlacementSettings) virtualNetwork() string {
	return strings.Join(strings.Split(p.Subnet, "/")[:9], "/")
}

func (p PlacementSettings) firewallSubnet() string {
	return p.virtualNetwork() + "/subnets/AzureFirewallSubnet"
}

func (t TopologySettings) validate(c Config, p PlacementSettings) error {
	parent, valid := privateIPv4Prefix(t.Prefix)
	firewall, private := privateIPv4Prefix(t.FirewallPrefix)
	worker, _ := netip.ParsePrefix(p.SubnetPrefix)
	endpoints, _ := netip.ParsePrefix(p.Private.SubnetPrefix)
	address, _ := netip.ParseAddr(p.Network.FirewallIP)
	if !valid || !private || firewall.Bits() > 26 || firewall.Overlaps(worker) || firewall.Overlaps(endpoints) || !firewall.Contains(address) || !resourceID(c, t.Firewall, "Microsoft.Network", "azureFirewalls") || !resourceID(c, t.Policy, "Microsoft.Network", "firewallPolicies") || !resourceID(c, t.PublicIP, "Microsoft.Network", "publicIPAddresses") {
		return hostedexecution.ErrConfiguration
	}
	for _, child := range []netip.Prefix{worker, endpoints, firewall} {
		if child.Bits() < parent.Bits() || !parent.Contains(child.Addr()) {
			return hostedexecution.ErrConfiguration
		}
	}
	start, host := firewall.Addr().As4(), address.As4()
	offset := binary.BigEndian.Uint32(host[:]) - binary.BigEndian.Uint32(start[:])
	if offset < 4 || offset == uint32(1)<<(32-firewall.Bits())-1 {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func (p PlacementSettings) firewallBinding() azureworker.FirewallResourceBinding {
	t := p.Network.Topology
	return azureworker.FirewallResourceBinding{Firewall: t.Firewall, Policy: t.Policy, PublicIP: t.PublicIP, Subnet: p.firewallSubnet(), SubnetPrefix: t.FirewallPrefix, Address: p.Network.FirewallIP}
}
