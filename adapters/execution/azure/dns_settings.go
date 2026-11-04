package azureworker

import (
	"net/url"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type DNSSettings struct {
	RegistryDataRegion string           `json:"registryDataRegion"`
	Firewall           FirewallSettings `json:"firewall"`
	Policy             string           `json:"policy"`
	AllowList          string           `json:"allowList"`
	BlockList          string           `json:"blockList"`
}

func (s DNSSettings) canonical(c Config) (DNSSettings, error) {
	var err error
	s.Firewall, err = s.Firewall.canonical(c)
	if err != nil || !region.MatchString(s.RegistryDataRegion) || !c.ownsResource(s.Policy, "Microsoft.Network", "dnsResolverPolicies") || !c.ownsResource(s.AllowList, "Microsoft.Network", "dnsResolverDomainLists") || !c.ownsResource(s.BlockList, "Microsoft.Network", "dnsResolverDomainLists") || strings.EqualFold(s.AllowList, s.BlockList) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func (s DNSSettings) rulePath() string { return s.Policy + "/dnsSecurityRules" }
func (s DNSSettings) linkPath() string { return s.Policy + "/virtualNetworkLinks" }
func (s DNSSettings) vnetPolicyPath() string {
	return s.Firewall.Perimeter.Topology.WorkerVNet + "/listDnsResolverPolicies"
}

func (s DNSSettings) ApprovedDomains(c Config) []string {
	u, err := url.Parse(c.ExchangeOrigin)
	if err != nil || !region.MatchString(s.RegistryDataRegion) {
		return nil
	}
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	domains := append(platformDomains(), u.Hostname(), c.Registry, name+"."+s.RegistryDataRegion+".data.azurecr.io", name+".privatelink.azurecr.io", name+"."+s.RegistryDataRegion+".data.privatelink.azurecr.io")
	slices.Sort(domains)
	return slices.Compact(domains)
}
