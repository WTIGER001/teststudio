package azureworker

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureBackendObservesFreshWorkerBindingAndSemanticConfiguration(t *testing.T) {
	c, s, f, g := backendServiceTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 5 {
		t.Fatal("selected backend rejected", err, f.reads)
	}
	s.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
	app := f.application.Properties
	slices.Reverse(app.Configuration.IdentitySettings)
	slices.Reverse(app.Template.Containers[0].Probes)
	app.Template.Containers[0].Env[1].Value = to.Ptr(`{ "driver":"azure", "azure": { "prefix":"teststudio", "container":"evidence", "managedIdentityClientId":"77777777-2222-3333-4444-555555555553", "serviceUrl":"https://studioevidence.blob.core.windows.net/" } }`)
	slices.Reverse(app.Template.Containers[0].Env)
	app.Template.Scale.CooldownPeriod = to.Ptr[int32](300)
	app.Template.Scale.PollingInterval = to.Ptr[int32](30)
	for _, probe := range app.Template.Containers[0].Probes {
		probe.SuccessThreshold = to.Ptr[int32](1)
	}
	if err := g.Check(t.Context(), c); err != nil || f.reads != 10 {
		t.Fatal("safe ordering, JSON formatting, or defaults changed authority", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("backend metadata became full network admission")
	}
	before := f.reads
	prior := g.inventory.reader.(*SDKWorkloadInventoryReader).http.base.(*workloadInventoryFixture)
	prior.mode = "missing-value"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("unproven workload inventory reached the backend")
	}
	prior.mode = ""
	other := clone(prior.apps[1])
	other.ID = to.Ptr(c.resourceID("Microsoft.App", "containerApps", "extra-backend-app"))
	other.Name = to.Ptr("extra-backend-app")
	other.Properties.ManagedEnvironmentID = to.Ptr(f.settings.Environment)
	prior.apps = append(prior.apps, other)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("another reported backend occupant reached service inspection")
	}
	prior.apps = prior.apps[:len(prior.apps)-1]
	prior.jobs[0].Properties.EnvironmentID = to.Ptr(f.settings.Environment)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("a reported backend job reached service inspection")
	}
	prior.jobs[0].Properties.EnvironmentID = to.Ptr(c.WorkerEnvironment)
	selected := prior.apps[len(prior.apps)-1]
	prior.apps = prior.apps[:len(prior.apps)-1]
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("an unlisted backend reached service inspection")
	}
	prior.apps = append(prior.apps, selected)
	c.NetworkProfile = "changed"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("changed profile reached backend inspection")
	}
}

func TestAzureBackendRejectsApplicationAuthorityAndPlacementChanges(t *testing.T) {
	for name, mutate := range map[string]func(*backendFixture){
		"api-role": func(f *backendFixture) {
			f.application.Properties.Template.Containers[0].Args[0] = to.Ptr("--serve-role=api")
		},
		"target-bindings": func(f *backendFixture) {
			f.application.Properties.Template.Containers[0].Args = append(f.application.Properties.Template.Containers[0].Args, to.Ptr("--secret-bindings=/private/targets"))
		},
		"plain-database": func(f *backendFixture) {
			f.application.Properties.Configuration.Secrets[0].Value = to.Ptr("private-database-canary")
		},
		"other-database": func(f *backendFixture) {
			f.application.Properties.Configuration.Secrets[0].KeyVaultURL = to.Ptr("https://foreignvault.vault.azure.net/secrets/database")
		},
		"startup-at-runtime": func(f *backendFixture) {
			f.application.Properties.Configuration.IdentitySettings[1].Lifecycle = to.Ptr(a.IdentitySettingsLifeCycleMain)
		},
		"wrong-runtime-client": func(f *backendFixture) {
			f.application.Identity.UserAssignedIdentities[f.settings.RuntimeIdentity].ClientID = to.Ptr("99999999-2222-3333-4444-555555555555")
		},
		"other-artifact-store": func(f *backendFixture) {
			f.application.Properties.Template.Containers[0].Env[1].Value = to.Ptr(`{"driver":"s3","s3":{"bucket":"foreign"}}`)
		},
		"duplicate-artifact-key": func(f *backendFixture) {
			f.application.Properties.Template.Containers[0].Env[1].Value = to.Ptr(`{"driver":"s3","driver":"azure","azure":{}}`)
		},
		"wrong-port": func(f *backendFixture) {
			f.application.Properties.Configuration.Ingress.TargetPort = to.Ptr[int32](8040)
		},
		"insecure-ingress": func(f *backendFixture) { f.application.Properties.Configuration.Ingress.AllowInsecure = to.Ptr(true) },
		"other-environment": func(f *backendFixture) {
			f.application.Properties.ManagedEnvironmentID = to.Ptr(f.settings.Exchange.environment())
		},
		"public-environment": func(f *backendFixture) {
			f.environment.Properties.PublicNetworkAccess = to.Ptr(a.PublicNetworkAccessEnabled)
		},
		"external-load-balancer": func(f *backendFixture) { f.environment.Properties.VnetConfiguration.Internal = to.Ptr(false) },
		"other-subnet": func(f *backendFixture) {
			f.environment.Properties.VnetConfiguration.InfrastructureSubnetID = to.Ptr(f.settings.Exchange.Subnet)
		},
		"foreign-domain": func(f *backendFixture) {
			f.environment.Properties.DefaultDomain = to.Ptr("otherzone.eastus.azurecontainerapps.io")
		},
		"user-auth": func(f *backendFixture) { f.auth.Properties.Platform.Enabled = to.Ptr(true) },
		"changed-active-revision": func(f *backendFixture) {
			f.revisions[0].Properties.Template.Containers[0].Args[0] = to.Ptr("--serve-role=all")
		},
		"retiring-revision-still-running": func(f *backendFixture) {
			old := clone(f.revisions[0])
			old.ID = to.Ptr(f.settings.Application + "/revisions/studio-worker-backend--old")
			old.Name = to.Ptr("studio-worker-backend--old")
			old.Properties.Active = to.Ptr(false)
			old.Properties.TrafficWeight = to.Ptr[int32](0)
			f.revisions = append(f.revisions, old)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := backendServiceTest(t)
			mutate(f)
			err := g.Check(t.Context(), c)
			if err == nil || strings.Contains(err.Error(), "canary") || strings.Contains(err.Error(), "private/") {
				t.Fatal("unsafe or inconsistent backend accepted or disclosed", err)
			}
		})
	}
}

