package azureworker

import (
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

type FirewallResourceBinding struct {
	Firewall, Policy, PublicIP, Subnet, SubnetPrefix, Address string
}

type FirewallObservation struct {
	Firewall n.AzureFirewall
	Policy   n.FirewallPolicy
	PublicIP n.PublicIPAddress
	Subnet   n.Subnet
	Groups   []n.FirewallPolicyRuleCollectionGroup
}

func (b FirewallResourceBinding) binding() firewallBinding {
	return firewallBinding{Firewall: b.Firewall, Policy: b.Policy, PublicIP: b.PublicIP, Subnet: b.Subnet, SubnetPrefix: b.SubnetPrefix, Address: b.Address}
}

func (b FirewallResourceBinding) FirewallDefinition(region string) n.AzureFirewall {
	return b.binding().firewallDefinition(region)
}

func (b FirewallResourceBinding) PolicyDefinition(region string) n.FirewallPolicy {
	return b.binding().policyDefinition(region)
}

func (b FirewallResourceBinding) Verify(actual FirewallObservation, region, groupName string, group n.FirewallPolicyRuleCollectionGroup) error {
	checks := []func() error{
		func() error { return b.binding().verifyFirewall(actual.Firewall, region) },
		func() error { return b.binding().verifySubnet(actual.Subnet) },
		func() error { return b.binding().verifyPublicIP(actual.PublicIP, region) },
		func() error {
			return b.binding().verifyPolicyDefinition(actual.Policy, actual.Groups, region, groupName, group)
		},
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}
