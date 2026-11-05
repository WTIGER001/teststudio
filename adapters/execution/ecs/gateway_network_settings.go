package ecsworker

import (
	"regexp"
	"slices"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type GatewayNetworkSettings struct {
	VPC               string   `json:"vpc"`
	Subnets           []string `json:"subnets"`
	SecurityGroup     string   `json:"securityGroup"`
	PeeringConnection string   `json:"peeringConnection"`
	ExchangeEndpoint  string   `json:"exchangeEndpoint"`
	ExchangeGroup     string   `json:"exchangeGroup"`
	NATGateways       []string `json:"natGateways"`
}

var peeringID = regexp.MustCompile(`^pcx-(?:[a-f0-9]{8}|[a-f0-9]{17})$`)
var natID = regexp.MustCompile(`^nat-(?:[a-f0-9]{8}|[a-f0-9]{17})$`)

func (s GatewayNetworkSettings) canonical() GatewayNetworkSettings {
	s.Subnets, s.NATGateways = slices.Clone(s.Subnets), slices.Clone(s.NATGateways)
	slices.Sort(s.Subnets)
	slices.Sort(s.NATGateways)
	return s
}

func (s GatewayNetworkSettings) validate(c Config, worker NetworkSettings) error {
	if !vpcID.MatchString(s.VPC) || s.VPC == worker.VPC || s.VPC == worker.ControllerVPC || !peeringID.MatchString(s.PeeringConnection) || len(s.Subnets) < 1 || len(s.Subnets) > 16 || len(s.NATGateways) < 1 || len(s.NATGateways) > 16 || !endpointID.MatchString(s.ExchangeEndpoint) || slices.Contains([]string{worker.ExchangeEndpoint, worker.APIEndpoint, worker.DKREndpoint, worker.S3Endpoint}, s.ExchangeEndpoint) {
		return hostedexecution.ErrConfiguration
	}
	groups := []string{c.SecurityGroups[0], worker.ExchangeGroup, worker.RegistryGroup, worker.LoadBalancerGroup, worker.ControllerGroup}
	for _, id := range []string{s.SecurityGroup, s.ExchangeGroup} {
		if !securityGroup.MatchString(id) || slices.Contains(groups, id) {
			return hostedexecution.ErrConfiguration
		}
		groups = append(groups, id)
	}
	seen := map[string]bool{}
	for _, id := range s.Subnets {
		if !subnet.MatchString(id) || seen[id] || slices.Contains(c.Subnets, id) {
			return hostedexecution.ErrConfiguration
		}
		seen[id] = true
	}
	for _, id := range s.NATGateways {
		if !natID.MatchString(id) || seen[id] {
			return hostedexecution.ErrConfiguration
		}
		seen[id] = true
	}
	return nil
}
