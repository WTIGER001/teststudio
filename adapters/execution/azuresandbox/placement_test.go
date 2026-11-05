package azuresandbox

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type placementTrackedBody struct {
	io.ReadCloser
	closed bool
}

func (b *placementTrackedBody) Close() error { b.closed = true; return b.ReadCloser.Close() }

func TestSandboxPlacementReadsSelectedResourcesAndComposesBothAdmissionGates(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings()
	s.Mode = "discovery"
	w := newPlacementWorld(t, s)
	credential := &runtimeCredential{token: "placement-metadata-canary"}
	protection := &placementProtection{}
	r, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: protection, ImageAdmission: protection, ARMTransport: w, Credentials: func(id string) (azcore.TokenCredential, error) {
		if id == s.MetadataClientID {
			return credential, nil
		}
		return &runtimeCredential{token: "unused-control-or-registry"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != nil {
		t.Fatal(err)
	}
	if protection.creates.Load() != 1 || protection.imports.Load() != 1 || len(w.calls) != 69 {
		t.Fatal("native assembly omitted a selected observation or protection gate")
	}
	for id := range placementVersions(s) {
		count := w.calls[id]
		if id == s.Placement.Subnet {
			count -= 2
		}
		if count != 2 {
			t.Fatal("placement observations were cached or retried", count)
		}
	}
	for id := range authorityVersions(s) {
		if w.calls[id] != 1 {
			t.Fatal("authority observations were not shared")
		}
	}
	actual := Snapshot{ID: "owned-instance", State: "Running", DefinitionHash: hash("verified by controller before admission")}
	for range 2 {
		if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, actual); err != nil {
			t.Fatal(err)
		}
	}
	protection.denied.Store(true)
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, actual); err != hostedexecution.ErrUnavailable {
		t.Fatal("placement cache hid protection loss", err)
	}
	protection.denied.Store(false)
	for id := range placementVersions(s) {
		count := w.calls[id]
		if id == s.Placement.Subnet {
			count -= 2
		}
		if count != 2 {
			t.Fatal("cached placement repeated ARM reads")
		}
	}
	for id := range authorityVersions(s) {
		if w.calls[id] != 1 {
			t.Fatal("cached authority repeated ARM reads")
		}
	}
	props := w.resources[s.Worker.connectionID()]["properties"].(map[string]any)
	props["subnetId"] = strings.Replace(s.Placement.Subnet, "/sandbox", "/other-subnet", 1)
	if err := r.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || protection.creates.Load() != 1 {
		t.Fatal("changed placement reached dispatch protection", err)
	}
	props["subnetId"] = s.Placement.Subnet
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, actual); err != nil || protection.verifies.Load() != 4 {
		t.Fatal("live placement did not recheck", err)
	}
	if w.calls[s.Placement.PullAssignment] != 3 || w.calls[s.Worker.groupID()] != 4 {
		t.Fatal("live checks repeated source-only registry observations")
	}
	if len(credential.scopes) == 0 {
		t.Fatal("metadata credential was not selected")
	}
	for _, scope := range credential.scopes {
		if scope != "https://management.core.windows.net//.default" {
			t.Fatal("metadata credential audience changed")
		}
	}
	r.Close()
	if err := r.placement.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed placement reader used credentials", err)
	}
}

func TestSandboxPlacementRejectsReportedOwnershipAndImportDrift(t *testing.T) {
	for _, mode := range []string{"group-environment", "group-identity", "subnet-security", "subnet-route", "subnet-egress", "subnet-delegation", "pull-tenant", "pull-client", "registry-public", "registry-abac", "grant-principal", "grant-scope", "grant-condition", "role-authority"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			prop := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			switch mode {
			case "group-environment":
				prop(s.Worker.groupID())["environmentId"] = "a-different-api-shape"
			case "group-identity":
				w.resources[s.Worker.groupID()]["identity"].(map[string]any)["type"] = "SystemAssigned, UserAssigned"
			case "subnet-security":
				prop(s.Placement.Subnet)["networkSecurityGroup"] = map[string]string{"id": "foreign"}
			case "subnet-route":
				prop(s.Placement.Subnet)["routeTable"] = map[string]string{"id": "foreign"}
			case "subnet-egress":
				prop(s.Placement.Subnet)["defaultOutboundAccess"] = true
			case "subnet-delegation":
				prop(s.Placement.Subnet)["delegations"] = []any{}
			case "pull-tenant":
				prop(s.PullIdentity)["tenantId"] = "a-different-tenant"
			case "pull-client":
				prop(s.PullIdentity)["clientId"] = s.ControlClientID
			case "registry-public":
				prop(s.Placement.Registry)["publicNetworkAccess"] = "Enabled"
			case "registry-abac":
				prop(s.Placement.Registry)["roleAssignmentMode"] = "AbacRepositoryPermissions"
			case "grant-principal":
				prop(s.Placement.PullAssignment)["principalId"] = s.ControlClientID
			case "grant-scope":
				prop(s.Placement.PullAssignment)["scope"] = "/subscriptions/" + s.Worker.Subscription
			case "grant-condition":
				prop(s.Placement.PullAssignment)["condition"] = "different-condition"
			case "role-authority":
				prop("/subscriptions/" + s.Worker.Subscription + pullRole)["permissions"] = []any{map[string]any{"actions": []string{"*"}}}
			}
			reader, err := NewSDKPlacement(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
				t.Fatal("reported drift was admitted", err)
			}
		})
	}
}

