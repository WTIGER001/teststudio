package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func serviceProperties(w *placementWorld, id string) map[string]any {
	return w.resources[id]["properties"].(map[string]any)
}

func serviceTemplate(w *placementWorld, id string) map[string]any {
	return serviceProperties(w, id)["template"].(map[string]any)
}

func TestSandboxServiceDefinitionsKeepRelayAndBackendAuthoritySeparate(t *testing.T) {
	s := runtimeSettings()
	defs, err := s.ServiceDefinitions()
	if err != nil || len(defs) != 4 {
		t.Fatal("missing reviewed service definitions", err)
	}
	r, b := s.Placement.Services.Relay, s.Placement.Services.Backend
	relay := defs[r.Application].(a.ContainerApp)
	backend := defs[b.Application].(a.ContainerApp)
	if len(relay.Identity.UserAssignedIdentities) != 1 || len(relay.Properties.Configuration.IdentitySettings) != 1 || *relay.Properties.Configuration.IdentitySettings[0].Lifecycle != a.IdentitySettingsLifeCycleNone || len(relay.Properties.Configuration.Secrets) != 0 {
		t.Fatal("relay receives runtime credentials or backend secrets")
	}
	if *relay.Properties.Template.Containers[0].Command[0] != "/app/teststudio-exchange" || *backend.Properties.Template.Containers[0].Command[0] != "/app/teststudio" {
		t.Fatal("trusted service command changed")
	}
	if len(backend.Identity.UserAssignedIdentities) != 3 || len(backend.Properties.Configuration.Secrets) != 1 || *backend.Properties.Configuration.Secrets[0].KeyVaultURL != b.DatabaseSecretURL {
		t.Fatal("backend credential bindings changed")
	}
	lifecycles := map[string]string{}
	for _, identity := range backend.Properties.Configuration.IdentitySettings {
		lifecycles[*identity.Identity] = string(*identity.Lifecycle)
	}
	if lifecycles[b.Pull.Resource] != "None" || lifecycles[b.Startup.Resource] != "None" || lifecycles[b.Runtime.Resource] != "Main" {
		t.Fatal("startup or pull identity reaches the backend runtime")
	}
	args := []string{}
	for _, arg := range backend.Properties.Template.Containers[0].Args {
		args = append(args, *arg)
	}
	if !strings.Contains(strings.Join(args, " "), "--serve-role=worker") || !strings.Contains(strings.Join(args, " "), "--require-postgres=true") {
		t.Fatal("backend enables another role or database mode")
	}
}

