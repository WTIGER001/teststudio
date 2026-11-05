package azuresandbox

import (
	"encoding/json"
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

func (s Settings) hostDefinition() a.ContainerApp {
	h := s.Host
	pins := s.hostWorkloadBinding().Identities
	identities := map[string]*a.UserAssignedIdentity{}
	lifecycle := []*a.IdentitySettings{}
	ids := []string{}
	for id := range pins {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		identities[id] = &a.UserAssignedIdentity{}
		lifecycle = append(lifecycle, &a.IdentitySettings{Identity: to.Ptr(id), Lifecycle: to.Ptr(a.IdentitySettingsLifeCycle(pins[id].Lifecycle))})
	}
	command := "/app/teststudio-prepare"
	args := []string{"--discovery-only", "--execution-location=azure-container-apps", "--database-env=TESTSTUDIO_DATABASE", "--database-host=" + h.DatabaseHost, "--require-postgres=true", "--azure-sandbox-env=TESTSTUDIO_SANDBOX"}
	env := []*a.EnvironmentVar{{Name: to.Ptr("TESTSTUDIO_HOST_PROFILE"), Value: to.Ptr(s.Host.Profile)}, {Name: to.Ptr("TESTSTUDIO_DATABASE"), SecretRef: to.Ptr("database")}, {Name: to.Ptr("TESTSTUDIO_SANDBOX"), SecretRef: to.Ptr("sandbox")}}
	probes := []*a.ContainerAppProbe{}
	if s.Mode == "execution" {
		command = "/app/teststudio"
		args = []string{"--serve-role=controller", "--addr=0.0.0.0:8043", "--execution-location=azure-container-apps", "--database-env=TESTSTUDIO_DATABASE", "--database-host=" + h.DatabaseHost, "--require-postgres=true", "--azure-sandbox-env=TESTSTUDIO_SANDBOX", "--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION", "--public-url=" + h.PublicOrigin}
		artifact, _ := json.Marshal(map[string]any{"driver": "azure", "azure": map[string]any{"serviceUrl": h.BlobServiceURL, "container": h.EvidenceContainer, "prefix": "teststudio", "managedIdentityClientId": h.Runtime.Client}})
		env = append(env, &a.EnvironmentVar{Name: to.Ptr("TESTSTUDIO_ARTIFACT_CONFIGURATION"), Value: to.Ptr(string(artifact))})
		for _, probe := range []struct {
			kind             a.Type
			path             string
			period, failures int32
		}{{a.TypeStartup, "/healthz", 3, 30}, {a.TypeReadiness, "/healthz", 10, 3}, {a.TypeLiveness, "/livez", 30, 3}} {
			value := &a.ContainerAppProbe{Type: to.Ptr(probe.kind), HTTPGet: &a.ContainerAppProbeHTTPGet{Path: to.Ptr(probe.path), Port: to.Ptr[int32](8043), Scheme: to.Ptr(a.SchemeHTTP)}, TimeoutSeconds: to.Ptr[int32](2), PeriodSeconds: to.Ptr(probe.period), FailureThreshold: to.Ptr(probe.failures)}
			if probe.kind == a.TypeStartup {
				value.InitialDelaySeconds = to.Ptr[int32](1)
			}
			probes = append(probes, value)
		}
	}
	arguments := []*string{}
	for _, arg := range args {
		arguments = append(arguments, to.Ptr(arg))
	}
	return a.ContainerApp{Location: to.Ptr(s.Worker.Region), Identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: identities}, Properties: &a.ContainerAppProperties{ManagedEnvironmentID: to.Ptr(h.Environment), WorkloadProfileName: to.Ptr("Consumption"), Configuration: &a.Configuration{ActiveRevisionsMode: to.Ptr(a.ActiveRevisionsModeSingle), MaxInactiveRevisions: to.Ptr[int32](1), IdentitySettings: lifecycle, Registries: []*a.RegistryCredentials{{Server: to.Ptr(s.Worker.Registry), Identity: to.Ptr(h.Pull.Resource)}}, Secrets: []*a.Secret{{Name: to.Ptr("database"), KeyVaultURL: to.Ptr(h.DatabaseSecretURL), Identity: to.Ptr(h.Startup.Resource)}, {Name: to.Ptr("sandbox"), KeyVaultURL: to.Ptr(h.ConfigurationSecretURL), Identity: to.Ptr(h.Startup.Resource)}}}, Template: &a.Template{TerminationGracePeriodSeconds: to.Ptr[int64](90), Containers: []*a.Container{{Name: to.Ptr("controller"), Image: to.Ptr(h.Image), Command: []*string{to.Ptr(command)}, Args: arguments, Env: env, Probes: probes, Resources: &a.ContainerResources{CPU: to.Ptr(0.5), Memory: to.Ptr("1Gi")}}}, Scale: &a.Scale{MinReplicas: to.Ptr(h.Replicas), MaxReplicas: to.Ptr(h.Replicas)}}}}
}

func (s Settings) HostDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	h := s.Host
	return map[string]any{h.Application: s.hostDefinition(), h.Application + "/authConfigs/current": a.AuthConfig{Properties: &a.AuthConfigProperties{Platform: &a.AuthPlatform{Enabled: to.Ptr(false)}}}}, nil
}
