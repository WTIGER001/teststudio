package azureworker

type privateZoneBinding struct {
	ID, VirtualNetwork, LinkName string
	Records                      map[string]string
	ResolutionPolicy             bool
}

func (s PrivateDNSSettings) binding(c Config, zone string) privateZoneBinding {
	return privateZoneBinding{ID: zone, VirtualNetwork: s.DNS.Firewall.Perimeter.Topology.WorkerVNet, LinkName: "worker-network", Records: s.records(c, zone), ResolutionPolicy: zone == s.RegistryZone}
}
