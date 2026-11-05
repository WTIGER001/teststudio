package azureworker

import (
	"net/netip"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type RelayTopologySettings struct {
	BackendEndpoint      BackendEndpointSettings `json:"backendEndpoint"`
	VirtualNetworkPrefix string                  `json:"virtualNetworkPrefix"`
}

func (s RelayTopologySettings) canonical(c Config) (RelayTopologySettings, error) {
	var err error
	s.BackendEndpoint, err = s.BackendEndpoint.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	prefix, ok := privatePrefix(s.VirtualNetworkPrefix)
	endpoint, _ := netip.ParsePrefix(s.BackendEndpoint.SubnetPrefix)
	relay, _ := netip.ParsePrefix(s.BackendEndpoint.RelaySubnetPrefix)
	worker, _ := netip.ParsePrefix(s.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNetPrefix)
	if !ok || prefix.Overlaps(worker) || !containsPrefix(prefix, endpoint) || !containsPrefix(prefix, relay) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func containsPrefix(parent, child netip.Prefix) bool {
	return parent.Bits() <= child.Bits() && parent.Contains(child.Addr())
}
