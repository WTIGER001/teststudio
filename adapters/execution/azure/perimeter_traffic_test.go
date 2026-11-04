package azureworker

import (
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

func testAddressMatch(prefix, address string) bool {
	if prefix == "*" || prefix == address {
		return true
	}
	p, err := netip.ParsePrefix(prefix)
	ip, ipErr := netip.ParseAddr(address)
	return err == nil && ipErr == nil && p.Contains(ip)
}

func testPortMatch(raw string, port int) bool {
	if raw == "*" {
		return true
	}
	parts := strings.Split(raw, "-")
	low, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	if len(parts) == 1 {
		return low == port
	}
	high, err := strconv.Atoi(parts[1])
	return err == nil && port >= low && port <= high
}

func TestAzurePerimeterPolicyTrafficMatrixUsesExplicitDecisions(t *testing.T) {
	c, s, _, _, _ := perimeterTest(t)
	definition, err := s.SecurityGroupDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	rules := slices.Clone(definition.Properties.SecurityRules)
	slices.SortFunc(rules, func(a, b *n.SecurityRule) int {
		return int(value(a.Properties.Priority) - value(b.Properties.Priority))
	})
	cases := []struct {
		name, source, destination string
		protocol                  n.SecurityRuleProtocol
		port                      int
		direction                 n.SecurityRuleDirection
		want                      n.SecurityRuleAccess
	}{
		{"public-web", "10.20.0.5", "8.8.8.8", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"controller", "10.20.0.5", "10.21.0.5", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"public-dns", "10.20.0.5", "8.8.8.8", n.SecurityRuleProtocolUDP, 53, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"private-exchange", "10.20.0.5", "10.20.2.4", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"private-registry", "10.20.0.5", "10.20.2.5", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"private-data", "10.20.0.5", "10.20.2.6", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"unselected-private-endpoint", "10.20.0.5", "10.20.2.7", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"private-plaintext", "10.20.0.5", "10.20.2.4", n.SecurityRuleProtocolTCP, 80, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"private-quic", "10.20.0.5", "10.20.2.4", n.SecurityRuleProtocolUDP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"platform-dns-udp", "10.20.0.5", "AzurePlatformDNS", n.SecurityRuleProtocolUDP, 53, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"platform-dns-tcp", "10.20.0.5", "AzurePlatformDNS", n.SecurityRuleProtocolTCP, 53, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"platform-dns-other", "10.20.0.5", "AzurePlatformDNS", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"metadata", "10.20.0.5", "AzurePlatformIMDS", n.SecurityRuleProtocolTCP, 80, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"platform-registry", "10.20.0.5", "MicrosoftContainerRegistry", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"platform-registry-cdn", "10.20.0.5", "AzureFrontDoor.FirstParty", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"platform-identity", "10.20.0.5", "AzureActiveDirectory", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"shared-storage", "10.20.0.5", "Storage", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"shared-monitoring", "10.20.0.5", "AzureMonitor", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessDeny},
		{"public-inbound", "8.8.8.8", "10.20.0.5", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionInbound, n.SecurityRuleAccessDeny},
		{"controller-inbound", "10.21.0.5", "10.20.0.5", n.SecurityRuleProtocolTCP, 443, n.SecurityRuleDirectionInbound, n.SecurityRuleAccessDeny},
		{"load-balancer-probe", "AzureLoadBalancer", "10.20.0.5", n.SecurityRuleProtocolTCP, 31000, n.SecurityRuleDirectionInbound, n.SecurityRuleAccessAllow},
		{"load-balancer-other-port", "AzureLoadBalancer", "10.20.0.5", n.SecurityRuleProtocolTCP, 80, n.SecurityRuleDirectionInbound, n.SecurityRuleAccessDeny},
		{"platform-subnet-outbound", "10.20.0.5", "10.20.0.6", n.SecurityRuleProtocolTCP, 9000, n.SecurityRuleDirectionOutbound, n.SecurityRuleAccessAllow},
		{"platform-subnet-inbound", "10.20.0.5", "10.20.0.6", n.SecurityRuleProtocolTCP, 9000, n.SecurityRuleDirectionInbound, n.SecurityRuleAccessAllow},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			for _, rule := range rules {
				p := rule.Properties
				if value(p.Direction) != test.direction || value(p.Protocol) != n.SecurityRuleProtocolAsterisk && value(p.Protocol) != test.protocol || !testAddressMatch(value(p.SourceAddressPrefix), test.source) || !testPortMatch(value(p.SourcePortRange), 41234) || !testPortMatch(value(p.DestinationPortRange), test.port) {
					continue
				}
				destination := testAddressMatch(value(p.DestinationAddressPrefix), test.destination)
				for _, prefix := range p.DestinationAddressPrefixes {
					destination = destination || testAddressMatch(value(prefix), test.destination)
				}
				if !destination {
					continue
				}
				if value(p.Access) != test.want {
					t.Fatalf("traffic decision %s from %s, want %s", value(p.Access), value(rule.Name), test.want)
				}
				return
			}
			t.Fatal("traffic relied on a default rule instead of an explicit policy decision")
		})
	}
}