func TestSandboxServicesRejectEnvironmentCertificateIdentityTemplateAndRevisionDrift(t *testing.T) {
	for _, mode := range []string{"relay-public", "backend-public", "wrong-subnet", "wrong-image", "wrong-command", "backend-role", "backend-origin", "runtime-client", "identity-exposure", "extra-identity", "kind", "managed-by", "auth", "route", "approval", "embedded-connection", "private-link", "certificate-host", "certificate-expiry", "certificate-secret", "revision-template", "revision-health", "two-active", "pending-app"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			p, v := s.Placement, s.Placement.Services
			relay := serviceProperties(w, v.Relay.Application)
			backend := serviceProperties(w, v.Backend.Application)
			revisions := w.pages[v.Relay.Application+"/revisions"]["value"].([]any)
			revision := revisions[0].(map[string]any)["properties"].(map[string]any)
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "relay-public":
				serviceProperties(w, p.Private.ExchangeEnvironment)["publicNetworkAccess"] = "Enabled"
			case "backend-public":
				serviceProperties(w, v.Backend.Environment)["publicNetworkAccess"] = "Enabled"
			case "wrong-subnet":
				serviceProperties(w, v.Backend.Environment)["vnetConfiguration"].(map[string]any)["infrastructureSubnetId"] = v.Relay.Subnet
			case "wrong-image":
				serviceTemplate(w, v.Relay.Application)["containers"].([]any)[0].(map[string]any)["image"] = v.Backend.Image
			case "wrong-command":
				serviceTemplate(w, v.Relay.Application)["containers"].([]any)[0].(map[string]any)["command"] = []string{"/bin/sh"}
			case "backend-role":
				serviceTemplate(w, v.Backend.Application)["containers"].([]any)[0].(map[string]any)["args"] = []string{"--serve-role=all"}
			case "backend-origin":
				serviceProperties(w, v.Backend.Environment)["defaultDomain"] = "other.westus2.azurecontainerapps.io"
			case "runtime-client":
				w.resources[v.Backend.Application]["identity"].(map[string]any)["userAssignedIdentities"].(map[string]any)[v.Backend.Runtime.Resource].(map[string]string)["clientId"] = s.ControlClientID
			case "identity-exposure":
				for _, raw := range backend["configuration"].(map[string]any)["identitySettings"].([]any) {
					setting := raw.(map[string]any)
					if setting["identity"] == v.Backend.Startup.Resource {
						setting["lifecycle"] = "Main"
					}
				}
			case "extra-identity":
				w.resources[v.Relay.Application]["identity"].(map[string]any)["userAssignedIdentities"].(map[string]any)[s.PullIdentity] = map[string]string{"clientId": s.Placement.PullClientID, "principalId": s.Placement.PullPrincipalID}
			case "kind":
				w.resources[v.Relay.Application]["kind"] = "unreviewed-kind"
			case "managed-by":
				w.resources[v.Relay.Application]["managedBy"] = "unreviewed-manager"
			case "auth":
				serviceProperties(w, v.Relay.Application+"/authConfigs/current")["platform"].(map[string]any)["enabled"] = true
			case "route":
				w.pages[p.Private.ExchangeEnvironment+"/httpRouteConfigs"]["value"] = []any{map[string]any{"id": p.Private.ExchangeEnvironment + "/httpRouteConfigs/escape"}}
			case "approval":
				w.pages[p.Private.ExchangeEnvironment+"/privateEndpointConnections"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["privateLinkServiceConnectionState"].(map[string]string)["status"] = "Rejected"
			case "embedded-connection":
				serviceProperties(w, p.Private.ExchangeEnvironment)["privateEndpointConnections"] = []any{}
				expected = hostedexecution.ErrUnavailable
			case "private-link":
				w.pages[p.Private.ExchangeEnvironment+"/privateLinkResources"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["groupId"] = "other"
			case "certificate-host":
				serviceProperties(w, v.Relay.Certificate)["subjectAlternativeNames"] = []string{"other.studio.example"}
			case "certificate-expiry":
				serviceProperties(w, v.Relay.Certificate)["expirationDate"] = time.Now().Add(time.Hour)
				expected = hostedexecution.ErrUnavailable
			case "certificate-secret":
				serviceProperties(w, v.Relay.Certificate)["password"] = "private-certificate-canary"
			case "revision-template":
				encoded, _ := json.Marshal(revision["template"])
				var template map[string]any
				if json.Unmarshal(encoded, &template) != nil {
					t.Fatal("invalid revision fixture")
				}
				template["containers"].([]any)[0].(map[string]any)["image"] = v.Backend.Image
				revision["template"] = template
			case "revision-health":
				revision["healthState"] = "Unhealthy"
				expected = hostedexecution.ErrUnavailable
			case "two-active":
				w.pages[v.Relay.Application+"/revisions"]["value"] = append(revisions, map[string]any{"id": v.Relay.Application + "/revisions/studio-exchange--other", "name": "studio-exchange--other", "type": "Microsoft.App/containerApps/revisions", "properties": revision})
			case "pending-app":
				relay["provisioningState"] = "Updating"
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKServices(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != expected || err != nil && strings.Contains(err.Error(), "canary") {
				t.Fatal("unsafe service binding admitted or diagnostics exposed", err)
			}
		})
	}
}

func TestSandboxServiceBindingsRejectInvalidSettingsBeforeCredentials(t *testing.T) {
	for _, mode := range []string{"missing", "source-image", "tagged-image", "shared-network", "shared-infrastructure", "case-app", "case-environment", "same-identity", "controller-identity", "registry-client", "inline-database", "public-address", "certificate-scope", "profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			v := &s.Placement.Services
			switch mode {
			case "missing":
				*v = ServicesSettings{}
			case "source-image":
				v.Relay.Image = s.Worker.Registry + "/" + s.Worker.RepositoryPrefix + "/relay@sha256:" + strings.Repeat("b", 64)
			case "tagged-image":
				v.Backend.Image = strings.Split(v.Backend.Image, "@")[0] + ":latest"
			case "shared-network":
				v.Relay.Subnet = s.Placement.Subnet
			case "shared-infrastructure":
				v.Backend.InfrastructureGroup = v.Relay.InfrastructureGroup
			case "case-app":
				v.Backend.Application = strings.Replace(v.Relay.Application, "/studio-relay/", "/STUDIO-RELAY/", 1)
			case "case-environment":
				v.Backend.Environment = strings.Replace(s.Placement.Private.ExchangeEnvironment, "/studio-relay/", "/STUDIO-RELAY/", 1)
			case "same-identity":
				v.Backend.Startup = v.Backend.Runtime
			case "controller-identity":
				v.Relay.Pull = ServiceIdentitySettings{s.Authority.Control.Resource, s.ControlClientID, s.Authority.Control.PrincipalID}
			case "registry-client":
				v.Backend.Runtime.Client = s.RegistryClientID
			case "inline-database":
				v.Backend.DatabaseSecretURL = "postgres://private-database-canary"
			case "public-address":
				v.Backend.Address = "198.51.100.4"
			case "certificate-scope":
				v.Relay.Certificate = v.Backend.Environment + "/certificates/studio-exchange"
			case "profile":
				v.Backend.Image = strings.Split(v.Backend.Image, "@")[0] + "@sha256:" + strings.Repeat("d", 64)
			}
			if mode != "profile" {
				s.Worker.NetworkProfile, _ = s.Placement.NetworkProfile(s.Worker)
			}
			f := newControllerFixture(t, "go", "")
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid services constructed a credential")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration || strings.Contains(err.Error(), "canary") {
				t.Fatal("invalid services accepted or raw settings exposed", err)
			}
		})
	}
}

