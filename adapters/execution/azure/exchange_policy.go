package azureworker

import (
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

func (s ExchangeSettings) ApplicationDefinition(c Config) (a.ContainerApp, error) {
	s, err := s.canonical(c)
	if err != nil {
		return a.ContainerApp{}, err
	}
	return RelayApplicationDefinition(RelayDefinitionBinding{Region: c.Region, Registry: c.Registry, Origin: c.ExchangeOrigin, Environment: s.environment(), PullIdentity: c.PullIdentity, Certificate: s.Certificate, Image: s.Image, BackendOrigin: s.BackendOrigin, BackendAddress: s.BackendAddress}), nil
}

type RelayDefinitionBinding struct {
	Region, Registry, Origin, Environment, PullIdentity, Certificate, Image, BackendOrigin, BackendAddress string
}

func RelayApplicationDefinition(b RelayDefinitionBinding) a.ContainerApp {
	origin, _ := url.Parse(b.Origin)
	probes := []*a.ContainerAppProbe{}
	for _, p := range []struct {
		kind             a.Type
		path             string
		period, failures int32
	}{{a.TypeStartup, "/healthz", 3, 30}, {a.TypeReadiness, "/readyz", 10, 3}, {a.TypeLiveness, "/livez", 30, 3}} {
		timeout := int32(2)
		if p.kind == a.TypeReadiness {
			timeout = 4
		}
		probes = append(probes, &a.ContainerAppProbe{Type: to.Ptr(p.kind), HTTPGet: &a.ContainerAppProbeHTTPGet{Path: to.Ptr(p.path), Port: to.Ptr[int32](8043), Scheme: to.Ptr(a.SchemeHTTP)}, InitialDelaySeconds: to.Ptr[int32](1), PeriodSeconds: to.Ptr(p.period), TimeoutSeconds: to.Ptr(timeout), SuccessThreshold: to.Ptr[int32](1), FailureThreshold: to.Ptr(p.failures)})
	}

	ingress := &a.Ingress{
		External: to.Ptr(true), AllowInsecure: to.Ptr(false), TargetPort: to.Ptr[int32](8043), Transport: to.Ptr(a.IngressTransportMethodHTTP),
		CustomDomains: []*a.CustomDomain{{Name: to.Ptr(origin.Hostname()), BindingType: to.Ptr(a.BindingTypeSniEnabled), CertificateID: to.Ptr(b.Certificate)}},
		Traffic:       []*a.TrafficWeight{{LatestRevision: to.Ptr(true), Weight: to.Ptr[int32](100)}},
	}
	configuration := &a.Configuration{
		ActiveRevisionsMode: to.Ptr(a.ActiveRevisionsModeSingle), MaxInactiveRevisions: to.Ptr[int32](1), Ingress: ingress,
		IdentitySettings: []*a.IdentitySettings{{Identity: to.Ptr(b.PullIdentity), Lifecycle: to.Ptr(a.IdentitySettingsLifeCycleNone)}},
		Registries:       []*a.RegistryCredentials{{Server: to.Ptr(b.Registry), Identity: to.Ptr(b.PullIdentity)}},
	}
	container := &a.Container{
		Name: to.Ptr("relay"), Image: to.Ptr(b.Image), Command: []*string{to.Ptr("/app/teststudio-exchange")}, Probes: probes,
		Args:      []*string{to.Ptr("--addr=0.0.0.0:8043"), to.Ptr("--public-origin=" + b.Origin), to.Ptr("--backend-origin=" + b.BackendOrigin), to.Ptr("--backend-address=" + b.BackendAddress)},
		Resources: &a.ContainerResources{CPU: to.Ptr(0.5), Memory: to.Ptr("1Gi")},
	}
	template := &a.Template{
		TerminationGracePeriodSeconds: to.Ptr[int64](30), Containers: []*a.Container{container},
		Scale: &a.Scale{MinReplicas: to.Ptr[int32](1), MaxReplicas: to.Ptr[int32](2), CooldownPeriod: to.Ptr[int32](300), PollingInterval: to.Ptr[int32](30)},
	}
	return a.ContainerApp{
		Location:   to.Ptr(b.Region),
		Identity:   &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{b.PullIdentity: {}}},
		Properties: &a.ContainerAppProperties{ManagedEnvironmentID: to.Ptr(b.Environment), WorkloadProfileName: to.Ptr("Consumption"), Configuration: configuration, Template: template},
	}

}

func (s ExchangeSettings) AuthDefinition(c Config) (a.AuthConfig, error) {
	if _, err := s.canonical(c); err != nil {
		return a.AuthConfig{}, err
	}
	return a.AuthConfig{Properties: &a.AuthConfigProperties{Platform: &a.AuthPlatform{Enabled: to.Ptr(false)}}}, nil
}
