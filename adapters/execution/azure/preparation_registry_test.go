package azureworker

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePreparationRegistryBindsFreshTrustedPathAfterServiceApproval(t *testing.T) {
	c, s, f, g := preparationRegistryTest(t)
	f.source.Properties.AddressPrefix = nil
	f.source.Properties.AddressPrefixes = []*string{to.Ptr(s.SourceSubnetPrefix)}
	f.network.Properties.Subnets[len(f.network.Properties.Subnets)-1] = to.Ptr(clone(f.source))
	core := f.network.Properties.Subnets[0]
	core.Properties.AddressPrefixes = []*string{core.Properties.AddressPrefix}
	core.Properties.AddressPrefix = nil
	s.ControllerPrefixes[0] = "10.22.0.0/16"
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*6 || f.dns.reads != i*3 || f.registry.reads != i*4 {
			t.Fatal("trusted preparation path rejected or parent repeated", err, f.reads, f.dns.reads, f.registry.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("preparation observation enabled native execution")
	}
	f.registry.connections[1].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.ConnectionStatusPending)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 12 || f.dns.reads != 6 {
		t.Fatal("pending service approval reached preparation reads")
	}
	f.registry.connections[1].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.ConnectionStatusApproved)
	f.nic.Properties.IPConfigurations[1].Properties.PrivateIPAddress = to.Ptr("10.21.3.6")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.dns.reads != 6 {
		t.Fatal("unverified endpoint reached private DNS")
	}
	f.nic.Properties.IPConfigurations[1].Properties.PrivateIPAddress = to.Ptr(s.DataIP)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("restored trusted path rejected", err)
	}
	c.NetworkProfile = "changed-profile"
	before := f.reads
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("foreign execution settings reached preparation reads")
	}
}

func TestAzurePreparationRegistryRejectsSourceEndpointAndDNSDrift(t *testing.T) {
	for name, check := range map[string]struct {
		change func(*preparationRegistryFixture)
		want   error
	}{
		"source-outside-vnet": {change: func(f *preparationRegistryFixture) { f.source.Properties.AddressPrefix = to.Ptr("10.22.2.0/24") }, want: hostedexecution.ErrOwnership},
		"source-overlaps-core": {change: func(f *preparationRegistryFixture) {
			f.network.Properties.Subnets[0].Properties.AddressPrefix = to.Ptr(f.settings.SourceSubnetPrefix)
		}, want: hostedexecution.ErrOwnership},
		"missing-embedded-source": {change: func(f *preparationRegistryFixture) { f.network.Properties.Subnets = f.network.Properties.Subnets[:2] }, want: hostedexecution.ErrUnavailable},
		"overlapping-address-space": {change: func(f *preparationRegistryFixture) {
			f.network.Properties.AddressSpace.AddressPrefixes = append(f.network.Properties.AddressSpace.AddressPrefixes, to.Ptr(f.settings.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNetPrefix))
		}, want: hostedexecution.ErrOwnership},
		"duplicate-address-space": {change: func(f *preparationRegistryFixture) {
			f.network.Properties.AddressSpace.AddressPrefixes = append(f.network.Properties.AddressSpace.AddressPrefixes, f.network.Properties.AddressSpace.AddressPrefixes[0])
		}, want: hostedexecution.ErrOwnership},
		"network-excludes-path": {change: func(f *preparationRegistryFixture) {
			f.network.Properties.AddressSpace.AddressPrefixes = []*string{to.Ptr("10.22.0.0/16")}
		}, want: hostedexecution.ErrOwnership},
		"worker-vnet": {change: func(f *preparationRegistryFixture) {
			f.network.ID = to.Ptr(f.settings.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNet)
		}, want: hostedexecution.ErrOwnership},
		"broad-source": {change: func(f *preparationRegistryFixture) {
			f.security.Properties.SecurityRules[0].Properties.SourceAddressPrefix = to.Ptr("VirtualNetwork")
		}, want: hostedexecution.ErrOwnership},
		"shared-endpoint-subnet": {change: func(f *preparationRegistryFixture) {
			f.subnet.Properties.PrivateEndpoints = append(f.subnet.Properties.PrivateEndpoints, &n.PrivateEndpoint{ID: to.Ptr(f.settings.Registry.Endpoints.RegistryEndpoint)})
		}, want: hostedexecution.ErrOwnership},
		"worker-dns-address": {change: func(f *preparationRegistryFixture) {
			f.dns.records[f.settings.RegistryZone][1].Properties.ARecords[0].IPv4Address = to.Ptr("10.20.2.5")
		}, want: hostedexecution.ErrOwnership},
		"worker-dns-link": {change: func(f *preparationRegistryFixture) {
			f.dns.links[f.settings.RegistryZone][0].Properties.VirtualNetwork.ID = to.Ptr(f.settings.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNet)
		}, want: hostedexecution.ErrOwnership},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := preparationRegistryTest(t)
			check.change(f)
			requireDeadlineError(t, g.Check(t.Context(), c), check.want)
		})
	}
}

func TestAzurePreparationRegistrySettingsRejectMixedTrustedAndWorkerBindings(t *testing.T) {
	c, s, f, g := preparationRegistryTest(t)
	for _, change := range []func(*PreparationRegistrySettings){
		func(s *PreparationRegistrySettings) {
			s.ControllerPrefixes = append(s.ControllerPrefixes, s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNetPrefix)
		},
		func(s *PreparationRegistrySettings) {
			s.ControllerPrefixes = append(s.ControllerPrefixes, s.ControllerPrefixes[0])
		},

		func(s *PreparationRegistrySettings) {
			s.SourceSubnet = s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerSubnet
		},
		func(s *PreparationRegistrySettings) { s.Subnet = s.SourceSubnet },
		func(s *PreparationRegistrySettings) { s.SourceSubnetPrefix = s.SubnetPrefix },
		func(s *PreparationRegistrySettings) { s.RegistryZone = s.Registry.Endpoints.PrivateDNS.RegistryZone },
		func(s *PreparationRegistrySettings) { s.Interface = s.Registry.Endpoints.RegistryNIC },
		func(s *PreparationRegistrySettings) {
			s.Registry.PreparationEndpoint = c.resourceID("Microsoft.Network", "privateEndpoints", "other-preparation")
		},
	} {
		changed := s
		change(&changed)
		_, err := NewPreparationRegistryGuard(c, changed, g.registry, g.reader, g.dns)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	if f.reads != 0 || f.dns.reads != 0 || f.registry.reads != 0 {
		t.Fatal("invalid trusted settings performed observation")
	}
}

func TestAzurePreparationRegistrySDKRestrictsResourcesAndRetainsUncertainty(t *testing.T) {
	c, s, f, g := preparationRegistryTest(t)
	reader := g.reader.(*SDKPreparationRegistryReader)
	for _, change := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.URL.Path = s.Registry.Endpoints.RegistryEndpoint },
		func(r *http.Request) {
			r.URL.Path = s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerSubnet
		},
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.RawQuery += "&%24expand=all" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Registry.PreparationEndpoint+"?api-version=2025-03-01", nil)
		change(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.VirtualNetwork(t.Context(), s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNet)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	dns := g.dns.(*SDKPrivateDNSReader)
	_, err = dns.Zone(t.Context(), s.Registry.Endpoints.PrivateDNS.RegistryZone)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 || f.dns.reads != 0 {
		t.Fatal("unselected preparation resource reached transport")
	}
	for _, mode := range []string{"unavailable", "redirect"} {
		f.mode = mode
		requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	}
	f.mode = ""
	f.dns.mode = "missing"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}
