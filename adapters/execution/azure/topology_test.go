package azureworker

import (
	"context"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureTopologyChecksActualBindingsAndRefreshesEveryInspection(t *testing.T) {
	c, _, f, g := topologyTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 6 {
		t.Fatal("valid private topology was rejected", err)
	}
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
		t.Fatal("inspection reused stale or unbounded reads", f.reads)
	}
	f.wnet.Properties.VirtualNetworkPeerings = []*n.VirtualNetworkPeering{{}}
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	c.NetworkProfile = "changed"
	reads := f.reads
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != reads {
		t.Fatal("changed configuration reached Azure")
	}
}

func TestAzureTopologyRejectsEnvironmentBoundaryDrift(t *testing.T) {
	cases := map[string]func(*topologyFixture){
		"controller-subnet": func(f *topologyFixture) {
			f.worker.Properties.VnetConfiguration.InfrastructureSubnetID = f.controller.Properties.VnetConfiguration.InfrastructureSubnetID
		},
		"public-access": func(f *topologyFixture) {
			f.worker.Properties.PublicNetworkAccess = to.Ptr(a.PublicNetworkAccessEnabled)
		},
		"external-balancer": func(f *topologyFixture) { f.worker.Properties.VnetConfiguration.Internal = to.Ptr(false) },
		"private-entrypoint": func(f *topologyFixture) {
			f.worker.Properties.PrivateEndpointConnections = []*a.PrivateEndpointConnection{{}}
		},
		"legacy-environment": func(f *topologyFixture) { f.worker.Properties.WorkloadProfiles = nil },
		"dedicated-profile":  func(f *topologyFixture) { f.worker.Properties.WorkloadProfiles[0].WorkloadProfileType = to.Ptr("D4") },
		"extra-profile": func(f *topologyFixture) {
			f.worker.Properties.WorkloadProfiles = append(f.worker.Properties.WorkloadProfiles, &a.WorkloadProfile{})
		},
		"shared-infrastructure": func(f *topologyFixture) {
			f.controller.Properties.InfrastructureResourceGroup = f.worker.Properties.InfrastructureResourceGroup
		},
		"runtime-identity": func(f *topologyFixture) {
			f.worker.Identity = &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeSystemAssigned)}
		},
		"telemetry-credential": func(f *topologyFixture) {
			f.worker.Properties.DaprAIConnectionString = to.Ptr("private-provider-canary")
		},
		"external-logs": func(f *topologyFixture) {
			f.worker.Properties.AppLogsConfiguration.Destination = to.Ptr("azure-monitor")
		},
		"foreign-region":        func(f *topologyFixture) { f.worker.Location = to.Ptr("westus") },
		"wrong-name":            func(f *topologyFixture) { f.worker.Name = to.Ptr("foreign") },
		"out-of-subnet-address": func(f *topologyFixture) { f.worker.Properties.StaticIP = to.Ptr("10.21.0.4") },
		"ipv6":                  func(f *topologyFixture) { f.worker.Properties.StaticIP = to.Ptr("fd00::4") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := topologyTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureTopologyRejectsVNetAndSubnetBypasses(t *testing.T) {
	cases := map[string]func(*topologyFixture){
		"controller-partly-public": func(f *topologyFixture) {
			f.cnet.Properties.AddressSpace.AddressPrefixes = []*string{to.Ptr("192.168.0.0/15")}
		},
		"controller-overlap": func(f *topologyFixture) {
			f.cnet.Properties.AddressSpace.AddressPrefixes = []*string{to.Ptr("10.20.0.0/16")}
		},
		"worker-ipv6": func(f *topologyFixture) {
			f.wnet.Properties.AddressSpace.AddressPrefixes = append(f.wnet.Properties.AddressSpace.AddressPrefixes, to.Ptr("fd00::/64"))
		},
		"peering": func(f *topologyFixture) { f.wnet.Properties.VirtualNetworkPeerings = []*n.VirtualNetworkPeering{{}} },
		"custom-dns": func(f *topologyFixture) {
			f.wnet.Properties.DhcpOptions = &n.DhcpOptions{DNSServers: []*string{to.Ptr("8.8.8.8")}}
		},
		"public-nat":       func(f *topologyFixture) { f.wnet.Properties.DefaultPublicNatGateway = &n.SubResource{} },
		"subnet-nat":       func(f *topologyFixture) { f.wsub.Properties.NatGateway = &n.SubResource{} },
		"default-outbound": func(f *topologyFixture) { f.wsub.Properties.DefaultOutboundAccess = to.Ptr(true) },
		"unknown-outbound": func(f *topologyFixture) { f.wsub.Properties.DefaultOutboundAccess = nil },
		"service-endpoint": func(f *topologyFixture) {
			f.wsub.Properties.ServiceEndpoints = []*n.ServiceEndpointPropertiesFormat{{}}
		},
		"endpoint-policy":             func(f *topologyFixture) { f.wsub.Properties.ServiceEndpointPolicies = []*n.ServiceEndpointPolicy{{}} },
		"private-endpoint-in-workers": func(f *topologyFixture) { f.wsub.Properties.PrivateEndpoints = []*n.PrivateEndpoint{{}} },
		"tenant-sharing":              func(f *topologyFixture) { f.wsub.Properties.SharingScope = to.Ptr(n.SharingScopeTenant) },
		"foreign-delegation": func(f *topologyFixture) {
			f.wsub.Properties.Delegations[0].Properties.ServiceName = to.Ptr("Microsoft.Web/serverFarms")
		},
		"missing-group":      func(f *topologyFixture) { f.wsub.Properties.NetworkSecurityGroup = nil },
		"wrong-route":        func(f *topologyFixture) { f.wsub.Properties.RouteTable.ID = to.Ptr("foreign") },
		"prefix-drift":       func(f *topologyFixture) { f.wsub.Properties.AddressPrefix = to.Ptr("10.20.1.0/24") },
		"conflicting-prefix": func(f *topologyFixture) { f.wsub.Properties.AddressPrefixes = []*string{to.Ptr("10.20.0.0/24")} },
		"duplicate-subnet": func(f *topologyFixture) {
			f.wnet.Properties.Subnets = append(f.wnet.Properties.Subnets, f.wnet.Properties.Subnets[0])
		},
		"stale-subnet": func(f *topologyFixture) {
			f.detachSubnet()
			f.wsub.Properties.DefaultOutboundAccess = to.Ptr(false)
			f.wnet.Properties.Subnets[0].Properties.DefaultOutboundAccess = to.Ptr(true)
		},
		"controller-subnet-outside-vnet": func(f *topologyFixture) { f.csub.Properties.AddressPrefix = to.Ptr("10.22.0.0/24") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := topologyTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureTopologyPreservesReadAndProvisioningUncertainty(t *testing.T) {
	for _, mode := range []string{"unavailable", "redirect", "oversized", "missing", "provisioning", "no-address", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			c, _, f, g := topologyTest(t)
			f.mode = mode
			ctx := t.Context()
			if mode == "provisioning" {
				f.worker.Properties.ProvisioningState = to.Ptr(a.EnvironmentProvisioningStateInfrastructureSetupInProgress)
			}
			if mode == "no-address" {
				f.worker.Properties.StaticIP = nil
			}
			if mode == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			requireDeadlineError(t, g.Check(ctx, c), hostedexecution.ErrUnavailable)
			if f.reads > 2 {
				t.Fatal("uncertain inspection retried or continued", f.reads)
			}
		})
	}
}
