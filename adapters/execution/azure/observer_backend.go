package azureworker

func (b *observerBuilder) registry(registrySettings RegistrySettings) *RegistryGuard {
	c, credential, transport := b.worker, b.credential, b.transport
	endpointSettings := registrySettings.Endpoints
	zoneSettings := endpointSettings.PrivateDNS
	dnsSettings := zoneSettings.DNS
	firewallSettings := dnsSettings.Firewall
	perimeterSettings := firewallSettings.Perimeter
	topologySettings := perimeterSettings.Topology
	topology := observerStage(b, func() (*SDKTopologyReader, error) {
		return NewSDKTopologyReader(c, topologySettings, credential, transport)
	}, func(reader *SDKTopologyReader) (*TopologyGuard, error) {
		return NewTopologyGuard(c, topologySettings, reader)
	})
	perimeter := observerStage(b, func() (*SDKPerimeterReader, error) {
		return NewSDKPerimeterReader(c, perimeterSettings, credential, transport)
	}, func(reader *SDKPerimeterReader) (*PerimeterGuard, error) {
		return NewPerimeterGuard(c, perimeterSettings, topology, reader)
	})
	firewall := observerStage(b, func() (*SDKFirewallReader, error) {
		return NewSDKFirewallReader(c, firewallSettings, credential, transport)
	}, func(reader *SDKFirewallReader) (*FirewallGuard, error) {
		return NewFirewallGuard(c, firewallSettings, perimeter, reader)
	})
	dns := observerStage(b, func() (*SDKDNSReader, error) {
		return NewSDKDNSReader(c, dnsSettings, credential, transport)
	}, func(reader *SDKDNSReader) (*DNSGuard, error) {
		return NewDNSGuard(c, dnsSettings, firewall, reader)
	})
	zones := observerStage(b, func() (*SDKPrivateDNSReader, error) {
		return NewSDKPrivateDNSReader(c, zoneSettings, credential, transport)
	}, func(reader *SDKPrivateDNSReader) (*PrivateDNSGuard, error) {
		return NewPrivateDNSGuard(c, zoneSettings, dns, reader)
	})
	endpoints := observerStage(b, func() (*SDKEndpointReader, error) {
		return NewSDKEndpointReader(c, endpointSettings, credential, transport)
	}, func(reader *SDKEndpointReader) (*EndpointGuard, error) {
		return NewEndpointGuard(c, endpointSettings, zones, reader)
	})
	return observerStage(b, func() (*SDKRegistryReader, error) {
		return NewSDKRegistryReader(c, registrySettings, credential, transport)
	}, func(reader *SDKRegistryReader) (*RegistryGuard, error) {
		return NewRegistryGuard(c, registrySettings, endpoints, reader)
	})
}

func (b *observerBuilder) backend(s BackendEndpointSettings) *BackendEndpointGuard {
	c, credential, transport := b.worker, b.credential, b.transport
	backendSettings := s.Backend
	exchangeSettings := backendSettings.Exchange
	registry := b.registry(exchangeSettings.Registry)
	exchange := observerStage(b, func() (*SDKExchangeReader, error) {
		return NewSDKExchangeReader(c, exchangeSettings, credential, transport)
	}, func(reader *SDKExchangeReader) (*ExchangeGuard, error) {
		return NewExchangeGuard(c, exchangeSettings, registry, reader)
	})
	inventory := observerStage(b, func() (*SDKWorkloadInventoryReader, error) {
		if b.host != nil {
			return NewSDKObserverWorkloadInventoryReader(c, *b.host, credential, transport)
		}
		return NewSDKWorkloadInventoryReader(c, credential, transport)
	}, func(reader *SDKWorkloadInventoryReader) (*WorkloadInventoryGuard, error) {
		return NewWorkloadInventoryGuard(c, exchangeSettings, exchange, reader)
	})
	backend := observerStage(b, func() (*SDKBackendReader, error) {
		return NewSDKBackendReader(c, backendSettings, credential, transport)
	}, func(reader *SDKBackendReader) (*BackendGuard, error) {
		return NewBackendGuard(c, backendSettings, inventory, reader)
	})
	endpoint := observerStage(b, func() (*SDKBackendEndpointReader, error) {
		return NewSDKBackendEndpointReader(c, s, credential, transport)
	}, func(reader *SDKBackendEndpointReader) (*BackendEndpointGuard, error) {
		return NewBackendEndpointGuard(c, s, backend, reader)
	})
	return endpoint
}
