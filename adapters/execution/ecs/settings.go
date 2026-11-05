package ecsworker

import (
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

type Settings struct {
	Environment *EnvironmentSettings `json:"environment,omitempty"`
	Worker      Config               `json:"worker"`
	Network     NetworkSettings      `json:"network"`
}

type EnvironmentSettings struct {
	Worker  Config          `json:"worker"`
	Network NetworkSettings `json:"network"`
	Gateway GatewayConfig   `json:"gateway"`
}

func (s Settings) bind() (Settings, error) {
	if s.Network.Gateway != nil {
		return s, hostedexecution.ErrConfiguration
	}
	worker, err := s.Network.Bind(s.Worker)
	if err != nil {
		return s, err
	}
	s.Worker = worker
	if s.Environment == nil {
		return s, nil
	}
	environment := *s.Environment
	environment.Worker, err = environment.Network.Bind(environment.Worker)
	if err != nil || environment.Network.Gateway == nil || validateGateway(environment.Worker, environment.Gateway) != nil {
		return s, hostedexecution.ErrConfiguration
	}
	e, n := environment.Worker, environment.Network
	gateway := n.Gateway
	if e.Region != worker.Region || e.Account != worker.Account || e.Architecture != worker.Architecture || e.Cluster != worker.Cluster || e.ExchangeOrigin != worker.ExchangeOrigin || n.ControllerVPC != s.Network.ControllerVPC || n.RepositoryPrefix != s.Network.RepositoryPrefix || n.ExchangeService != s.Network.ExchangeService || n.LoadBalancer != s.Network.LoadBalancer || n.TargetGroup != s.Network.TargetGroup || n.LoadBalancerGroup != s.Network.LoadBalancerGroup || n.ControllerGroup != s.Network.ControllerGroup || n.VPC == s.Network.VPC || gateway.VPC == s.Network.VPC || environment.Gateway.SecurityGroup != gateway.SecurityGroup || !equalSet(environment.Gateway.Subnets, gateway.Subnets) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, id := range append(slices.Clone(e.Subnets), gateway.Subnets...) {
		if slices.Contains(worker.Subnets, id) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	for _, id := range []string{e.SecurityGroups[0], n.ExchangeGroup, n.RegistryGroup, gateway.SecurityGroup, gateway.ExchangeGroup} {
		if slices.Contains([]string{worker.SecurityGroups[0], s.Network.ExchangeGroup, s.Network.RegistryGroup, s.Network.ControllerGroup, s.Network.LoadBalancerGroup}, id) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	for _, id := range []string{n.ExchangeEndpoint, n.APIEndpoint, n.DKREndpoint, n.S3Endpoint, gateway.ExchangeEndpoint} {
		if slices.Contains([]string{s.Network.ExchangeEndpoint, s.Network.APIEndpoint, s.Network.DKREndpoint, s.Network.S3Endpoint}, id) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	imagePrefix := strings.TrimPrefix(registryOrigin(worker), "https://") + "/" + n.RepositoryPrefix + "/"
	if !strings.HasPrefix(environment.Gateway.Image, imagePrefix) {
		return s, hostedexecution.ErrConfiguration
	}
	environment.Gateway = canonicalGateway(environment.Gateway)
	selected := gateway.canonical()
	environment.Network.Gateway = &selected
	s.Environment = &environment
	return s, nil
}

func LoadSettings(file, variable string, lookup func(string) (string, bool)) (*Settings, error) {
	if variable != "" && lookup == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	var settings Settings
	found, err := runtimeconfig.LoadJSON(file, variable, "ECS worker", lookup, &settings)
	if err != nil || !found {
		return nil, err
	}
	settings, err = settings.bind()
	if err != nil {
		return nil, err
	}
	return &settings, nil
}
