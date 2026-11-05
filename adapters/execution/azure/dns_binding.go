package azureworker

type dnsBinding struct {
	Policy, AllowList, BlockList, VirtualNetwork, Role string
	Domains                                            []string
}

func (s dnsBinding) rulePath() string { return s.Policy + "/dnsSecurityRules" }
func (s dnsBinding) linkPath() string { return s.Policy + "/virtualNetworkLinks" }
func (s dnsBinding) vnetPolicyPath() string {
	return s.VirtualNetwork + "/listDnsResolverPolicies"
}

func (s DNSSettings) binding(c Config) dnsBinding {
	return dnsBinding{Policy: s.Policy, AllowList: s.AllowList, BlockList: s.BlockList, VirtualNetwork: s.Firewall.Perimeter.Topology.WorkerVNet, Role: "worker", Domains: s.ApprovedDomains(c)}
}
