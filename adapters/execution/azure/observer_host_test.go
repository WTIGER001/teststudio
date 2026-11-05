package azureworker

import (
	"net/http"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureObserverHostBindsFreshMainIdentityAndHealthyRevision(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrap := f.parent.bootstrapReads
	if f.reads != 0 || f.parent.credential.reads.Load() != 0 || len(observer.readers) != f.parent.bootstrapReaders+2 {
		t.Fatal("host binding performed an observer request or skipped an owned reader")
	}
	for i := 1; i <= 2; i++ {
		slices.Reverse(f.application.Properties.Configuration.IdentitySettings)
		if err := observer.Check(t.Context(), c); err != nil || f.reads != i*4 || f.parent.bootstrapReads != bootstrap || len(f.parent.factoryClients) != 1 {
			t.Fatal("fresh host attachment rejected or bootstrap repeated", err, f.reads)
		}
	}
	f.application.Identity.UserAssignedIdentities[s.Identity.Identity].ClientID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	before := f.reads
	f.parent.unavailable = true
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("unverified coverage reached host attachment reads")
	}
	if _, ok := any(observer).(Network); ok {
		t.Fatal("host attachment enabled native execution admission")
	}
	observer.Close()
	if f.parent.closed != f.parent.bootstrapReaders+2 {
		t.Fatal("host observer did not close all 22 readers", f.parent.closed)
	}
}

func TestAzureObserverHostRejectsExposureMetadataAndPlacementChanges(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	for name, check := range map[string]struct {
		change func(*a.ContainerApp, []a.Revision)
		want   error
	}{
		"worker-placement": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.ManagedEnvironmentID = to.Ptr(c.WorkerEnvironment)
		}, want: hostedexecution.ErrOwnership},
		"legacy-placement": {change: func(app *a.ContainerApp, _ []a.Revision) { app.Properties.EnvironmentID = to.Ptr(c.WorkerEnvironment) }, want: hostedexecution.ErrOwnership},
		"system-identity": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Identity.Type = to.Ptr(a.ManagedServiceIdentityTypeSystemAssignedUserAssigned)
		}, want: hostedexecution.ErrOwnership},
		"missing-identity": {change: func(app *a.ContainerApp, _ []a.Revision) { app.Identity = nil }, want: hostedexecution.ErrUnavailable},
		"observer-client": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Identity.UserAssignedIdentities[s.Identity.Identity].ClientID = to.Ptr(s.Identity.PrincipalID)
		}, want: hostedexecution.ErrOwnership},
		"missing-principal": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Identity.UserAssignedIdentities[s.Identity.Identity].PrincipalID = nil
		}, want: hostedexecution.ErrUnavailable},
		"duplicate-client": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Identity.UserAssignedIdentities[s.RuntimeIdentity].ClientID = to.Ptr(s.Identity.ClientID)
		}, want: hostedexecution.ErrOwnership},
		"observer-unavailable": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Configuration.IdentitySettings[3].Lifecycle = to.Ptr(a.IdentitySettingsLifeCycleNone)
		}, want: hostedexecution.ErrOwnership},
		"exposed-startup": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Configuration.IdentitySettings[1].Lifecycle = to.Ptr(a.IdentitySettingsLifeCycleMain)
		}, want: hostedexecution.ErrOwnership},
		"missing-lifecycle": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Configuration.IdentitySettings[3].Lifecycle = nil
		}, want: hostedexecution.ErrUnavailable},
		"duplicate-lifecycle": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Configuration.IdentitySettings[2] = clone(app.Properties.Configuration.IdentitySettings[3])
		}, want: hostedexecution.ErrOwnership},
		"extra-identity": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Identity.UserAssignedIdentities[c.PullIdentity] = clone(app.Identity.UserAssignedIdentities[s.RuntimeIdentity])
		}, want: hostedexecution.ErrOwnership},
		"image-drift": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Template.Containers[0].Image = to.Ptr("other-image-canary")
		}, want: hostedexecution.ErrOwnership},
		"sidecar": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Template.Containers = append(app.Properties.Template.Containers, clone(app.Properties.Template.Containers[0]))
		}, want: hostedexecution.ErrOwnership},
		"init-container": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Template.InitContainers = []*a.InitContainer{{Name: to.Ptr("extra")}}
		}, want: hostedexecution.ErrOwnership},
		"multi-revision": {change: func(app *a.ContainerApp, _ []a.Revision) {
			app.Properties.Configuration.ActiveRevisionsMode = to.Ptr(a.ActiveRevisionsModeMultiple)
		}, want: hostedexecution.ErrOwnership},
		"unready": {change: func(app *a.ContainerApp, _ []a.Revision) { app.Properties.LatestReadyRevisionName = to.Ptr("previous") }, want: hostedexecution.ErrUnavailable},
		"revision-template": {change: func(_ *a.ContainerApp, revisions []a.Revision) {
			revisions[0].Properties.Template.Containers[0].Image = to.Ptr("other-image-canary")
		}, want: hostedexecution.ErrOwnership},
		"revision-unhealthy": {change: func(_ *a.ContainerApp, revisions []a.Revision) {
			revisions[0].Properties.HealthState = to.Ptr(a.RevisionHealthStateUnhealthy)
		}, want: hostedexecution.ErrUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			app, revisions := clone(f.application), clone(f.revisions)
			check.change(&app, revisions)
			requireReadDenyError(t, verifyObserverHost(c, s, app, revisions), check.want)
		})
	}
	for _, mutate := range []func(*ObserverHostSettings){
		func(s *ObserverHostSettings) { s.PullIdentity = c.PullIdentity },
		func(s *ObserverHostSettings) { s.StartupIdentity = s.Identity.Identity },
		func(s *ObserverHostSettings) {
			s.Application = s.Identity.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Application
		},
		func(s *ObserverHostSettings) { s.Image = c.Registry + "/" + c.RepositoryPrefix + "@sha256:invalid" },
		func(s *ObserverHostSettings) { s.Identity.ClientID = "aaaaaaaa-1111-2222-3333-444444444444" },
	} {
		changed := s
		mutate(&changed)
		_, err := g.BindHostObserver(t.Context(), c, changed, f)
		requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
	}
	if f.parent.bootstrapReads != 0 || len(f.parent.factoryClients) != 0 {
		t.Fatal("invalid host settings reached bootstrap or credential selection")
	}
}

func TestAzureObserverHostSDKRejectsUnselectedReadsAndMissingRevisions(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	reader, err := NewSDKObserverHostReader(c, s, f.parent.credential, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.URL.Path += "/listSecrets" },
		func(r *http.Request) { r.URL.Path = c.ControllerEnvironment },
		func(r *http.Request) { r.URL.Path = s.Application + "/authConfigs/foreign" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Application+"?api-version="+exchangeVersion, nil)
		mutate(request)
		_, err := reader.service.http.Do(request)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 {
		t.Fatal("unselected host operation reached transport")
	}
	for _, mode := range []string{"unavailable", "redirect", "missing-revisions"} {
		f.mode = mode
		_, err := reader.Revisions(t.Context(), s.Application)
		requireReadDenyError(t, err, hostedexecution.ErrUnavailable)
	}
	_, err = NewSDKObserverHostReader(c, s, nil, nil)
	requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
}
