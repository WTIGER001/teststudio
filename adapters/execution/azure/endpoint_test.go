package azureworker

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureEndpointsVerifyFreshAddressOwnershipAndPriorStages(t *testing.T) {
	c, s, f, g := endpointTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 6 {
		t.Fatal("valid endpoint ownership rejected", err, f.reads)
	}
	s.PrivateDNS.DNS.Firewall.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			if err := g.Check(t.Context(), c); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.reads != 24 {
		t.Fatal("endpoint inspection reused stale state", f.reads)
	}
	slices.Reverse(f.interfaces[s.RegistryNIC].Properties.IPConfigurations)
	slices.Reverse(f.subnet.Properties.IPConfigurations)
	slices.Reverse(f.security.Properties.SecurityRules)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("resource order changed ownership", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("endpoint inspection became full network authority")
	}
	before := f.reads
	previous := g.dns.reader.(*SDKPrivateDNSReader)
	wire := previous.http.base.(*privateDNSFixture)
	wire.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("unverified DNS permitted endpoint observation")
	}
	wire.mode = ""
	c.NetworkProfile = "changed"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("configuration change reached endpoint resources")
	}
}

func TestAzureEndpointsRejectForeignServicesAddressesAndPolicyBypass(t *testing.T) {
	cases := map[string]func(*endpointFixture){
		"foreign-registry": func(f *endpointFixture) {
			f.endpoints[f.settings.RegistryEndpoint].Properties.PrivateLinkServiceConnections[0].Properties.PrivateLinkServiceID = to.Ptr(f.settings.ExchangeEnvironment)
		},
		"wrong-group": func(f *endpointFixture) {
			f.endpoints[f.settings.ExchangeEndpoint].Properties.PrivateLinkServiceConnections[0].Properties.GroupIDs = []*string{to.Ptr("sites")}
		},
		"rejected-connection": func(f *endpointFixture) {
			f.endpoints[f.settings.ExchangeEndpoint].Properties.PrivateLinkServiceConnections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr("Rejected")
		},
		"second-connection": func(f *endpointFixture) {
			p := f.endpoints[f.settings.RegistryEndpoint].Properties
			p.PrivateLinkServiceConnections = append(p.PrivateLinkServiceConnections, clone(p.PrivateLinkServiceConnections[0]))
		},
		"manual-connection": func(f *endpointFixture) {
			p := f.endpoints[f.settings.ExchangeEndpoint].Properties
			p.ManualPrivateLinkServiceConnections = p.PrivateLinkServiceConnections
		},
		"wrong-nic": func(f *endpointFixture) {
			f.endpoints[f.settings.ExchangeEndpoint].Properties.NetworkInterfaces[0].ID = to.Ptr(f.settings.RegistryNIC)
		},
		"wrong-parent": func(f *endpointFixture) {
			f.interfaces[f.settings.ExchangeNIC].Properties.PrivateEndpoint.ID = to.Ptr(f.settings.RegistryEndpoint)
		},
		"swapped-registry-addresses": func(f *endpointFixture) {
			ips := f.interfaces[f.settings.RegistryNIC].Properties.IPConfigurations
			ips[0].Properties.PrivateIPAddress, ips[1].Properties.PrivateIPAddress = ips[1].Properties.PrivateIPAddress, ips[0].Properties.PrivateIPAddress
		},
		"wrong-data-region": func(f *endpointFixture) {
			f.interfaces[f.settings.RegistryNIC].Properties.IPConfigurations[1].Properties.PrivateLinkConnectionProperties.RequiredMemberName = to.Ptr("registry_data_eastus2")
		},
		"extra-nic-address": func(f *endpointFixture) {
			p := f.interfaces[f.settings.ExchangeNIC].Properties
			p.IPConfigurations = append(p.IPConfigurations, clone(p.IPConfigurations[0]))
		},
		"public-nic": func(f *endpointFixture) {
			f.interfaces[f.settings.ExchangeNIC].Properties.IPConfigurations[0].Properties.PublicIPAddress = &n.PublicIPAddress{ID: to.Ptr("foreign")}
		},
		"forwarding": func(f *endpointFixture) {
			f.interfaces[f.settings.ExchangeNIC].Properties.EnableIPForwarding = to.Ptr(true)
		},
		"unrelated-vm": func(f *endpointFixture) {
			f.interfaces[f.settings.ExchangeNIC].Properties.VirtualMachine = &n.SubResource{ID: to.Ptr("foreign")}
		},
		"ip-subnet": func(f *endpointFixture) {
			f.interfaces[f.settings.RegistryNIC].Properties.IPConfigurations[0].Properties.Subnet.ID = to.Ptr(f.settings.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerSubnet)
		},
		"disabled-nsg": func(f *endpointFixture) {
			f.subnet.Properties.PrivateEndpointNetworkPolicies = to.Ptr(n.VirtualNetworkPrivateEndpointNetworkPoliciesRouteTableEnabled)
		},
		"nat":                func(f *endpointFixture) { f.subnet.Properties.NatGateway = &n.SubResource{ID: to.Ptr("foreign")} },
		"foreign-subnet-nic": func(f *endpointFixture) { f.subnet.Properties.IPConfigurations[0].ID = to.Ptr("foreign") },
		"extra-endpoint": func(f *endpointFixture) {
			f.subnet.Properties.PrivateEndpoints = append(f.subnet.Properties.PrivateEndpoints, &n.PrivateEndpoint{ID: to.Ptr("foreign")})
		},
		"foreign-nsg-attachment": func(f *endpointFixture) {
			f.security.Properties.Subnets = append(f.security.Properties.Subnets, &n.Subnet{ID: to.Ptr("foreign")})
		},
		"permissive-https-source": func(f *endpointFixture) {
			f.security.Properties.SecurityRules[0].Properties.SourceAddressPrefix = to.Ptr("VirtualNetwork")
		},
		"extra-nsg-rule": func(f *endpointFixture) {
			f.security.Properties.SecurityRules = append(f.security.Properties.SecurityRules, perimeterRule("foreign", n.SecurityRuleDirectionInbound, 90, n.SecurityRuleProtocolAsterisk, n.SecurityRuleAccessAllow, "*", "*", "*"))
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := endpointTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureEndpointsPreservePendingAndUnavailableResources(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "redirect", "oversized", "pending-endpoint", "pending-approval", "approval-action", "pending-nic", "unknown-member", "missing-subnet-endpoint", "missing-subnet-ip", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			c, s, f, g := endpointTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "pending-endpoint":
				f.endpoints[s.ExchangeEndpoint].Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "pending-approval":
				f.endpoints[s.ExchangeEndpoint].Properties.PrivateLinkServiceConnections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr("Pending")
			case "approval-action":
				f.endpoints[s.ExchangeEndpoint].Properties.PrivateLinkServiceConnections[0].Properties.PrivateLinkServiceConnectionState.ActionsRequired = to.Ptr("Recreate")
			case "pending-nic":
				f.interfaces[s.ExchangeNIC].Properties.ProvisioningState = to.Ptr(n.ProvisioningStateUpdating)
			case "unknown-member":
				f.interfaces[s.ExchangeNIC].Properties.IPConfigurations[0].Properties.PrivateLinkConnectionProperties = nil
			case "missing-subnet-endpoint":
				f.subnet.Properties.PrivateEndpoints = nil
			case "missing-subnet-ip":
				f.subnet.Properties.IPConfigurations = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			requireDeadlineError(t, g.Check(ctx, c), hostedexecution.ErrUnavailable)
			if f.reads > 6 {
				t.Fatal("endpoint reads retried", f.reads)
			}
		})
	}
}

