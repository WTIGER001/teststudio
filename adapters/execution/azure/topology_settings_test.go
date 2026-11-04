package azureworker

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureTopologySettingsRequirePrivateDedicatedCanonicalResources(t *testing.T) {
	c, s, _, g := topologyTest(t)
	cases := map[string]func(*Config, *TopologySettings){
		"shared-vnet": func(_ *Config, s *TopologySettings) {
			s.ControllerVNet = s.WorkerVNet
			s.ControllerSubnet = s.WorkerVNet + "/subnets/core"
		},
		"foreign-subscription": func(_ *Config, s *TopologySettings) {
			s.WorkerVNet = strings.Replace(s.WorkerVNet, "11111111-2222-3333-4444-555555555555", "99999999-2222-3333-4444-555555555555", 1)
		},
		"foreign-subnet-parent": func(_ *Config, s *TopologySettings) { s.WorkerSubnet = s.ControllerVNet + "/subnets/workers" },
		"public-prefix":         func(_ *Config, s *TopologySettings) { s.WorkerVNetPrefix = "8.0.0.0/8" },
		"partly-public-prefix":  func(_ *Config, s *TopologySettings) { s.WorkerVNetPrefix = "192.168.0.0/15" },
		"host-prefix":           func(_ *Config, s *TopologySettings) { s.WorkerSubnetPrefix = "10.20.0.4/24" },
		"outside-prefix":        func(_ *Config, s *TopologySettings) { s.WorkerSubnetPrefix = "10.21.0.0/24" },
		"small-subnet":          func(_ *Config, s *TopologySettings) { s.WorkerSubnetPrefix = "10.20.0.0/28" },
		"mapped-ipv4":           func(_ *Config, s *TopologySettings) { s.WorkerVNetPrefix = "::ffff:10.20.0.0/112" },
		"shared-infrastructure": func(c *Config, s *TopologySettings) { s.InfrastructureGroup = c.ResourceGroup },
		"query-resource":        func(_ *Config, s *TopologySettings) { s.RouteTable += "?expand=secret" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cc, ss := c, s
			mutate(&cc, &ss)
			_, err := NewTopologyGuard(cc, ss, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	for _, raw := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		if _, valid := privatePrefix(raw); !valid {
			t.Fatal("valid private space rejected", raw)
		}
	}
	if _, ok := any(&TopologyGuard{}).(Network); ok {
		t.Fatal("topology inspection alone became an execution admission hook")
	}
}
