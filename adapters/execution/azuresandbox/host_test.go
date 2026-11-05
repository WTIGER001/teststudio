package azuresandbox

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxHostModesKeepImagePoolAndSeparateFixedHostProfiles(t *testing.T) {
	execution, discovery := runtimeSettings(), runtimeSettings("discovery")
	ep, ee := ExecutionProfile(execution.Worker)
	dp, de := ExecutionProfile(discovery.Worker)
	if ee != nil || de != nil || ep != dp || execution.Host.Profile == discovery.Host.Profile {
		t.Fatal("host selection fragmented the converted image pool or lost its separate binding")
	}
	for _, s := range []Settings{execution, discovery} {
		defs, err := s.HostDefinitions()
		if err != nil || len(defs) != 2 {
			t.Fatal("missing host packaging inputs", err)
		}
		app := defs[s.Host.Application].(a.ContainerApp)
		count, command, first := 6, "/app/teststudio", "--serve-role=controller"
		if s.Mode == "discovery" {
			count, command, first = 5, "/app/teststudio-prepare", "--discovery-only"
		}
		databaseArgument := false
		for _, arg := range app.Properties.Template.Containers[0].Args {
			if *arg == "--database-host="+s.Host.DatabaseHost {
				databaseArgument = true
			}
		}
		if !databaseArgument || app.Properties.Configuration.Ingress != nil || len(app.Identity.UserAssignedIdentities) != count || *app.Properties.Template.Containers[0].Command[0] != command || *app.Properties.Template.Containers[0].Args[0] != first {
			t.Fatal("host roles or identity sets were combined")
		}
		w := newPlacementWorld(t, s)
		reader, err := NewSDKHost(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		if err := reader.Check(t.Context(), s.Worker); err != nil || len(w.calls) != 6 {
			t.Fatal("fixed host metadata rejected", err, len(w.calls))
		}
	}
	f := newControllerFixture(t, "go", "")
	for _, mode := range []string{"missing", "invalid-database-host", "application-alias", "network-alias", "identity-alias", "configuration-secret", "discovery-runtime", "stale-profile"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			switch mode {
			case "missing":
				s.Host = HostSettings{}
			case "invalid-database-host":
				s.Host.DatabaseHost = "127.0.0.1"
			case "application-alias":
				s.Host.Application = s.Placement.Services.Relay.Application
			case "network-alias":
				s.Host.Subnet = s.Placement.Subnet
			case "identity-alias":
				s.Host.Pull = s.Placement.Services.Backend.Pull
			case "configuration-secret":
				s.Host.ConfigurationSecretURL = "https://foreign.example/config"
			case "discovery-runtime":
				s.Mode = "discovery"
			case "stale-profile":
				s.Host.Replicas++
			}
			if mode != "stale-profile" {
				s.Host.Profile = s.HostProfile()
			}
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid host constructed credentials")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid host settings admitted", err)
			}
		})
	}
}

