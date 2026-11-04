package azureworker

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

type exchangeFixture struct {
	t           *testing.T
	mu          sync.Mutex
	settings    ExchangeSettings
	environment a.ManagedEnvironment
	application a.ContainerApp
	certificate a.Certificate
	auth        a.AuthConfig
	connections []*a.PrivateEndpointConnection
	links       []*a.PrivateLinkResource
	routes      []*a.HTTPRouteConfig
	revisions   []*a.Revision
	reads       int
	pages       map[string]int
	mode        string
}

func exchangeServiceTest(t *testing.T) (Config, ExchangeSettings, *exchangeFixture, *ExchangeGuard) {
	t.Helper()
	c, registry, _, guard := registryServiceTest(t)
	env := registry.Endpoints.ExchangeEnvironment
	vnet := c.resourceID("Microsoft.Network", "virtualNetworks", "exchange-network")
	s := ExchangeSettings{Registry: registry, Application: c.resourceID("Microsoft.App", "containerApps", "worker-exchange-relay"), Certificate: env + "/certificates/worker-https", VirtualNetwork: vnet, Subnet: vnet + "/subnets/infrastructure", InfrastructureGroup: "studio-exchange-infra", Image: c.Registry + "/services/worker-relay@sha256:" + strings.Repeat("d", 64), BackendOrigin: "https://studio-worker-backend.backendzone." + c.Region + ".azurecontainerapps.io"}
	f := &exchangeFixture{t: t, settings: s, pages: map[string]int{}, routes: []*a.HTTPRouteConfig{}}
	_, envName := resourceParts(env)
	domain := "exchangezone." + c.Region + ".azurecontainerapps.io"
	f.environment = a.ManagedEnvironment{ID: to.Ptr(env), Name: to.Ptr(envName), Type: to.Ptr("Microsoft.App/managedEnvironments"), Location: to.Ptr(c.Region), Properties: &a.ManagedEnvironmentProperties{ProvisioningState: to.Ptr(a.EnvironmentProvisioningStateSucceeded), VnetConfiguration: &a.VnetConfiguration{InfrastructureSubnetID: to.Ptr(s.Subnet), Internal: to.Ptr(true)}, PublicNetworkAccess: to.Ptr(a.PublicNetworkAccessDisabled), InfrastructureResourceGroup: to.Ptr(s.InfrastructureGroup), DefaultDomain: to.Ptr(domain), AppLogsConfiguration: &a.AppLogsConfiguration{Destination: to.Ptr("none")}, WorkloadProfiles: []*a.WorkloadProfile{{Name: to.Ptr("Consumption"), WorkloadProfileType: to.Ptr("Consumption")}}}}
	f.connections = []*a.PrivateEndpointConnection{{ID: to.Ptr(s.connections() + "/worker.approval"), Name: to.Ptr("worker.approval"), Type: to.Ptr("Microsoft.App/managedEnvironments/privateEndpointConnections"), Properties: &a.PrivateEndpointConnectionProperties{PrivateEndpoint: &a.PrivateEndpoint{ID: to.Ptr(registry.Endpoints.ExchangeEndpoint)}, GroupIDs: []*string{to.Ptr("managedEnvironment")}, ProvisioningState: to.Ptr(a.PrivateEndpointConnectionProvisioningStateSucceeded), PrivateLinkServiceConnectionState: &a.PrivateLinkServiceConnectionState{Status: to.Ptr(a.PrivateEndpointServiceConnectionStatusApproved), ActionsRequired: to.Ptr("None")}}}}
	f.environment.Properties.PrivateEndpointConnections = clone(f.connections)
	f.links = []*a.PrivateLinkResource{{ID: to.Ptr(s.links() + "/managedEnvironment"), Name: to.Ptr("managedEnvironment"), Type: to.Ptr("Microsoft.App/managedEnvironments/privateLinkResources"), Properties: &a.PrivateLinkResourceProperties{GroupID: to.Ptr("managedEnvironment"), RequiredMembers: []*string{to.Ptr("managedEnvironment")}, RequiredZoneNames: []*string{to.Ptr("privatelink." + c.Region + ".azurecontainerapps.io")}}}}
	origin, _ := url.Parse(c.ExchangeOrigin)
	f.certificate = a.Certificate{ID: to.Ptr(s.Certificate), Name: to.Ptr("worker-https"), Type: to.Ptr("Microsoft.App/managedEnvironments/certificates"), Location: to.Ptr(c.Region), Properties: &a.CertificateProperties{ProvisioningState: to.Ptr(a.CertificateProvisioningStateSucceeded), Valid: to.Ptr(true), IssueDate: to.Ptr(time.Now().Add(-time.Hour)), ExpirationDate: to.Ptr(time.Now().Add(48 * time.Hour)), SubjectAlternativeNames: []*string{to.Ptr(origin.Hostname())}}}
	var err error
	f.application, err = s.ApplicationDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	_, appName := resourceParts(s.Application)
	f.application.ID, f.application.Name, f.application.Type = to.Ptr(s.Application), to.Ptr(appName), to.Ptr("Microsoft.App/containerApps")
	f.application.Properties.ProvisioningState = to.Ptr(a.ContainerAppProvisioningStateSucceeded)
	f.application.Properties.RunningStatus = to.Ptr(a.ContainerAppRunningStatusRunning)
	f.application.Properties.Configuration.Ingress.Fqdn = to.Ptr(appName + "." + domain)
	f.application.Properties.LatestRevisionName = to.Ptr(appName + "--r1")
	f.application.Properties.LatestReadyRevisionName = to.Ptr(appName + "--r1")
	f.revisions = []*a.Revision{{ID: to.Ptr(s.revisions() + "/" + appName + "--r1"), Name: to.Ptr(appName + "--r1"), Type: to.Ptr("Microsoft.App/containerApps/revisions"), Properties: &a.RevisionProperties{Active: to.Ptr(true), ProvisioningState: to.Ptr(a.RevisionProvisioningStateProvisioned), HealthState: to.Ptr(a.RevisionHealthStateHealthy), RunningState: to.Ptr(a.RevisionRunningStateRunning), Replicas: to.Ptr[int32](1), TrafficWeight: to.Ptr[int32](100), Template: clone(f.application.Properties.Template)}}, {ID: to.Ptr(s.revisions() + "/" + appName + "--r0"), Name: to.Ptr(appName + "--r0"), Type: to.Ptr("Microsoft.App/containerApps/revisions"), Properties: &a.RevisionProperties{Active: to.Ptr(false), Replicas: to.Ptr[int32](0), TrafficWeight: to.Ptr[int32](0)}}}
	f.auth, _ = s.AuthDefinition(c)
	f.auth.ID, f.auth.Name, f.auth.Type = to.Ptr(s.auth()), to.Ptr("current"), to.Ptr("Microsoft.App/containerApps/authConfigs")
	reader, err := NewSDKExchangeReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	result, err := NewExchangeGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, result
}

