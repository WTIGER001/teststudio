package azureworker

import (
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayDNSSettings struct {
	Firewall  RelayFirewallSettings `json:"firewall"`
	Policy    string                `json:"policy"`
	AllowList string                `json:"allowList"`
	BlockList string                `json:"blockList"`
}

func (s RelayDNSSettings) canonical(c Config) (RelayDNSSettings, error) {
	var err error
	s.Firewall, err = s.Firewall.canonical(c)
	if err != nil || !c.ownsResource(s.Policy, "Microsoft.Network", "dnsResolverPolicies") || !c.ownsResource(s.AllowList, "Microsoft.Network", "dnsResolverDomainLists") || !c.ownsResource(s.BlockList, "Microsoft.Network", "dnsResolverDomainLists") || strings.EqualFold(s.AllowList, s.BlockList) {
		return s, hostedexecution.ErrConfiguration
	}
	worker := s.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS
	if strings.EqualFold(s.Policy, worker.Policy) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, selected := range []string{s.AllowList, s.BlockList} {
		if strings.EqualFold(selected, worker.AllowList) || strings.EqualFold(selected, worker.BlockList) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	return s, nil
}

func (s RelayDNSSettings) ApprovedDomains(c Config) []string {
	registry := s.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry
	if Validate(c) != nil || !region.MatchString(registry.region()) {
		return nil
	}
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	domains := append(platformDomains(), c.Registry, name+"."+registry.region()+".data.azurecr.io", name+".privatelink.azurecr.io", name+"."+registry.region()+".data.privatelink.azurecr.io")
	slices.Sort(domains)
	return slices.Compact(domains)
}

func (s RelayDNSSettings) binding(c Config) dnsBinding {
	return dnsBinding{Policy: s.Policy, AllowList: s.AllowList, BlockList: s.BlockList, VirtualNetwork: s.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork, Role: "relay", Domains: s.ApprovedDomains(c)}
}
