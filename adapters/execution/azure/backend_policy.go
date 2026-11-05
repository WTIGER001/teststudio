package azureworker

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

func (s BackendSettings) artifactConfiguration() map[string]any {
	return map[string]any{"driver": "azure", "azure": map[string]any{"serviceUrl": s.BlobServiceURL, "container": s.EvidenceContainer, "prefix": "teststudio", "managedIdentityClientId": s.RuntimeClientID}}
}

func (s BackendSettings) ApplicationDefinition(c Config) (a.ContainerApp, error) {
	s, err := s.canonical(c)
	if err != nil {
		return a.ContainerApp{}, err
	}
	return BackendApplicationDefinition(c.Region, c.Registry, s), nil
}

func BackendApplicationDefinition(region, registry string, s BackendSettings) a.ContainerApp {
	artifact, _ := json.Marshal(s.artifactConfiguration())
	probes := []*a.ContainerAppProbe{}
	for _, p := range []struct {
		kind             a.Type
		path             string
		period, failures int32
	}{{a.TypeStartup, "/healthz", 3, 30}, {a.TypeReadiness, "/healthz", 10, 3}, {a.TypeLiveness, "/livez", 30, 3}} {
		probe := &a.ContainerAppProbe{Type: to.Ptr(p.kind), HTTPGet: &a.ContainerAppProbeHTTPGet{Path: to.Ptr(p.path), Port: to.Ptr[int32](8042), Scheme: to.Ptr(a.SchemeHTTP)}, PeriodSeconds: to.Ptr(p.period), TimeoutSeconds: to.Ptr[int32](2), FailureThreshold: to.Ptr(p.failures)}
		if p.kind == a.TypeStartup {
			probe.InitialDelaySeconds = to.Ptr[int32](1)
		}
		probes = append(probes, probe)
	}
	return a.ContainerApp{
		Location: to.Ptr(region),
		Identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{s.PullIdentity: {}, s.StartupIdentity: {}, s.RuntimeIdentity: {}}},
		Properties: &a.ContainerAppProperties{ManagedEnvironmentID: to.Ptr(s.Environment), WorkloadProfileName: to.Ptr("Consumption"),
			Configuration: &a.Configuration{
				ActiveRevisionsMode: to.Ptr(a.ActiveRevisionsModeSingle), MaxInactiveRevisions: to.Ptr[int32](1),
				IdentitySettings: []*a.IdentitySettings{{Identity: to.Ptr(s.PullIdentity), Lifecycle: to.Ptr(a.IdentitySettingsLifeCycleNone)}, {Identity: to.Ptr(s.StartupIdentity), Lifecycle: to.Ptr(a.IdentitySettingsLifeCycleNone)}, {Identity: to.Ptr(s.RuntimeIdentity), Lifecycle: to.Ptr(a.IdentitySettingsLifeCycleMain)}},
				Registries:       []*a.RegistryCredentials{{Server: to.Ptr(registry), Identity: to.Ptr(s.PullIdentity)}},
				Secrets:          []*a.Secret{{Name: to.Ptr("database"), KeyVaultURL: to.Ptr(s.DatabaseSecretURL), Identity: to.Ptr(s.StartupIdentity)}},
				Ingress:          &a.Ingress{External: to.Ptr(true), AllowInsecure: to.Ptr(false), TargetPort: to.Ptr[int32](8042), Transport: to.Ptr(a.IngressTransportMethodHTTP), Traffic: []*a.TrafficWeight{{LatestRevision: to.Ptr(true), Weight: to.Ptr[int32](100)}}},
			},
			Template: &a.Template{TerminationGracePeriodSeconds: to.Ptr[int64](30),
				Containers: []*a.Container{{Name: to.Ptr("worker-backend"), Image: to.Ptr(s.Image), Command: []*string{to.Ptr("/app/teststudio")},
					Args:      []*string{to.Ptr("--serve-role=worker"), to.Ptr("--worker-addr=0.0.0.0:8042"), to.Ptr("--database-env=TESTSTUDIO_DATABASE"), to.Ptr("--require-postgres=true"), to.Ptr("--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION"), to.Ptr("--execution-location=azure-container-apps")},
					Env:       []*a.EnvironmentVar{{Name: to.Ptr("TESTSTUDIO_DATABASE"), SecretRef: to.Ptr("database")}, {Name: to.Ptr("TESTSTUDIO_ARTIFACT_CONFIGURATION"), Value: to.Ptr(string(artifact))}},
					Resources: &a.ContainerResources{CPU: to.Ptr(0.5), Memory: to.Ptr("1Gi")}, Probes: probes,
				}}, Scale: &a.Scale{MinReplicas: to.Ptr(s.Replicas), MaxReplicas: to.Ptr(s.Replicas)},
			},
		},
	}
}
