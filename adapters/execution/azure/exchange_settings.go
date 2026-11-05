package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/adapters/workerexchange/relay"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ExchangeSettings struct {
	Registry            RegistrySettings `json:"registry"`
	Application         string           `json:"application"`
	Certificate         string           `json:"certificate"`
	VirtualNetwork      string           `json:"virtualNetwork"`
	Subnet              string           `json:"subnet"`
	InfrastructureGroup string           `json:"infrastructureGroup"`
	Image               string           `json:"image"`
	BackendOrigin       string           `json:"backendOrigin"`
	BackendAddress      string           `json:"backendAddress"`
}

func (s ExchangeSettings) environment() string { return s.Registry.Endpoints.ExchangeEnvironment }
func (s ExchangeSettings) connections() string {
	return s.environment() + "/privateEndpointConnections"
}
func (s ExchangeSettings) links() string     { return s.environment() + "/privateLinkResources" }
func (s ExchangeSettings) routes() string    { return s.environment() + "/httpRouteConfigs" }
func (s ExchangeSettings) revisions() string { return s.Application + "/revisions" }
func (s ExchangeSettings) auth() string      { return s.Application + "/authConfigs/current" }

func (s ExchangeSettings) canonical(c Config) (ExchangeSettings, error) {
	var err error
	s.Registry, err = s.Registry.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	topology := s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology
	cert := strings.TrimPrefix(s.Certificate, s.environment()+"/certificates/")
	_, backendPrivate := privateHost(s.BackendAddress)
	if !c.ownsResource(s.Application, "Microsoft.App", "containerApps") ||
		!backendPrivate ||
		!resourceName.MatchString(cert) ||
		s.Certificate != s.environment()+"/certificates/"+cert ||
		!c.ownsResource(s.VirtualNetwork, "Microsoft.Network", "virtualNetworks") ||
		strings.EqualFold(s.VirtualNetwork, topology.WorkerVNet) ||
		strings.EqualFold(s.VirtualNetwork, topology.ControllerVNet) ||
		!subnetResource(c, s.Subnet, s.VirtualNetwork) ||
		!groupName.MatchString(s.InfrastructureGroup) ||
		strings.HasSuffix(s.InfrastructureGroup, ".") ||
		strings.EqualFold(s.InfrastructureGroup, topology.InfrastructureGroup) ||
		strings.EqualFold(s.InfrastructureGroup, c.ResourceGroup) ||
		relay.ValidateOrigins(c.ExchangeOrigin, s.BackendOrigin) != nil {
		return s, hostedexecution.ErrConfiguration
	}
	_, name := resourceParts(s.Application)
	if len(name) > 32 || strings.Contains(name, "--") {
		return s, hostedexecution.ErrConfiguration
	}
	if !trustedServiceImage(c, s.Image) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}
