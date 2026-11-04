package azureworker

import (
	"net/netip"
	"slices"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PerimeterSettings struct {
	Topology           TopologySettings `json:"topology"`
	FirewallIP         string           `json:"firewallIp"`
	PrivateEndpointIPs []string         `json:"privateEndpointIps"`
}

func privateHost(raw string) (netip.Addr, bool) {
	ip, err := netip.ParseAddr(raw)
	return ip, err == nil && ip.Is4() && ip.IsPrivate() && ip.String() == raw
}

func (s PerimeterSettings) canonical(c Config) (PerimeterSettings, error) {
	if s.Topology.validate(c) != nil || len(s.PrivateEndpointIPs) < 2 || len(s.PrivateEndpointIPs) > 8 {
		return s, hostedexecution.ErrConfiguration
	}
	s.PrivateEndpointIPs = slices.Clone(s.PrivateEndpointIPs)
	slices.Sort(s.PrivateEndpointIPs)
	vnet, _ := netip.ParsePrefix(s.Topology.WorkerVNetPrefix)
	worker, _ := netip.ParsePrefix(s.Topology.WorkerSubnetPrefix)
	seen := map[string]bool{}
	for _, raw := range append([]string{s.FirewallIP}, s.PrivateEndpointIPs...) {
		ip, ok := privateHost(raw)
		if !ok || !vnet.Contains(ip) || worker.Contains(ip) || seen[raw] {
			return s, hostedexecution.ErrConfiguration
		}
		seen[raw] = true
	}
	return s, nil
}