func TestSandboxPlacementMetadataGapsNeverBecomeWorkerAbsence(t *testing.T) {
	for _, status := range []int{404, 403, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			w.status = status
			reader, err := NewSDKPlacement(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable || strings.Contains(err.Error(), "canary") {
				t.Fatal("metadata gap became absence or leaked diagnostics", err)
			}
			for _, count := range w.calls {
				if count != 1 {
					t.Fatal("metadata read was implicitly retried")
				}
			}
		})
	}
}

func TestSandboxPlacementSettingsBindProfilesBeforeCredentialConstruction(t *testing.T) {
	for _, mode := range []string{"missing-reader", "reader-reuse", "pull-reuse", "profile", "public-subnet", "foreign-subnet", "grant-scope"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			switch mode {
			case "missing-reader":
				s.MetadataClientID = ""
			case "reader-reuse":
				s.MetadataClientID = s.ControlClientID
			case "pull-reuse":
				s.RegistryClientID = s.Placement.PullClientID
			case "profile":
				s.Placement.PolicyProfile = hash("changed protected policy")
			case "public-subnet":
				s.Placement.SubnetPrefix = "198.51.100.0/24"
			case "foreign-subnet":
				s.Placement.Subnet = strings.Replace(s.Placement.Subnet, s.Worker.Subscription, "foreign", 1)
			case "grant-scope":
				s.Placement.PullAssignment = "/subscriptions/" + s.Worker.Subscription + "/providers/Microsoft.Authorization/roleAssignments/77777777-7777-7777-7777-777777777777"
			}
			if s.Validate() != hostedexecution.ErrConfiguration {
				t.Fatal("unbound native placement was accepted")
			}
		})
	}
}

func TestSandboxPlacementRejectsUnselectedRequestsAndCanceledObservation(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	credential := &runtimeCredential{token: "placement-metadata-canary"}
	reader, err := NewSDKPlacement(s, credential, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if _, err := reader.get(t.Context(), s.Worker.groupID()+"/other"); err != hostedexecution.ErrOwnership || len(credential.scopes) != 0 {
		t.Fatal("unselected resource obtained a credential", err)
	}
	for _, change := range []string{"mutation", "version", "redirect-host", "filtered"} {
		request, _ := http.NewRequest("GET", "https://management.azure.com"+s.Worker.groupID()+"?api-version="+APIVersion, nil)
		switch change {
		case "mutation":
			request.Method = "PUT"
		case "version":
			request.URL.RawQuery = "api-version=2026-07-01"
		case "redirect-host":
			request.URL.Host = "foreign.example"
		case "filtered":
			request.URL.RawQuery += "&$filter=other"
		}
		if _, err := reader.http.Do(request); err != hostedexecution.ErrOwnership {
			t.Fatal("unselected request reached the transport", err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.Check(ctx, s.Worker); err != hostedexecution.ErrUnavailable || len(w.calls) != 0 {
		t.Fatal("canceled observation read metadata", err)
	}
	for _, mode := range []string{"redirect", "oversize-header", "oversize-body"} {
		resp := response(200, "sensitive-provider-canary")
		switch mode {
		case "redirect":
			resp.StatusCode = 302
			resp.Header.Set("Location", "https://foreign.example")
		case "oversize-header":
			resp.ContentLength = (1 << 20) + 1
		case "oversize-body":
			resp = response(200, strings.Repeat("a", (1<<20)+1))
			resp.ContentLength = -1
		}
		body := &placementTrackedBody{ReadCloser: resp.Body}
		resp.Body = body
		guard := newPlacementHTTP(s, runtimeTransport(func(*http.Request) (*http.Response, error) { return resp, nil }))
		request, _ := http.NewRequest("GET", "https://management.azure.com"+s.Worker.groupID()+"?api-version="+APIVersion, nil)
		_, err := guard.Do(request)
		guard.Close()
		if err != hostedexecution.ErrUnavailable || !body.closed {
			t.Fatal("bounded response escaped or leaked its body", mode, err)
		}
	}
}
