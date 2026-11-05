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
	environment string
	settings    ObserverHostSettings
	application a.ContainerApp
	auth        a.AuthConfig
	authMode    string
	routesMode  string
	routeReads  int
	revisions   []a.Revision
	reads       int
	mode        string
}

func observerHostTest(t *testing.T) (Config, ObserverHostSettings, *observerHostFixture, *ReadIdentityGuard) {
	t.Helper()
	c, identity, parent, g := observerTest(t)
	s := ObserverHostSettings{Identity: identity, Application: c.resourceID("Microsoft.App", "containerApps", "studio-api"), Image: c.Registry + "/services/teststudio@sha256:" + strings.Repeat("e", 64), PullIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "api-pull"), StartupIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "api-startup"), RuntimeIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "api-runtime")}
	s.Runtime = ObserverHostRuntime{RuntimeClientID: "66666666-aaaa-bbbb-cccc-111111111112", DatabaseSecretURL: "https://studiovault.vault.azure.net/secrets/api-database", OIDCConfigurationSecretURL: "https://studiovault.vault.azure.net/secrets/oidc-configuration", OIDCClientOneSecretURL: "https://studiovault.vault.azure.net/secrets/oidc-client-one", OIDCClientTwoSecretURL: "https://studiovault.vault.azure.net/secrets/oidc-client-two", BlobServiceURL: "https://studioevidence.blob.core.windows.net/", EvidenceContainer: "evidence", UIOrigin: "https://studio-api.controllerzone.eastus.azurecontainerapps.io", ArtifactOrigin: "https://studio-artifacts.controllerzone.eastus.azurecontainerapps.io", BootstrapIssuer: "https://login.example.test/tenant", BootstrapSubject: "teststudio-owner", Replicas: 1}
	f := &observerHostFixture{t: t, parent: parent, settings: s, environment: c.ControllerEnvironment}
	_, name := resourceParts(s.Application)
	var err error
	f.application, err = s.ApplicationDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	f.application.ID, f.application.Name, f.application.Type = to.Ptr(s.Application), to.Ptr(name), to.Ptr("Microsoft.App/containerApps")
	f.application.Properties.ProvisioningState = to.Ptr(a.ContainerAppProvisioningStateSucceeded)
	f.application.Properties.RunningStatus = to.Ptr(a.ContainerAppRunningStatusRunning)
	f.application.Properties.LatestRevisionName, f.application.Properties.LatestReadyRevisionName = to.Ptr(name+"--current"), to.Ptr(name+"--current")
	f.application.Properties.Configuration.Ingress.Fqdn = to.Ptr(strings.TrimPrefix(s.Runtime.UIOrigin, "https://"))

	for i, id := range []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity, s.Identity.Identity} {
		client := fmt.Sprintf("66666666-aaaa-bbbb-cccc-11111111111%d", i)
		principal := fmt.Sprintf("77777777-aaaa-bbbb-cccc-11111111111%d", i)
		if id == s.Identity.Identity {
			client, principal = s.Identity.ClientID, s.Identity.PrincipalID
		}
		f.application.Identity.UserAssignedIdentities[id] = &a.UserAssignedIdentity{ClientID: &client, PrincipalID: &principal}
	}
	f.revisions = []a.Revision{{ID: to.Ptr(s.Application + "/revisions/" + name + "--current"), Name: to.Ptr(name + "--current"), Type: to.Ptr("Microsoft.App/containerApps/revisions"), Properties: &a.RevisionProperties{Active: to.Ptr(true), ProvisioningState: to.Ptr(a.RevisionProvisioningStateProvisioned), HealthState: to.Ptr(a.RevisionHealthStateHealthy), RunningState: to.Ptr(a.RevisionRunningStateRunning), Replicas: to.Ptr[int32](1), TrafficWeight: to.Ptr[int32](100), Template: clone(f.application.Properties.Template)}}}
	f.auth, err = s.AuthDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	f.auth.ID, f.auth.Name, f.auth.Type = to.Ptr(s.Application+"/authConfigs/current"), to.Ptr("current"), to.Ptr("Microsoft.App/containerApps/authConfigs")
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
	if r.URL.Path != f.settings.Application && r.URL.Path != f.settings.Application+"/revisions" && r.URL.Path != f.settings.Application+"/authConfigs/current" && r.URL.Path != f.environment+"/httpRouteConfigs" {
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
	if r.URL.Path == f.environment+"/httpRouteConfigs" {
		f.routeReads++
		switch f.routesMode {
		case "missing-array":
			return deadlineResponse(r, 200, map[string]any{}), nil
		case "null-array":
			return deadlineResponse(r, 200, map[string]any{"value": nil}), nil
		case "null-row":
			return deadlineResponse(r, 200, map[string]any{"value": []any{nil}}), nil
		case "present":
			return deadlineResponse(r, 200, map[string]any{"value": []any{map[string]any{"name": "private-controller-route-canary"}}}), nil
		case "present-and-next":
			return deadlineResponse(r, 200, map[string]any{"value": []any{map[string]any{}}, "nextLink": "https://foreign.invalid/private-route-canary"}), nil
		case "same-next":
			return deadlineResponse(r, 200, map[string]any{"value": []any{}, "nextLink": "https://management.azure.com" + f.environment + "/httpRouteConfigs?api-version=" + exchangeVersion + "&%24skipToken=private-next-canary"}), nil
		case "foreign-next":
			return deadlineResponse(r, 200, map[string]any{"value": []any{}, "nextLink": "https://foreign.invalid/private-next-canary"}), nil
		case "missing":
			return deadlineResponse(r, 404, nil), nil
		case "unavailable":
			return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"message": "private-route-error-canary"}}), nil
		case "redirect":
			response := deadlineResponse(r, 302, nil)
			response.Header.Set("Location", "https://foreign.invalid/private-route-canary")
			return response, nil
		}
		return deadlineResponse(r, 200, map[string]any{"value": []any{}}), nil
	}
	if r.URL.Path == f.settings.Application+"/authConfigs/current" {
		switch f.authMode {
		case "unavailable":
			return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"message": "private-auth-configuration-canary"}}), nil
		case "missing":
			return deadlineResponse(r, 404, nil), nil
		case "redirect":
			response := deadlineResponse(r, 302, nil)
			response.Header.Set("Location", "https://foreign.invalid/private-auth-canary")
			return response, nil
		}
		return deadlineResponse(r, 200, f.auth), nil
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
