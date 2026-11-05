package azureworker

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

type observerHostFixture struct {
	t           *testing.T
	mu          sync.Mutex
	parent      *observerFixture
	settings    ObserverHostSettings
	application a.ContainerApp
	revisions   []a.Revision
	reads       int
	mode        string
}

func observerHostTest(t *testing.T) (Config, ObserverHostSettings, *observerHostFixture, *ReadIdentityGuard) {
	t.Helper()
	c, identity, parent, g := observerTest(t)
	s := ObserverHostSettings{Identity: identity, Application: c.resourceID("Microsoft.App", "containerApps", "studio-api"), Image: c.Registry + "/services/teststudio@sha256:" + strings.Repeat("e", 64), PullIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "api-pull"), StartupIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "api-startup"), RuntimeIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "api-runtime")}
	f := &observerHostFixture{t: t, parent: parent, settings: s}
	_, name := resourceParts(s.Application)
	f.application = a.ContainerApp{
		ID: to.Ptr(s.Application), Name: to.Ptr(name), Type: to.Ptr("Microsoft.App/containerApps"), Location: to.Ptr(c.Region),
		Identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{}},
		Properties: &a.ContainerAppProperties{
			ManagedEnvironmentID: to.Ptr(c.ControllerEnvironment), ProvisioningState: to.Ptr(a.ContainerAppProvisioningStateSucceeded), RunningStatus: to.Ptr(a.ContainerAppRunningStatusRunning),
			LatestRevisionName: to.Ptr(name + "--current"), LatestReadyRevisionName: to.Ptr(name + "--current"),
			Configuration: &a.Configuration{ActiveRevisionsMode: to.Ptr(a.ActiveRevisionsModeSingle)},
			Template:      &a.Template{Containers: []*a.Container{{Name: to.Ptr("service"), Image: to.Ptr(s.Image), Args: []*string{to.Ptr("--serve-role=api")}}}},
		},
	}
	for i, id := range []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity, s.Identity.Identity} {
		client := fmt.Sprintf("66666666-aaaa-bbbb-cccc-11111111111%d", i)
		principal := fmt.Sprintf("77777777-aaaa-bbbb-cccc-11111111111%d", i)
		if id == s.Identity.Identity {
			client, principal = s.Identity.ClientID, s.Identity.PrincipalID
		}
		f.application.Identity.UserAssignedIdentities[id] = &a.UserAssignedIdentity{ClientID: &client, PrincipalID: &principal}
		lifecycle := a.IdentitySettingsLifeCycle(s.lifecycles()[inventoryIdentityKey(id)])
		f.application.Properties.Configuration.IdentitySettings = append(f.application.Properties.Configuration.IdentitySettings, &a.IdentitySettings{Identity: to.Ptr(id), Lifecycle: &lifecycle})
	}
	f.revisions = []a.Revision{{ID: to.Ptr(s.Application + "/revisions/" + name + "--current"), Name: to.Ptr(name + "--current"), Type: to.Ptr("Microsoft.App/containerApps/revisions"), Properties: &a.RevisionProperties{Active: to.Ptr(true), ProvisioningState: to.Ptr(a.RevisionProvisioningStateProvisioned), HealthState: to.Ptr(a.RevisionHealthStateHealthy), RunningState: to.Ptr(a.RevisionRunningStateRunning), Replicas: to.Ptr[int32](1), TrafficWeight: to.Ptr[int32](100), Template: clone(f.application.Properties.Template)}}}
	parent.workloads.apps = append(parent.workloads.apps, to.Ptr(clone(f.application)))
	for _, app := range parent.workloads.apps {
		if app.Identity == nil {
			app.Identity = &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeNone)}
		}
	}
	for _, job := range parent.workloads.jobs {
		if job.Identity == nil {
			job.Identity = &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeNone)}
		}
	}
	return c, s, f, g
}

func (f *observerHostFixture) Do(r *http.Request) (*http.Response, error) {
	if r.URL.Path != f.settings.Application && r.URL.Path != f.settings.Application+"/revisions" {
		return f.parent.Do(r)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version="+exchangeVersion || r.Header.Get("Authorization") != "Bearer observer-credential-canary" {
		f.t.Error("host attachment reader widened its endpoint or identity")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-observer-host-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	var body any = f.application
	if r.URL.Path == f.settings.Application+"/revisions" {
		body = map[string]any{"value": f.revisions}
		if f.mode == "missing-revisions" {
			body = map[string]any{}
		}
	}
	return deadlineResponse(r, 200, body), nil
}

func (f *observerHostFixture) CloseIdleConnections() { f.parent.CloseIdleConnections() }
