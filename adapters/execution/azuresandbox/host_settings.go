package azuresandbox

import (
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type HostSettings struct {
	Network                HostNetworkSettings      `json:"network"`
	SubnetPrefix           string                   `json:"subnetPrefix"`
	Registry               RelayRegistrySettings    `json:"registry"`
	Profile                string                   `json:"profile"`
	Application            string                   `json:"application"`
	Environment            string                   `json:"environment"`
	Subnet                 string                   `json:"subnet"`
	InfrastructureGroup    string                   `json:"infrastructureGroup"`
	Image                  string                   `json:"image"`
	Pull                   ServiceIdentitySettings  `json:"pull"`
	Startup                ServiceIdentitySettings  `json:"startup"`
	Runtime                *ServiceIdentitySettings `json:"runtime,omitempty"`
	DatabaseSecretURL      string                   `json:"databaseSecretUrl"`
	ConfigurationSecretURL string                   `json:"configurationSecretUrl"`
	BlobServiceURL         string                   `json:"blobServiceUrl,omitempty"`
	EvidenceContainer      string                   `json:"evidenceContainer,omitempty"`
	PublicOrigin           string                   `json:"publicOrigin,omitempty"`
	Replicas               int32                    `json:"replicas"`
}

func (h HostSettings) identities() []ServiceIdentitySettings {
	result := []ServiceIdentitySettings{h.Pull, h.Startup}
	if h.Runtime != nil {
		result = append(result, *h.Runtime)
	}
	return result
}

func (h HostSettings) validate(c Config, p PlacementSettings) error {
	if !resourceID(c, h.Application, "Microsoft.App", "containerApps") || !resourceID(c, h.Environment, "Microsoft.App", "managedEnvironments") || !serviceSubnet(c, h.Subnet) || !serviceGroup(h.InfrastructureGroup) || !trustedImage(c, h.Image) || !azureworker.ValidServiceSecretReference(h.DatabaseSecretURL) || !azureworker.ValidServiceSecretReference(h.ConfigurationSecretURL) || h.DatabaseSecretURL == h.ConfigurationSecretURL || h.Replicas < 1 || h.Replicas > 10 {
		return hostedexecution.ErrConfiguration
	}
	name := h.Application[strings.LastIndex(h.Application, "/")+1:]
	if len(name) > 32 || strings.Contains(name, "--") {
		return hostedexecution.ErrConfiguration
	}
	for _, id := range []string{p.Services.Relay.Application, p.Services.Backend.Application} {
		if strings.EqualFold(id, h.Application) {
			return hostedexecution.ErrConfiguration
		}
	}
	for _, id := range []string{p.Private.ExchangeEnvironment, p.Services.Backend.Environment} {
		if strings.EqualFold(id, h.Environment) {
			return hostedexecution.ErrConfiguration
		}
	}
	vnet := strings.Join(strings.Split(h.Subnet, "/")[:9], "/")
	for _, subnet := range []string{p.Subnet, p.Services.Relay.Subnet, p.Services.Backend.Subnet} {
		if strings.EqualFold(vnet, strings.Join(strings.Split(subnet, "/")[:9], "/")) {
			return hostedexecution.ErrConfiguration
		}
	}
	for _, group := range []string{c.ResourceGroup, p.Services.Relay.InfrastructureGroup, p.Services.Backend.InfrastructureGroup} {
		if strings.EqualFold(group, h.InfrastructureGroup) {
			return hostedexecution.ErrConfiguration
		}
	}
	resources, clients, principals := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, pin := range append(p.Services.identities(), h.identities()...) {
		id := strings.ToLower(pin.Resource)
		if !c.validPullIdentity(pin.Resource) || !nonzeroGUID(pin.Client) || !nonzeroGUID(pin.Principal) || pin.Client == pin.Principal || resources[id] || clients[pin.Client] || principals[pin.Principal] {
			return hostedexecution.ErrConfiguration
		}
		resources[id], clients[pin.Client], principals[pin.Principal] = true, true, true
	}
	return nil
}

func (h HostSettings) validateMode(s Settings) error {
	if s.Mode == "execution" {
		c := s.Worker
		c.ExchangeOrigin = h.PublicOrigin
		if h.Runtime == nil || Validate(c) != nil || h.PublicOrigin == s.Worker.ExchangeOrigin || h.PublicOrigin == s.Placement.Services.Backend.Origin || !azureworker.ValidBackendDependencies(h.DatabaseSecretURL, h.BlobServiceURL, h.EvidenceContainer) {
			return hostedexecution.ErrConfiguration
		}
	} else if h.Runtime != nil || h.BlobServiceURL != "" || h.EvidenceContainer != "" || h.PublicOrigin != "" {
		return hostedexecution.ErrConfiguration
	}
	known := []ServiceIdentitySettings{{s.PullIdentity, s.Placement.PullClientID, s.Placement.PullPrincipalID}, {s.Timer.StopIdentity, s.Authority.StopClientID, s.Authority.StopPrincipalID}, {s.Authority.Control.Resource, s.ControlClientID, s.Authority.Control.PrincipalID}, {s.Authority.Registry.Resource, s.RegistryClientID, s.Authority.Registry.PrincipalID}, {s.Authority.Metadata.Resource, s.MetadataClientID, s.Authority.Metadata.PrincipalID}}
	for _, pin := range h.identities() {
		for _, other := range known {
			if strings.EqualFold(pin.Resource, other.Resource) || pin.Client == other.Client || pin.Principal == other.Principal {
				return hostedexecution.ErrConfiguration
			}
		}
	}
	return nil
}

func (s Settings) hostWorkloadBinding() azureworker.ServiceWorkloadBinding {
	h := s.Host
	pins := map[string]azureworker.ServiceIdentityPin{h.Pull.Resource: {Client: h.Pull.Client, Principal: h.Pull.Principal, Lifecycle: "None"}, h.Startup.Resource: {Client: h.Startup.Client, Principal: h.Startup.Principal, Lifecycle: "None"}, s.Authority.Control.Resource: {Client: s.ControlClientID, Principal: s.Authority.Control.PrincipalID, Lifecycle: "Main"}, s.Authority.Registry.Resource: {Client: s.RegistryClientID, Principal: s.Authority.Registry.PrincipalID, Lifecycle: "Main"}, s.Authority.Metadata.Resource: {Client: s.MetadataClientID, Principal: s.Authority.Metadata.PrincipalID, Lifecycle: "Main"}}
	if h.Runtime != nil {
		pins[h.Runtime.Resource] = azureworker.ServiceIdentityPin{Client: h.Runtime.Client, Principal: h.Runtime.Principal, Lifecycle: "Main"}
	}
	return azureworker.ServiceWorkloadBinding{Application: h.Application, Environment: h.Environment, Identities: pins}
}

func (s Settings) workloadIdentities() []ServiceIdentitySettings {
	result := append(s.Placement.Services.identities(), s.Host.identities()...)
	return append(result, ServiceIdentitySettings{s.Authority.Control.Resource, s.ControlClientID, s.Authority.Control.PrincipalID}, ServiceIdentitySettings{s.Authority.Registry.Resource, s.RegistryClientID, s.Authority.Registry.PrincipalID}, ServiceIdentitySettings{s.Authority.Metadata.Resource, s.MetadataClientID, s.Authority.Metadata.PrincipalID})
}
