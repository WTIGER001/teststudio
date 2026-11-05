package azuresandbox

import (
	"net/netip"
	"slices"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type NetworkSettings struct {
	FirewallIP         string           `json:"firewallIp"`
	PrivateEndpointIPs []string         `json:"privateEndpointIps"`
	Topology           TopologySettings `json:"topology"`
	Resolver           ResolverSettings `json:"resolver"`
}

func (n NetworkSettings) validate(subnet string) error {
	space, err := netip.ParsePrefix(subnet)
	if err != nil || len(n.PrivateEndpointIPs) == 0 || len(n.PrivateEndpointIPs) > 8 || !slices.IsSorted(n.PrivateEndpointIPs) {
		return hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	for _, raw := range append([]string{n.FirewallIP}, n.PrivateEndpointIPs...) {
		ip, err := netip.ParseAddr(raw)
		if err != nil || !ip.Is4() || !ip.IsPrivate() || ip.String() != raw || space.Contains(ip) || seen[raw] {
			return hostedexecution.ErrConfiguration
		}
		seen[raw] = true
	}
	return nil
}
