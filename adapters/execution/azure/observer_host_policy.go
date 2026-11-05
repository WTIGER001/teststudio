package azureworker

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

func (s ObserverHostSettings) ApplicationDefinition(c Config) (a.ContainerApp, error) {
	s, err := s.canonical(c)
	if err != nil {
		return a.ContainerApp{}, err
	}
	r := s.Runtime
	artifact, _ := json.Marshal(map[string]any{"driver": "azure", "azure": map[string]any{"serviceUrl": r.BlobServiceURL, "container": r.EvidenceContainer, "prefix": "teststudio", "managedIdentityClientId": r.RuntimeClientID}})
	secrets := []*a.Secret{{Name: to.Ptr("database"), KeyVaultURL: to.Ptr(r.DatabaseSecretURL), Identity: to.Ptr(s.StartupIdentity)}, {Name: to.Ptr("oidc-configuration"), KeyVaultURL: to.Ptr(r.OIDCConfigurationSecretURL), Identity: to.Ptr(s.StartupIdentity)}}
	env := []*a.EnvironmentVar{{Name: to.Ptr("TESTSTUDIO_DATABASE"), SecretRef: to.Ptr("database")}, {Name: to.Ptr("TESTSTUDIO_ARTIFACT_CONFIGURATION"), Value: to.Ptr(string(artifact))}, {Name: to.Ptr("TESTSTUDIO_OIDC_CONFIGURATION"), SecretRef: to.Ptr("oidc-configuration")}}
	refs := []string{}
	for i, secret := range []string{r.OIDCClientOneSecretURL, r.OIDCClientTwoSecretURL} {
		if secret == "" {
			continue
		}
		suffix := []string{"one", "two"}[i]
		name, environment := "oidc-client-"+suffix, "TESTSTUDIO_OIDC_CLIENT_"+strings.ToUpper(suffix)
		secrets = append(secrets, &a.Secret{Name: to.Ptr(name), KeyVaultURL: to.Ptr(secret), Identity: to.Ptr(s.StartupIdentity)})
		env = append(env, &a.EnvironmentVar{Name: to.Ptr(environment), SecretRef: to.Ptr(name)})
		refs = append(refs, "env:"+environment)
	}
	args := []string{"--serve-role=api", "--require-postgres=true", "--public-url=" + r.UIOrigin, "--artifact-origin=" + r.ArtifactOrigin, "--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION", "--execution-location=azure-container-apps", "--oidc-config-env=TESTSTUDIO_OIDC_CONFIGURATION", "--oidc-secret-refs=" + strings.Join(refs, ","), "--bootstrap-issuer=" + r.BootstrapIssuer, "--bootstrap-subject=" + r.BootstrapSubject, "--local-accounts=" + strconv.FormatBool(r.LocalAccounts)}
	arguments := make([]*string, len(args))
	for i := range args {
		arguments[i] = to.Ptr(args[i])
	}
	identities, lifecycle := map[string]*a.UserAssignedIdentity{}, []*a.IdentitySettings{}
	for _, id := range []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity, s.Identity.Identity} {
		identities[id] = &a.UserAssignedIdentity{}
		life := a.IdentitySettingsLifeCycle(s.lifecycles()[inventoryIdentityKey(id)])
		lifecycle = append(lifecycle, &a.IdentitySettings{Identity: to.Ptr(id), Lifecycle: &life})
	}
	return a.ContainerApp{Location: to.Ptr(c.Region), Identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: identities}, Properties: &a.ContainerAppProperties{ManagedEnvironmentID: to.Ptr(c.ControllerEnvironment), Configuration: &a.Configuration{
		ActiveRevisionsMode: to.Ptr(a.ActiveRevisionsModeSingle), IdentitySettings: lifecycle,
		Registries: []*a.RegistryCredentials{{Server: to.Ptr(c.Registry), Identity: to.Ptr(s.PullIdentity)}}, Secrets: secrets,
		Ingress: &a.Ingress{External: to.Ptr(true), AllowInsecure: to.Ptr(false), TargetPort: to.Ptr[int32](8040), Transport: to.Ptr(a.IngressTransportMethodHTTP), Traffic: []*a.TrafficWeight{{LatestRevision: to.Ptr(true), Weight: to.Ptr[int32](100)}}},
	}, Template: &a.Template{TerminationGracePeriodSeconds: to.Ptr[int64](90), Containers: []*a.Container{{Name: to.Ptr("service"), Image: to.Ptr(s.Image), Args: arguments, Env: env, Resources: &a.ContainerResources{CPU: to.Ptr(0.5), Memory: to.Ptr("1Gi")}, Probes: observerHostProbes()}}, Scale: &a.Scale{MinReplicas: to.Ptr(r.Replicas), MaxReplicas: to.Ptr(r.Replicas)}}}}, nil
}

func observerHostProbes() []*a.ContainerAppProbe {
	probes := []*a.ContainerAppProbe{}
	for _, p := range []struct {
		kind             a.Type
		path             string
		period, failures int32
	}{{a.TypeStartup, "/healthz", 3, 30}, {a.TypeReadiness, "/healthz", 10, 3}, {a.TypeLiveness, "/livez", 30, 3}} {
		probe := &a.ContainerAppProbe{Type: to.Ptr(p.kind), HTTPGet: &a.ContainerAppProbeHTTPGet{Path: to.Ptr(p.path), Port: to.Ptr[int32](8040), Scheme: to.Ptr(a.SchemeHTTP)}, PeriodSeconds: to.Ptr(p.period), TimeoutSeconds: to.Ptr[int32](2), FailureThreshold: to.Ptr(p.failures)}
		if p.kind == a.TypeStartup {
			probe.InitialDelaySeconds = to.Ptr[int32](1)
		}
		probes = append(probes, probe)
	}
	return probes
}
