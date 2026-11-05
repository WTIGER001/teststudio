package azureworker

type firewallBinding struct {
	Firewall, Policy, PublicIP, Subnet, SubnetPrefix string
	Address, SourcePrefix, Role                      string
}

func (s firewallBinding) subnet() string { return s.Subnet }
func (s firewallBinding) group() string {
	return s.Policy + "/ruleCollectionGroups/" + s.Role + "-platform"
}
func (s firewallBinding) configuration() string {
	return s.Firewall + "/azureFirewallIpConfigurations/egress"
}

func (s FirewallSettings) binding() firewallBinding {
	return firewallBinding{Firewall: s.Firewall, Policy: s.Policy, PublicIP: s.PublicIP, Subnet: s.subnet(), SubnetPrefix: s.SubnetPrefix, Address: s.Perimeter.FirewallIP, SourcePrefix: s.Perimeter.Topology.WorkerSubnetPrefix, Role: "worker"}
}
