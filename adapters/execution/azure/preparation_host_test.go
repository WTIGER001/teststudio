package azureworker

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePreparationHostBindsFreshPlacementRuntimeAndReadGrant(t *testing.T) {
	s, f, g := preparationHostTest(t)
	s.Host.GoRuntime = "changed-caller-runtime"
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), s.Worker); err != nil || f.reads != 7*i || f.parent.reads != 3*i {
			t.Fatal("host observation was rejected, cached, or repeated its parent", err, f.reads, f.parent.reads)
		}
	}
	reader := g.reader.(*SDKPreparationHostReader)
	storage, err := reader.Storage(t.Context(), g.settings.Host.storage())
	data, _ := json.Marshal(storage)
	if err != nil || strings.Contains(string(data), "private-journal-key-canary") || storage.Properties.AzureFile.AccountKey != nil || storage.Properties.AzureFile.AccountKeyVaultProperties != nil {
		t.Fatal("storage observation exposed platform credentials", err)
	}
	f.role.Properties.Permissions[0].Actions = append(f.role.Properties.Permissions[0].Actions, to.Ptr("Microsoft.App/jobs/write"))
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrOwnership)
	if _, ok := any(g).(Network); ok {
		t.Fatal("trusted assembly placement became untrusted execution admission")
	}
}

func TestAzurePreparationHostRejectsSynchronizedRuntimeAndMountDrift(t *testing.T) {
	for name, mutate := range map[string]func(*preparationHostFixture){
		"command": func(f *preparationHostFixture) {
			f.application.Properties.Template.Containers[0].Command = []*string{to.Ptr("/app/teststudio")}
		},
		"extra-mode": func(f *preparationHostFixture) {
			f.application.Properties.Template.Containers[0].Args = append(f.application.Properties.Template.Containers[0].Args, to.Ptr("--github-sources=true"))
		},
		"ephemeral-journal": func(f *preparationHostFixture) {
			f.application.Properties.Template.Volumes[0].StorageType = to.Ptr(a.StorageTypeEmptyDir)
		},
		"public-mount": func(f *preparationHostFixture) {
			f.application.Properties.Template.Volumes[0].MountOptions = to.Ptr("dir_mode=0777,file_mode=0666,uid=65532,gid=65532")
		},
		"wrong-path": func(f *preparationHostFixture) {
			f.application.Properties.Template.Containers[0].VolumeMounts[0].MountPath = to.Ptr("/tmp/journals")
		},
		"vault-reference": func(f *preparationHostFixture) {
			f.application.Properties.Configuration.Secrets[0].KeyVaultURL = to.Ptr("https://foreignvault.vault.azure.net/secrets/database")
		},
		"ingress": func(f *preparationHostFixture) {
			f.application.Properties.Configuration.Ingress = &a.Ingress{External: to.Ptr(true)}
		},
		"sidecar": func(f *preparationHostFixture) {
			f.application.Properties.Template.Containers = append(f.application.Properties.Template.Containers, &a.Container{Name: to.Ptr("foreign")})
		},
		"startup-main": func(f *preparationHostFixture) {
			f.application.Properties.Configuration.IdentitySettings[3].Lifecycle = to.Ptr(a.IdentitySettingsLifeCycleMain)
		},
		"reader-client": func(f *preparationHostFixture) {
			f.application.Identity.UserAssignedIdentities[f.settings.Host.ReaderIdentity].ClientID = to.Ptr("cccccccc-aaaa-bbbb-cccc-111111111111")
		},
		"worker-environment": func(f *preparationHostFixture) {
			f.application.Properties.ManagedEnvironmentID = to.Ptr(f.settings.Worker.WorkerEnvironment)
		},
		"environment-subnet": func(f *preparationHostFixture) {
			f.environment.Properties.VnetConfiguration.InfrastructureSubnetID = to.Ptr(f.settings.Writer.Registry.Subnet)
		},
		"public-environment": func(f *preparationHostFixture) {
			f.environment.Properties.PublicNetworkAccess = to.Ptr(a.PublicNetworkAccessEnabled)
		},
		"read-only-share": func(f *preparationHostFixture) {
			f.storage.Properties.AzureFile.AccessMode = to.Ptr(a.AccessModeReadOnly)
		},
		"nfs-share": func(f *preparationHostFixture) {
			f.storage.Properties.NfsAzureFile = &a.NfsAzureFileProperties{}
			f.storage.Properties.AzureFile = nil
		},
		"other-share":        func(f *preparationHostFixture) { f.storage.Properties.AzureFile.ShareName = to.Ptr("other-journals") },
		"reader-grant-scope": func(f *preparationHostFixture) { f.assignment.Properties.Scope = to.Ptr(f.settings.Host.Environment) },
	} {
		t.Run(name, func(t *testing.T) {
			s, f, g := preparationHostTest(t)
			mutate(f)
			f.revisions[0].Properties.Template = clone(f.application.Properties.Template)
			requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzurePreparationHostPreservesUnknownMetadataAndExactSDKPaths(t *testing.T) {
	s, f, g := preparationHostTest(t)
	reader := g.reader.(*SDKPreparationHostReader)
	for _, mode := range []string{"unavailable", "redirect", "missing-revisions", "incomplete-revisions"} {
		f.mode = mode
		want := hostedexecution.ErrUnavailable
		if mode == "incomplete-revisions" {
			want = hostedexecution.ErrOwnership
		}
		requireDeadlineError(t, g.Check(t.Context(), s.Worker), want)
	}
	f.mode = ""
	f.storage.Properties.AzureFile.AccessMode = nil
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrUnavailable)
	f.storage.Properties.AzureFile.AccessMode = to.Ptr(a.AccessModeReadWrite)
	f.revisions[0].Properties.HealthState = nil
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrUnavailable)
	before := f.reads
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Path = s.Host.Application + "/authConfigs/current" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$expand=secrets" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Host.storage()+"?api-version="+exchangeVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.Storage(t.Context(), s.Worker.ControllerEnvironment+"/storages/foreign")
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("unselected host operation reached transport")
	}
}

func TestAzurePreparationHostPublisherGatesCredentialsAndOwnsAllReaders(t *testing.T) {
	s, f, g := preparationHostTest(t)
	transport := preparationPublisherTransport(t, g.writer)
	for id := range g.reader.(*SDKPreparationHostReader).http.resources {
		transport.routes[id] = f
	}
	publisher, err := s.OpenPublisher(fixtureCredential{}, transport)
	if err != nil || len(publisher.readers) != 11 || transport.reads != 0 {
		t.Fatal("host publisher lost ownership or made constructor requests", err)
	}
	t.Cleanup(publisher.Close)
	credentials := 0
	publisher.guard.credential = func(id string) (azcore.TokenCredential, error) {
		credentials++
		if id != s.Writer.ClientID {
			t.Fatal("host admission selected another writer")
		}
		return fixtureCredential{}, nil
	}
	client, err := publisher.OpenRegistry(t.Context(), s.Worker)
	if err != nil || client == nil || credentials != 1 || f.reads != 7 || f.parent.reads != 3 {
		t.Fatal("bound host was not checked exactly once before credential selection", err, credentials, f.reads, f.parent.reads)
	}
	client.Close()
	f.storage.Properties.AzureFile.AccessMode = to.Ptr(a.AccessModeReadOnly)
	_, err = publisher.OpenRegistry(t.Context(), s.Worker)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if credentials != 1 {
		t.Fatal("changed journal storage reached writer credential selection")
	}
	publisher.Close()
	publisher.Close()
	if transport.closed != 11 {
		t.Fatal("host publisher did not close every owned reader once", transport.closed)
	}
}
