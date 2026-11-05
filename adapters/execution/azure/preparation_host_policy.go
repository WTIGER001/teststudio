package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s PreparationSettings) PreparationApplicationDefinition() (a.ContainerApp, error) {
	if s.validateHost() != nil {
		return a.ContainerApp{}, hostedexecution.ErrConfiguration
	}
	c, h := s.Worker, *s.Host
	identities := map[string]*a.UserAssignedIdentity{}
	lifecycle := []*a.IdentitySettings{}
	for i, id := range h.identities(s.Writer) {
		identities[id] = &a.UserAssignedIdentity{}
		life := a.IdentitySettingsLifeCycleNone
		if i < 2 {
			life = a.IdentitySettingsLifeCycleMain
		}
		lifecycle = append(lifecycle, &a.IdentitySettings{Identity: to.Ptr(id), Lifecycle: &life})
	}
	args := []*string{to.Ptr("--database-env=TESTSTUDIO_DATABASE"), to.Ptr("--require-postgres=true"), to.Ptr("--assembly-only=true"), to.Ptr("--execution-location=" + Location), to.Ptr("--azure-assembly-env=TESTSTUDIO_AZURE_ASSEMBLY"), to.Ptr("--build-directory=" + PreparationJournalDirectory)}
	if h.GoRuntime != "" {
		args = append(args, to.Ptr("--registry-go-runtime="+h.GoRuntime))
	}
	if h.PlaywrightRuntime != "" {
		args = append(args, to.Ptr("--registry-playwright-runtime="+h.PlaywrightRuntime))
	}
	return a.ContainerApp{Location: to.Ptr(c.Region), Identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: identities}, Properties: &a.ContainerAppProperties{
		ManagedEnvironmentID: to.Ptr(h.Environment), WorkloadProfileName: to.Ptr("Consumption"),
		Configuration: &a.Configuration{ActiveRevisionsMode: to.Ptr(a.ActiveRevisionsModeSingle), IdentitySettings: lifecycle, Registries: []*a.RegistryCredentials{{Server: to.Ptr(c.Registry), Identity: to.Ptr(h.PullIdentity)}}, Secrets: []*a.Secret{
			{Name: to.Ptr("database"), KeyVaultURL: to.Ptr(h.DatabaseSecretURL), Identity: to.Ptr(h.StartupIdentity)},
			{Name: to.Ptr("assembly-configuration"), KeyVaultURL: to.Ptr(h.ConfigurationSecretURL), Identity: to.Ptr(h.StartupIdentity)},
		}},
		Template: &a.Template{TerminationGracePeriodSeconds: to.Ptr[int64](90), Containers: []*a.Container{{Name: to.Ptr("preparation"), Image: to.Ptr(h.Image), Command: []*string{to.Ptr("/app/teststudio-prepare")}, Args: args, Env: []*a.EnvironmentVar{{Name: to.Ptr("TESTSTUDIO_DATABASE"), SecretRef: to.Ptr("database")}, {Name: to.Ptr("TESTSTUDIO_AZURE_ASSEMBLY"), SecretRef: to.Ptr("assembly-configuration")}}, Resources: &a.ContainerResources{CPU: to.Ptr(2.0), Memory: to.Ptr("4Gi")}, VolumeMounts: []*a.VolumeMount{{VolumeName: to.Ptr("source-journals"), MountPath: to.Ptr(PreparationJournalDirectory)}}}}, Volumes: []*a.Volume{{Name: to.Ptr("source-journals"), StorageType: to.Ptr(a.StorageTypeAzureFile), StorageName: to.Ptr(h.JournalStorageName), MountOptions: to.Ptr(PreparationJournalMountOptions)}}, Scale: &a.Scale{MinReplicas: to.Ptr[int32](1), MaxReplicas: to.Ptr[int32](1)}},
	}}, nil
}