func (f *exchangeFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2025-07-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("exchange SDK widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	switch f.mode {
	case "missing":
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	case "unavailable":
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "exchange-private-canary"}}), nil
	case "redirect":
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	var body any
	path := r.URL.Path
	switch path {
	case f.settings.environment():
		body = f.environment
	case f.settings.Application:
		body = f.application
	case f.settings.Certificate:
		body = f.certificate
	case f.settings.auth():
		body = f.auth
	case f.settings.connections():
		body = map[string]any{"value": f.connections}
	case f.settings.links():
		body = map[string]any{"value": f.links}
	case f.settings.routes():
		body = map[string]any{"value": f.routes}
	case f.settings.revisions():
		f.pages[path]++
		page := map[string]any{"value": f.revisions}
		switch f.mode {
		case "paged", "cycle":
			if r.URL.Query().Get("$skipToken") == "" {
				page["value"] = f.revisions[:1]
				page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-07-01&%24skipToken=one"
			} else {
				page["value"] = f.revisions[1:]
				if f.mode == "cycle" {
					page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-07-01&%24skipToken=one"
				}
			}
		case "foreign-page":
			page["nextLink"] = "https://foreign.invalid/private?api-version=2025-07-01&%24skipToken=one"
		case "other-list":
			page["nextLink"] = "https://management.azure.com" + f.settings.connections() + "?api-version=2025-07-01&%24skipToken=one"
		case "endless":
			page["value"] = []any{}
			page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-07-01&%24skipToken=" + strconv.Itoa(f.pages[path])
		}
		body = page
	default:
		f.t.Error("unowned exchange resource requested")
		return deadlineResponse(r, 404, nil), nil
	}
	response := deadlineResponse(r, 200, body)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}