func TestAzureEndpointsBindConfigurationAndReturnFreshDefinitions(t *testing.T) {
	c, s, _, g := endpointTest(t)
	for name, mutate := range map[string]func(*EndpointSettings){
		"overlap-worker": func(s *EndpointSettings) {
			s.SubnetPrefix = s.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerSubnetPrefix
		},
		"overlap-firewall": func(s *EndpointSettings) { s.SubnetPrefix = s.PrivateDNS.DNS.Firewall.SubnetPrefix },
		"outside-vnet":     func(s *EndpointSettings) { s.SubnetPrefix = "10.99.0.0/24" },
		"reserved-address": func(s *EndpointSettings) { s.SubnetPrefix = "10.20.2.4/30" },
		"foreign-subnet":   func(s *EndpointSettings) { s.Subnet = s.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerSubnet },
		"shared-nsg": func(s *EndpointSettings) {
			s.SecurityGroup = s.PrivateDNS.DNS.Firewall.Perimeter.Topology.SecurityGroup
		},
		"shared-nic":      func(s *EndpointSettings) { s.ExchangeNIC = s.RegistryNIC },
		"shared-endpoint": func(s *EndpointSettings) { s.ExchangeEndpoint = s.RegistryEndpoint },
		"foreign-registry": func(s *EndpointSettings) {
			s.Registry = strings.TrimSuffix(s.Registry, "/"+strings.TrimSuffix(c.Registry, ".azurecr.io")) + "/otherregistry"
		},
		"controller-exchange": func(s *EndpointSettings) { s.ExchangeEnvironment = c.ControllerEnvironment },
	} {
		t.Run(name, func(t *testing.T) {
			changed := s
			mutate(&changed)
			_, err := NewEndpointGuard(c, changed, g.dns, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	subnet, err := s.SubnetDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	subnet.Properties.NetworkSecurityGroup.ID = to.Ptr("foreign")
	security, err := s.SecurityGroupDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	security.Properties.SecurityRules[0].Properties.DestinationAddressPrefixes[0] = to.Ptr("8.8.8.8")
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("definition mutation changed guard", err)
	}
}

func TestAzureEndpointsExcludeAzureReservedSubnetAddresses(t *testing.T) {
	c, s, _, _ := endpointTest(t)
	s.SubnetPrefix = "10.20.2.0/29"
	if _, err := s.canonical(c); err != nil {
		t.Fatal("three usable addresses in a /29 rejected", err)
	}
	for _, ip := range []string{"10.20.2.0", "10.20.2.1", "10.20.2.2", "10.20.2.3", "10.20.2.7"} {
		t.Run(ip, func(t *testing.T) {
			changed := clone(s)
			changed.PrivateDNS.ExchangeIP = ip
			changed.PrivateDNS.DNS.Firewall.Perimeter.PrivateEndpointIPs = []string{ip, s.PrivateDNS.RegistryIP, s.PrivateDNS.DataIP}
			_, err := changed.canonical(c)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
}

func TestAzureEndpointsAcceptInactiveNICDefaultsAndRejectExtensions(t *testing.T) {
	c, s, f, g := endpointTest(t)
	p := f.interfaces[s.ExchangeNIC].Properties
	p.AuxiliaryMode = to.Ptr(n.NetworkInterfaceAuxiliaryModeNone)
	p.AuxiliarySKU = to.Ptr(n.NetworkInterfaceAuxiliarySKUNone)
	p.MigrationPhase = to.Ptr(n.NetworkInterfaceMigrationPhaseNone)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("inactive provider defaults changed ownership", err)
	}
	p.AuxiliaryMode = to.Ptr(n.NetworkInterfaceAuxiliaryModeFloating)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
}