func TestSandboxHostRejectsTemplateIdentitySecretRevisionAndEnvironmentDrift(t *testing.T) {
	for _, mode := range []string{"command", "role", "database-host", "extra-argument", "ingress", "identity-lifecycle", "client", "secret-identity", "inline-secret", "host-profile", "extra-environment", "sidecar", "environment", "public-access", "routes", "endpoint", "revision", "pending"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			h := s.Host
			props := w.resources[h.Application]["properties"].(map[string]any)
			container := props["template"].(map[string]any)["containers"].([]any)[0].(map[string]any)
			configuration := props["configuration"].(map[string]any)
			expected := hostedexecution.ErrOwnership
			switch mode {
			case "command":
				container["command"] = []string{"/app/teststudio-job"}
			case "role":
				container["args"].([]any)[0] = "--serve-role=all"
			case "database-host":
				for i, raw := range container["args"].([]any) {
					if strings.HasPrefix(raw.(string), "--database-host=") {
						container["args"].([]any)[i] = "--database-host=foreign.example.test"
					}
				}
			case "extra-argument":
				container["args"] = append(container["args"].([]any), "--go-images=untrusted")
			case "ingress":
				configuration["ingress"] = map[string]any{"external": true, "targetPort": 8043}
			case "identity-lifecycle":
				for _, raw := range configuration["identitySettings"].([]any) {
					entry := raw.(map[string]any)
					if entry["identity"] == s.Authority.Control.Resource {
						entry["lifecycle"] = "None"
					}
				}
			case "client":
				w.resources[h.Application]["identity"].(map[string]any)["userAssignedIdentities"].(map[string]any)[s.Authority.Control.Resource] = map[string]string{"clientId": s.RegistryClientID, "principalId": s.Authority.Control.PrincipalID}
			case "secret-identity":
				configuration["secrets"].([]any)[0].(map[string]any)["identity"] = s.Authority.Metadata.Resource
			case "inline-secret":
				configuration["secrets"].([]any)[0].(map[string]any)["value"] = "private-canary"
			case "host-profile":
				container["env"].([]any)[0].(map[string]any)["value"] = hash("other host")
			case "extra-environment":
				container["env"] = append(container["env"].([]any), map[string]string{"name": "HTTP_PROXY", "value": "private-canary"})
			case "sidecar":
				props["template"].(map[string]any)["containers"] = append(props["template"].(map[string]any)["containers"].([]any), container)
			case "environment":
				props["managedEnvironmentId"] = s.Placement.Services.Backend.Environment
			case "public-access":
				w.resources[h.Environment]["properties"].(map[string]any)["publicNetworkAccess"] = "Enabled"
			case "routes":
				w.pages[h.Environment+"/httpRouteConfigs"]["value"] = []any{map[string]string{"id": h.Environment + "/httpRouteConfigs/unexpected"}}
			case "endpoint":
				w.pages[h.Environment+"/privateEndpointConnections"]["value"] = []any{map[string]string{"id": h.Environment + "/privateEndpointConnections/unexpected"}}
			case "revision":
				revision := w.pages[h.Application+"/revisions"]["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)
				template := objectMap(revision["template"])
				template["containers"].([]any)[0].(map[string]any)["image"] = s.Placement.Services.Relay.Image
				revision["template"] = template
			case "pending":
				props["provisioningState"] = "Updating"
				expected = hostedexecution.ErrUnavailable
			}
			reader, err := NewSDKHost(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			err = reader.Check(t.Context(), s.Worker)
			if err != expected || strings.Contains(err.Error(), "canary") {
				t.Fatal("unsafe host admitted or diagnostics leaked", err)
			}
		})
	}
}

func TestSandboxControllerIdentitiesRejectSharingFederationAndTenantDrift(t *testing.T) {
	for _, mode := range []string{"sharing", "federation", "tenant"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			switch mode {
			case "sharing":
				other := unrelatedServiceWorkload(s, "jobs")
				other["identity"] = map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{s.Authority.Control.Resource: map[string]string{"clientId": s.ControlClientID, "principalId": s.Authority.Control.PrincipalID}}}
				addServiceWorkload(w, "jobs", other)
			case "federation":
				id := s.Authority.Metadata.Resource
				w.pages[id+"/federatedIdentityCredentials"]["value"] = []any{map[string]string{"id": id + "/federatedIdentityCredentials/unexpected"}}
			case "tenant":
				w.resources[s.Host.Startup.Resource]["properties"].(map[string]any)["tenantId"] = "other"
			}
			reader, err := NewSDKServiceWorkloads(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
				t.Fatal("controller identity boundary drift admitted", err)
			}
		})
	}
}

func TestSandboxRuntimeHostBlocksAllAdmissionsAndClosesItsReader(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings("discovery")
	w := newPlacementWorld(t, s)
	protection := &placementProtection{}
	runtime, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: protection, ImageAdmission: protection, ARMTransport: w, Credentials: func(string) (azcore.TokenCredential, error) {
		return &runtimeCredential{token: "placement-metadata-canary"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	w.resources[s.Host.Application]["properties"].(map[string]any)["template"].(map[string]any)["containers"].([]any)[0].(map[string]any)["args"].([]any)[0] = "--assembly-only"
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("host drift reached import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("host drift reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || protection.verifies.Load() != 0 {
		t.Fatal("fresh host failure retained safe live evidence", err)
	}
	runtime.Close()
	if err := runtime.host.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("runtime left host reader open", err)
	}
}
