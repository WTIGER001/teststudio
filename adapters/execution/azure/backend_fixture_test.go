package azureworker

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

type backendFixture struct {
	t           *testing.T
	mu          sync.Mutex
	settings    BackendSettings
	environment a.ManagedEnvironment
	application a.ContainerApp
	auth        a.AuthConfig
	revisions   []*a.Revision
	reads       int
	page        int
	mode        string
}

func backendServiceTest(t *testing.T) (Config, BackendSettings, *backendFixture, *BackendGuard) {
	t.Helper()
	c, exchange, _, prior := workloadInventoryTest(t)
	topology := exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology
	s := BackendSettings{Exchange: exchange, Application: c.resourceID("Microsoft.App", "containerApps", "studio-worker-backend"), Environment: c.resourceID("Microsoft.App", "managedEnvironments", "studio-backend"), Subnet: topology.ControllerVNet + "/subnets/backend", InfrastructureGroup: "studio-backend-infra", Image: c.Registry + "/services/teststudio@sha256:" + strings.Repeat("e", 64), PullIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "backend-pull"), StartupIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "backend-startup"), RuntimeIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "backend-runtime"), RuntimeClientID: "77777777-2222-3333-4444-555555555553", DatabaseSecretURL: "https://studiovault.vault.azure.net/secrets/backend-database", BlobServiceURL: "https://studioevidence.blob.core.windows.net/", EvidenceContainer: "evidence", Replicas: 1}
	f := &backendFixture{t: t, settings: s}
	_, envName := resourceParts(s.Environment)
	domain := "backendzone." + c.Region + ".azurecontainerapps.io"
	f.environment = a.ManagedEnvironment{ID: to.Ptr(s.Environment), Name: to.Ptr(envName), Type: to.Ptr("Microsoft.App/managedEnvironments"), Location: to.Ptr(c.Region), Properties: &a.ManagedEnvironmentProperties{ProvisioningState: to.Ptr(a.EnvironmentProvisioningStateSucceeded), VnetConfiguration: &a.VnetConfiguration{InfrastructureSubnetID: to.Ptr(s.Subnet), Internal: to.Ptr(true)}, PublicNetworkAccess: to.Ptr(a.PublicNetworkAccessDisabled), InfrastructureResourceGroup: to.Ptr(s.InfrastructureGroup), DefaultDomain: to.Ptr(domain), AppLogsConfiguration: &a.AppLogsConfiguration{Destination: to.Ptr("none")}, WorkloadProfiles: []*a.WorkloadProfile{{Name: to.Ptr("Consumption"), WorkloadProfileType: to.Ptr("Consumption")}}}}
	var err error
	f.application, err = s.ApplicationDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	_, appName := resourceParts(s.Application)
	f.application.ID, f.application.Name, f.application.Type = to.Ptr(s.Application), to.Ptr(appName), to.Ptr("Microsoft.App/containerApps")
	p := f.application.Properties
	p.ProvisioningState, p.RunningStatus = to.Ptr(a.ContainerAppProvisioningStateSucceeded), to.Ptr(a.ContainerAppRunningStatusRunning)
	p.Configuration.Ingress.Fqdn = to.Ptr(appName + "." + domain)
	p.LatestRevisionName, p.LatestReadyRevisionName = to.Ptr(appName+"--current"), to.Ptr(appName+"--current")
	for index, id := range []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity} {
		f.application.Identity.UserAssignedIdentities[id] = &a.UserAssignedIdentity{ClientID: to.Ptr(fmt.Sprintf("77777777-2222-3333-4444-55555555555%d", index+1)), PrincipalID: to.Ptr(fmt.Sprintf("88888888-2222-3333-4444-55555555555%d", index+1))}
	}
	f.revisions = []*a.Revision{{ID: to.Ptr(s.Application + "/revisions/" + appName + "--current"), Name: to.Ptr(appName + "--current"), Type: to.Ptr("Microsoft.App/containerApps/revisions"), Properties: &a.RevisionProperties{Active: to.Ptr(true), ProvisioningState: to.Ptr(a.RevisionProvisioningStateProvisioned), HealthState: to.Ptr(a.RevisionHealthStateHealthy), RunningState: to.Ptr(a.RevisionRunningStateRunning), Replicas: to.Ptr[int32](1), TrafficWeight: to.Ptr[int32](100), Template: clone(p.Template)}}}
	f.auth = a.AuthConfig{ID: to.Ptr(s.Application + "/authConfigs/current"), Name: to.Ptr("current"), Type: to.Ptr("Microsoft.App/containerApps/authConfigs"), Properties: &a.AuthConfigProperties{Platform: &a.AuthPlatform{Enabled: to.Ptr(false)}}}
	placement := prior.reader.(*SDKWorkloadInventoryReader).http.base.(*workloadInventoryFixture)
	placement.apps = append(placement.apps, to.Ptr(clone(f.application)))
	reader, err := NewSDKBackendReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewBackendGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *backendFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != exchangeVersion || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("backend reader widened its authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-backend-canary"}}), nil
	}
	var body any
	switch r.URL.Path {
	case f.settings.Environment:
		body = f.environment
	case f.settings.Application:
		body = f.application
	case f.settings.Application + "/authConfigs/current":
		body = f.auth
	case f.settings.Environment + "/httpRouteConfigs":
		body = map[string]any{"value": []any{}}
		if f.mode == "missing-routes" {
			body = map[string]any{}
		}
		if f.mode == "null-routes" {
			body = map[string]any{"value": nil}
		}
		if f.mode == "extra-route" {
			body = map[string]any{"value": []*a.HTTPRouteConfig{{Name: to.Ptr("other-route")}}}
		}
	case f.settings.Application + "/revisions":
		body = map[string]any{"value": f.revisions}
		if f.mode == "missing-revisions" {
			body = map[string]any{}
		}
		if f.mode == "foreign-page" {
			body = map[string]any{"value": f.revisions, "nextLink": "https://foreign.invalid/private-canary"}
		}
		if f.mode == "unfinished" {
			f.page++
			body = map[string]any{"value": []any{}, "nextLink": "https://management.azure.com" + r.URL.Path + "?api-version=" + exchangeVersion + "&%24skipToken=" + strconv.Itoa(f.page)}
		}
	default:
		f.t.Error("unselected backend resource reached transport")
		return deadlineResponse(r, 404, nil), nil
	}
	return deadlineResponse(r, 200, body), nil
}