func TestSandboxServicesAcceptOnlyQuiescedPriorRevisionsAcrossBoundedPages(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	app := s.Placement.Services.Relay.Application
	path := app + "/revisions"
	values := w.pages[path]["value"].([]any)
	for i := range 3 {
		name := "studio-exchange--prior" + strconv.Itoa(i)
		values = append(values, map[string]any{"id": app + "/revisions/" + name, "name": name, "type": "Microsoft.App/containerApps/revisions", "properties": map[string]any{"active": false, "trafficWeight": 0, "replicas": 0}})
	}
	calls := 0
	transport := runtimeTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != path {
			return w.Do(r)
		}
		if r.Method != "GET" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2025-07-01" || r.Header.Get("Authorization") != "Bearer placement-metadata-canary" || calls >= 4 {
			t.Error("revision inventory escaped its selected request")
			return response(503, `{}`), nil
		}
		if calls > 0 && r.URL.Query().Get("$skipToken") != strconv.Itoa(calls) {
			t.Error("revision continuation changed or repeated")
		}
		page := map[string]any{"value": []any{values[calls]}}
		calls++
		if calls < 4 {
			page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-07-01&$skipToken=" + strconv.Itoa(calls)
		}
		return jsonResponse(page), nil
	})
	reader, err := NewSDKServices(s, &runtimeCredential{token: "placement-metadata-canary"}, transport)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if err := reader.Check(t.Context(), s.Worker); err != nil || calls != 4 {
		t.Fatal("complete safe revision inventory failed", err)
	}
	values[1].(map[string]any)["properties"].(map[string]any)["replicas"] = 1
	calls = 0
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("inactive revision with replicas counted as quiesced", err)
	}
}

func TestSandboxServiceMetadataSharesBoundedEvidenceAndRetainsCurrentFailure(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	reader, err := NewSDKServices(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	now := time.Now()
	reader.now = func() time.Time { return now }
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			if err := reader.CheckRunning(t.Context(), s.Worker); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if len(w.calls) != 13 {
		t.Fatal("service observation scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("concurrent service checks repeated reads")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("missing service metadata became worker absence", err)
	}
	w.status = 0
	serviceTemplate(w, s.Placement.Services.Relay.Application)["containers"].([]any)[0].(map[string]any)["command"] = []string{"/bin/sh"}
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("fresh dispatch used cached safe service", err)
	}
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("current failed observation retained earlier success", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.CheckRunning(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled service check returned evidence", err)
	}
	reader.Close()
	if err := reader.CheckRunning(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed service check returned evidence", err)
	}
}

func TestSandboxRuntimeServicesGateBlocksCreationImportAndLiveCommands(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings()
	s.Mode = "discovery"
	w := newPlacementWorld(t, s)
	protection := &placementProtection{}
	r, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: protection, ImageAdmission: protection, ARMTransport: w, Credentials: func(string) (azcore.TokenCredential, error) {
		return &runtimeCredential{token: "placement-metadata-canary"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	serviceTemplate(w, s.Placement.Services.Backend.Application)["containers"].([]any)[0].(map[string]any)["args"] = []string{"--serve-role=all"}
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("changed backend reached image import", err)
	}
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("changed backend reached creation", err)
	}
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("changed backend reached live command admission", err)
	}
	r.Close()
	if err := r.services.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left service reader open", err)
	}
}
