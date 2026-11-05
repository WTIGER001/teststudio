package azureworker

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	r "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
)

type preparationHostFixture struct {
	t           *testing.T
	mu          sync.Mutex
	settings    PreparationSettings
	application a.ContainerApp
	environment a.ManagedEnvironment
	storage     a.ManagedEnvironmentStorage
	identity    m.Identity
	assignment  r.RoleAssignment
	role        r.RoleDefinition
	revisions   []a.Revision
	reads       int
	mode        string
	parent      *preparationWriterFixture
}

func preparationHostTest(t *testing.T) (PreparationSettings, *preparationHostFixture, *PreparationHostGuard) {
	t.Helper()
	c, writer, previous, _, prior := preparationWriterTest(t)
	h := PreparationHostSettings{Application: c.resourceID("Microsoft.App", "containerApps", "studio-preparation"), Environment: c.resourceID("Microsoft.App", "managedEnvironments", "studio-preparation"), Image: c.Registry + "/services/teststudio@sha256:" + strings.Repeat("e", 64), ReaderIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "source-reader"), ReaderPrincipalID: "bbbbbbbb-1111-2222-3333-444444444444", ReaderAssignment: observerScope(c) + "/providers/Microsoft.Authorization/roleAssignments/cccccccc-1111-2222-3333-444444444444", ReaderRole: observerScope(c) + "/providers/Microsoft.Authorization/roleDefinitions/dddddddd-1111-2222-3333-444444444444", PullIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "source-pull"), StartupIdentity: c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "source-startup"), JournalStorageName: "source-journals", JournalAccount: "studiorecovery", JournalShare: "source-journals", DatabaseSecretURL: "https://studiovault.vault.azure.net/secrets/database", ConfigurationSecretURL: "https://studiovault.vault.azure.net/secrets/source-settings", GoRuntime: c.Registry + "/" + c.RepositoryPrefix + "/go@sha256:" + strings.Repeat("a", 64), PlaywrightRuntime: c.Registry + "/" + c.RepositoryPrefix + "/playwright@sha256:" + strings.Repeat("b", 64)}
	s := PreparationSettings{Worker: c, Writer: writer, ReaderClientID: "aaaaaaaa-1111-2222-3333-444444444444", Host: &h}
	f := &preparationHostFixture{t: t, settings: s, parent: previous}
	var err error
	f.application, err = s.PreparationApplicationDefinition()
	if err != nil {
		t.Fatal(err)
	}
	_, name := resourceParts(h.Application)
	f.application.ID, f.application.Name, f.application.Type = to.Ptr(h.Application), to.Ptr(name), to.Ptr("Microsoft.App/containerApps")
	f.application.Properties.ProvisioningState = to.Ptr(a.ContainerAppProvisioningStateSucceeded)
	f.application.Properties.RunningStatus = to.Ptr(a.ContainerAppRunningStatusRunning)
	f.application.Properties.LatestRevisionName, f.application.Properties.LatestReadyRevisionName = to.Ptr(name+"--current"), to.Ptr(name+"--current")
	for i, id := range h.identities(writer) {
		client, principal := fmt.Sprintf("66666666-aaaa-bbbb-cccc-11111111111%d", i), fmt.Sprintf("77777777-aaaa-bbbb-cccc-11111111111%d", i)
		if id == writer.Identity {
			client, principal = writer.ClientID, writer.PrincipalID
		} else if id == h.ReaderIdentity {
			client, principal = s.ReaderClientID, h.ReaderPrincipalID
		}
		f.application.Identity.UserAssignedIdentities[id] = &a.UserAssignedIdentity{ClientID: &client, PrincipalID: &principal}
	}
	f.revisions = []a.Revision{{ID: to.Ptr(h.Application + "/revisions/" + name + "--current"), Name: to.Ptr(name + "--current"), Type: to.Ptr("Microsoft.App/containerApps/revisions"), Properties: &a.RevisionProperties{Active: to.Ptr(true), ProvisioningState: to.Ptr(a.RevisionProvisioningStateProvisioned), HealthState: to.Ptr(a.RevisionHealthStateHealthy), RunningState: to.Ptr(a.RevisionRunningStateRunning), Replicas: to.Ptr[int32](1), Template: clone(f.application.Properties.Template)}}}
	_, environment := resourceParts(h.Environment)
	f.environment = a.ManagedEnvironment{ID: to.Ptr(h.Environment), Name: to.Ptr(environment), Type: to.Ptr("Microsoft.App/managedEnvironments"), Location: to.Ptr(c.Region), Properties: &a.ManagedEnvironmentProperties{ProvisioningState: to.Ptr(a.EnvironmentProvisioningStateSucceeded), PublicNetworkAccess: to.Ptr(a.PublicNetworkAccessDisabled), VnetConfiguration: &a.VnetConfiguration{InfrastructureSubnetID: to.Ptr(writer.Registry.SourceSubnet), Internal: to.Ptr(true)}, WorkloadProfiles: []*a.WorkloadProfile{{Name: to.Ptr("Consumption"), WorkloadProfileType: to.Ptr("Consumption")}}}}
	f.storage = a.ManagedEnvironmentStorage{ID: to.Ptr(h.storage()), Name: to.Ptr(h.JournalStorageName), Type: to.Ptr("Microsoft.App/managedEnvironments/storages"), Properties: &a.ManagedEnvironmentStorageProperties{AzureFile: &a.AzureFileProperties{AccountName: to.Ptr(h.JournalAccount), ShareName: to.Ptr(h.JournalShare), AccessMode: to.Ptr(a.AccessModeReadWrite), AccountKey: to.Ptr("private-journal-key-canary")}}}
	f.identity = m.Identity{ID: to.Ptr(h.ReaderIdentity), Name: to.Ptr("source-reader"), Type: to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities"), Location: to.Ptr(c.Region), Properties: &m.UserAssignedIdentityProperties{ClientID: to.Ptr(s.ReaderClientID), PrincipalID: to.Ptr(h.ReaderPrincipalID), TenantID: to.Ptr(writer.TenantID)}}
	f.assignment = r.RoleAssignment{ID: to.Ptr(h.ReaderAssignment), Name: to.Ptr("cccccccc-1111-2222-3333-444444444444"), Type: to.Ptr("Microsoft.Authorization/roleAssignments"), Properties: &r.RoleAssignmentProperties{Scope: to.Ptr(observerScope(c)), PrincipalID: to.Ptr(h.ReaderPrincipalID), PrincipalType: to.Ptr(r.PrincipalTypeServicePrincipal), RoleDefinitionID: to.Ptr(h.ReaderRole)}}
	f.role = r.RoleDefinition{ID: to.Ptr(h.ReaderRole), Name: to.Ptr("dddddddd-1111-2222-3333-444444444444"), Type: to.Ptr("Microsoft.Authorization/roleDefinitions"), Properties: &r.RoleDefinitionProperties{RoleType: to.Ptr("CustomRole"), AssignableScopes: []*string{to.Ptr(observerScope(c))}, Permissions: []*r.Permission{{Actions: []*string{to.Ptr("*/read"), to.Ptr(networkReadAssociationAction)}}}}}
	reader, err := NewSDKPreparationHostReader(s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewPreparationHostGuard(s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return s, f, g
}

func (f *preparationHostFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	h := f.settings.Host
	version := exchangeVersion
	var body any
	switch r.URL.Path {
	case h.Application:
		body = f.application
	case h.Environment:
		body = f.environment
	case h.storage():
		body = f.storage
	case h.ReaderIdentity:
		body, version = f.identity, readIdentityVersion
	case h.ReaderAssignment:
		body, version = f.assignment, readAssignmentVersion
	case h.ReaderRole:
		body, version = f.role, readRoleVersion
	case h.Application + "/revisions":
		body = map[string]any{"value": f.revisions}
		if f.mode == "missing-revisions" {
			body = map[string]any{}
		} else if f.mode == "incomplete-revisions" {
			body = map[string]any{"value": f.revisions, "nextLink": "https://foreign.invalid/private-revision-canary"}
		}
	default:
		f.t.Error("preparation host reached an unselected resource")
		return deadlineResponse(r, 404, nil), nil
	}
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version="+version || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("preparation host reader widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"message": "private-host-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private-host-canary")
		return response, nil
	}
	return deadlineResponse(r, 200, body), nil
}