func TestAzureBackendSDKRejectsUnprovenListsAndUnselectedOperations(t *testing.T) {
	c, s, f, g := backendServiceTest(t)
	reader := g.reader.(*SDKBackendReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.Application + "/listSecrets" },
		func(r *http.Request) { r.URL.Path = s.Exchange.Application },
		func(r *http.Request) { r.URL.Path = s.Environment + "/certificates/selected" },
		func(r *http.Request) { r.URL.RawQuery += "&$expand=secrets" },
		func(r *http.Request) { r.URL.RawQuery += "&$filter=active" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Application+"?api-version="+exchangeVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 {
		t.Fatal("unselected operation reached transport")
	}
	for _, mode := range []string{"missing-routes", "null-routes", "missing-revisions", "unavailable", "unfinished", "foreign-page", "extra-route"} {
		f.mode = mode
		want := hostedexecution.ErrUnavailable
		if mode == "foreign-page" || mode == "extra-route" {
			want = hostedexecution.ErrOwnership
		}
		requireDeadlineError(t, g.Check(t.Context(), c), want)
	}
}

func TestAzureBackendSettingsBindDedicatedResourcesAndStoredReferences(t *testing.T) {
	c, s, _, g := backendServiceTest(t)
	for _, mutate := range []func(*BackendSettings){
		func(s *BackendSettings) { s.Environment = c.ControllerEnvironment },
		func(s *BackendSettings) { s.Application = s.Exchange.Application },
		func(s *BackendSettings) { s.Subnet = s.Exchange.Subnet },
		func(s *BackendSettings) { s.RuntimeIdentity = s.StartupIdentity },
		func(s *BackendSettings) { s.PullIdentity = c.PullIdentity },
		func(s *BackendSettings) {
			s.Image = c.Registry + "/" + c.RepositoryPrefix + "/backend@sha256:" + strings.Repeat("a", 64)
		},
		func(s *BackendSettings) { s.DatabaseSecretURL += "?secret=private-canary" },
		func(s *BackendSettings) { s.BlobServiceURL = "https://foreign.invalid/" },
		func(s *BackendSettings) { s.Exchange.BackendOrigin += ":8443" },
		func(s *BackendSettings) { s.Replicas = 11 },
	} {
		changed := s
		mutate(&changed)
		_, err := NewBackendGuard(c, changed, g.inventory, g.reader)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	if _, err := NewBackendGuard(c, s, nil, g.reader); err != hostedexecution.ErrConfiguration {
		t.Fatal("missing inventory accepted")
	}
	if _, err := NewSDKBackendReader(c, s, nil, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("implicit credential discovery accepted")
	}
	first, err := s.ApplicationDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	first.Properties.Template.Containers[0].Args[0] = to.Ptr("--serve-role=api")
	second, _ := s.ApplicationDefinition(c)
	data, _ := json.Marshal(second)
	if value(second.Properties.Template.Containers[0].Args[0]) != "--serve-role=worker" || strings.Contains(string(data), "private-canary") {
		t.Fatal("caller mutation or private settings altered another definition")
	}
}
