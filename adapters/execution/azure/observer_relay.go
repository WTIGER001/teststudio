package azureworker

func (b *observerBuilder) relay(s ReadIdentitySettings, endpoint *BackendEndpointGuard) *ReadIdentityGuard {
	c, credential, transport := b.worker, b.credential, b.transport
	zoneSettings := s.Protection.PrivateDNS
	registrySettings := zoneSettings.Endpoint
	dnsSettings := registrySettings.DNS
	firewallSettings := dnsSettings.Firewall
	perimeterSettings := firewallSettings.Perimeter
	topologySettings := perimeterSettings.Topology
	topology := observerStage(b, func() (*SDKRelayTopologyReader, error) {
		return NewSDKRelayTopologyReader(c, topologySettings, credential, transport)
	}, func(reader *SDKRelayTopologyReader) (*RelayTopologyGuard, error) {
		return NewRelayTopologyGuard(c, topologySettings, endpoint, reader)
	})
	perimeter := observerStage(b, func() (*SDKPerimeterReader, error) {
		return NewSDKRelayPerimeterReader(c, perimeterSettings, credential, transport)
	}, func(reader *SDKPerimeterReader) (*RelayPerimeterGuard, error) {
		return NewRelayPerimeterGuard(c, perimeterSettings, topology, reader)
	})
	firewall := observerStage(b, func() (*SDKFirewallReader, error) {
		return NewSDKRelayFirewallReader(c, firewallSettings, credential, transport)
	}, func(reader *SDKFirewallReader) (*RelayFirewallGuard, error) {
		return NewRelayFirewallGuard(c, firewallSettings, perimeter, reader)
	})
	dns := observerStage(b, func() (*SDKDNSReader, error) {
		return NewSDKRelayDNSReader(c, dnsSettings, credential, transport)
	}, func(reader *SDKDNSReader) (*RelayDNSGuard, error) {
		return NewRelayDNSGuard(c, dnsSettings, firewall, reader)
	})
	registry := observerStage(b, func() (*SDKRelayRegistryEndpointReader, error) {
		return NewSDKRelayRegistryEndpointReader(c, registrySettings, credential, transport)
	}, func(reader *SDKRelayRegistryEndpointReader) (*RelayRegistryEndpointGuard, error) {
		return NewRelayRegistryEndpointGuard(c, registrySettings, dns, reader)
	})
	zones := observerStage(b, func() (*SDKPrivateDNSReader, error) {
		return NewSDKRelayPrivateDNSReader(c, zoneSettings, credential, transport)
	}, func(reader *SDKPrivateDNSReader) (*RelayPrivateDNSGuard, error) {
		return NewRelayPrivateDNSGuard(c, zoneSettings, registry, reader)
	})
	inventory := observerStage(b, func() (*SDKPrivateDNSInventoryReader, error) {
		return NewSDKPrivateDNSInventoryReader(c, credential, transport)
	}, func(reader *SDKPrivateDNSInventoryReader) (*PrivateDNSInventoryGuard, error) {
		return NewPrivateDNSInventoryGuard(c, zoneSettings, zones, reader)
	})
	protection := observerStage(b, func() (*SDKNetworkDenyReader, error) {
		return NewSDKNetworkDenyReader(c, s.Protection, credential, transport)
	}, func(reader *SDKNetworkDenyReader) (*NetworkDenyGuard, error) {
		return NewNetworkDenyGuard(c, s.Protection, inventory, reader)
	})
	identity := observerStage(b, func() (*SDKReadIdentityReader, error) {
		return NewSDKReadIdentityReader(c, s, credential, transport)
	}, func(reader *SDKReadIdentityReader) (*ReadIdentityGuard, error) {
		return NewReadIdentityGuard(c, s, protection, reader)
	})
	return identity
}
