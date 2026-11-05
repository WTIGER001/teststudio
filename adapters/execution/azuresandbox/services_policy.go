package azuresandbox

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
)

func validBackendDependencies(b BackendServiceSettings) bool {
	return azureworker.ValidBackendDependencies(b.DatabaseSecretURL, b.BlobServiceURL, b.EvidenceContainer)
}

func (s Settings) relayDefinition() a.ContainerApp {
	r, b := s.Placement.Services.Relay, s.Placement.Services.Backend
	return azureworker.RelayApplicationDefinition(azureworker.RelayDefinitionBinding{Region: s.Worker.Region, Registry: s.Worker.Registry, Origin: s.Worker.ExchangeOrigin, Environment: s.Placement.Private.ExchangeEnvironment, PullIdentity: r.Pull.Resource, Certificate: r.Certificate, Image: r.Image, BackendOrigin: b.Origin, BackendAddress: b.Address})
}

func (s Settings) backendDefinition() a.ContainerApp {
	b := s.Placement.Services.Backend
	return azureworker.BackendApplicationDefinition(s.Worker.Region, s.Worker.Registry, azureworker.BackendSettings{Application: b.Application, Environment: b.Environment, Subnet: b.Subnet, InfrastructureGroup: b.InfrastructureGroup, Image: b.Image, PullIdentity: b.Pull.Resource, StartupIdentity: b.Startup.Resource, RuntimeIdentity: b.Runtime.Resource, RuntimeClientID: b.Runtime.Client, DatabaseSecretURL: b.DatabaseSecretURL, BlobServiceURL: b.BlobServiceURL, EvidenceContainer: b.EvidenceContainer, Replicas: b.Replicas})
}

func (s Settings) ServiceDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	result := map[string]any{}
	for id, body := range map[string]a.ContainerApp{s.Placement.Services.Relay.Application: s.relayDefinition(), s.Placement.Services.Backend.Application: s.backendDefinition()} {
		result[id] = body
		result[id+"/authConfigs/current"] = a.AuthConfig{Properties: &a.AuthConfigProperties{Platform: &a.AuthPlatform{Enabled: to.Ptr(false)}}}
	}
	return result, nil
}
