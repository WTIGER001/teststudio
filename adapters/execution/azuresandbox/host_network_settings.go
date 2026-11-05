package azuresandbox

import (
	"net/netip"
	"net/url"
	"slices"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type HostNetworkSettings struct {
	Topology      TopologySettings `json:"topology"`
	FirewallIP    string           `json:"firewallIp"`
	SecurityGroup string           `json:"securityGroup"`
	RouteTable    string           `json:"routeTable"`
	Resolver      ResolverSettings `json:"resolver"`
}

func (h HostSettings) validateNetwork(s Settings) error {
	r, c := h.Network, s.Worker
	parent, parentOK := privateIPv4Prefix(r.Topology.Prefix)
	firewall, firewallOK := privateIPv4Prefix(r.Topology.FirewallPrefix)
	if !parentOK || !firewallOK || firewall.Bits() > 26 || !usablePrivateAddress(firewall, r.FirewallIP) || r.Resolver.validate(c) != nil {
		return hostedexecution.ErrConfiguration
	}
	prefixes := []netip.Prefix{firewall}
	for _, raw := range []string{h.SubnetPrefix, h.Registry.SubnetPrefix, h.Database.SubnetPrefix} {
		prefix, _ := netip.ParsePrefix(raw)
		prefixes = append(prefixes, prefix)
	}
	for i, prefix := range prefixes {
		if prefix.Bits() < parent.Bits() || !parent.Contains(prefix.Addr()) {
			return hostedexecution.ErrConfiguration
		}
		for _, other := range prefixes[:i] {
			if prefix.Overlaps(other) {
				return hostedexecution.ErrConfiguration
			}
		}
	}
	for _, raw := range []string{s.Placement.Network.Topology.Prefix, s.Placement.Services.RelayNetwork.Topology.Prefix} {
		other, _ := netip.ParsePrefix(raw)
		if parent.Overlaps(other) {
			return hostedexecution.ErrConfiguration
		}
	}
	seen := map[string]bool{}
	p, relay := s.Placement, s.Placement.Services.RelayNetwork
	for _, id := range []string{h.Registry.SecurityGroup, p.SecurityGroup, p.RouteTable, p.Private.SecurityGroup, p.Network.Topology.Firewall, p.Network.Topology.Policy, p.Network.Topology.PublicIP, p.Network.Resolver.Policy, p.Network.Resolver.AllowList, p.Network.Resolver.BlockList, relay.SecurityGroup, relay.RouteTable, relay.Registry.SecurityGroup, relay.Topology.Firewall, relay.Topology.Policy, relay.Topology.PublicIP, relay.Resolver.Policy, relay.Resolver.AllowList, relay.Resolver.BlockList, p.Services.BackendEndpoint.SecurityGroup} {
		seen[strings.ToLower(id)] = true
	}
	for _, entry := range [][2]string{{r.SecurityGroup, "networkSecurityGroups"}, {r.RouteTable, "routeTables"}, {r.Topology.Firewall, "azureFirewalls"}, {r.Topology.Policy, "firewallPolicies"}, {r.Topology.PublicIP, "publicIPAddresses"}, {r.Resolver.Policy, "dnsResolverPolicies"}, {r.Resolver.AllowList, "dnsResolverDomainLists"}, {r.Resolver.BlockList, "dnsResolverDomainLists"}} {
		if !resourceID(c, entry[0], "Microsoft.Network", entry[1]) || seen[strings.ToLower(entry[0])] {
			return hostedexecution.ErrConfiguration
		}
		seen[strings.ToLower(entry[0])] = true
	}
	return nil
}

func (h HostSettings) dependencyDomains(s Settings) []string {
	result := append(azureworker.PlatformDomains(), "management.azure.com", strings.TrimPrefix(s.Worker.endpoint(), "https://"))
	for _, raw := range []string{h.DatabaseSecretURL, h.ConfigurationSecretURL, h.BlobServiceURL} {
		if raw != "" {
			u, _ := url.Parse(raw)
			result = append(result, u.Hostname())
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func (h HostSettings) resolverBinding(s Settings) azureworker.ResolverBinding {
	name := strings.TrimSuffix(s.Worker.Registry, ".azurecr.io")
	domains := append(h.dependencyDomains(s), h.DatabaseHost, h.Database.name()+"."+postgresPrivateZone, s.Worker.Registry, name+"."+s.Worker.Region+".data.azurecr.io", name+".privatelink.azurecr.io", name+"."+s.Worker.Region+".data.privatelink.azurecr.io")
	slices.Sort(domains)
	return azureworker.ResolverBinding{Policy: h.Network.Resolver.Policy, AllowList: h.Network.Resolver.AllowList, BlockList: h.Network.Resolver.BlockList, VirtualNetwork: h.virtualNetwork(), Role: "controller", Domains: domains}
}

func (h HostSettings) firewallBinding() azureworker.FirewallResourceBinding {
	r := h.Network
	return azureworker.FirewallResourceBinding{Firewall: r.Topology.Firewall, Policy: r.Topology.Policy, PublicIP: r.Topology.PublicIP, Subnet: h.virtualNetwork() + "/subnets/AzureFirewallSubnet", SubnetPrefix: r.Topology.FirewallPrefix, Address: r.FirewallIP}
}
