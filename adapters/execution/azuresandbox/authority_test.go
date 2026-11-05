package azuresandbox

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func rolePermission(w *placementWorld, id string) map[string]any {
	return w.resources[id]["properties"].(map[string]any)["permissions"].([]any)[0].(map[string]any)
}

func TestSandboxAuthorityDefinitionsSeparateDataControlFromTimerManagement(t *testing.T) {
	for _, mode := range []string{"execution", "discovery"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings(mode)
			defs, err := s.AuthorityDefinitions()
			if err != nil || len(defs) != 11 {
				t.Fatal("missing role or grant definitions", err)
			}
			perm := func(id string) map[string]any {
				return defs[id].(map[string]any)["properties"].(map[string]any)["permissions"].([]any)[0].(map[string]any)
			}
			stop := perm(s.Authority.StopGrant.Role)
			if !exactActions(stop["dataActions"].([]string), []string{"Microsoft.App/sandboxGroups/sandboxes/read", "Microsoft.App/sandboxGroups/sandboxes/stop/action"}) || len(stop["actions"].([]string)) != 0 {
				t.Fatal("timer can create or execute code")
			}
			controller := perm(s.Authority.ControlGrant.Role)
			expected := []string{"Microsoft.App/sandboxGroups/diskimages/read", "Microsoft.App/sandboxGroups/sandboxes/read", "Microsoft.App/sandboxGroups/sandboxes/write", "Microsoft.App/sandboxGroups/sandboxes/delete", "Microsoft.App/sandboxGroups/sandboxes/executeCommand/action", "Microsoft.App/sandboxGroups/sandboxes/stop/action"}
			if mode == "discovery" {
				expected = append(expected, "Microsoft.App/sandboxGroups/diskimages/write")
			}
			if !exactActions(controller["dataActions"].([]string), expected) || len(controller["actions"].([]string)) != 0 {
				t.Fatal("controller received another mode or management authority")
			}
			workflow := perm(s.Authority.WorkflowGrant.Role)
			if !exactActions(workflow["actions"].([]string), []string{"Microsoft.Logic/workflows/read", "Microsoft.Logic/workflows/write", "Microsoft.Logic/workflows/delete"}) || len(workflow["dataActions"].([]string)) != 0 {
				t.Fatal("timer administrator received secret or data access")
			}
			grant := defs[s.Authority.AssignGrant.Assignment].(map[string]any)["properties"].(map[string]any)
			if grant["scope"] != s.Timer.StopIdentity || grant["principalId"] != s.Authority.Control.PrincipalID {
				t.Fatal("controller can attach another identity")
			}
		})
	}
}

func TestSandboxAuthorityRejectsChangedIdentityScopeAndPermissions(t *testing.T) {
	for _, mode := range []string{"control-client", "stop-principal", "controller-scope", "workflow-scope", "assign-target", "metadata-write", "stop-execute", "execution-import", "registry-push", "role-exception"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			switch mode {
			case "control-client":
				props(s.Authority.Control.Resource)["clientId"] = s.RegistryClientID
			case "stop-principal":
				props(s.Timer.StopIdentity)["principalId"] = s.Authority.Control.PrincipalID
			case "controller-scope":
				props(s.Authority.ControlGrant.Assignment)["scope"] = "/subscriptions/" + s.Worker.Subscription
			case "workflow-scope":
				props(s.Authority.WorkflowGrant.Assignment)["scope"] = resourceGroupID(s.Worker.groupID())
			case "assign-target":
				props(s.Authority.AssignGrant.Assignment)["scope"] = s.PullIdentity
			case "metadata-write":
				rolePermission(w, s.Authority.MetadataGrant.Role)["actions"] = []string{"*"}
			case "stop-execute":
				permission := rolePermission(w, s.Authority.StopGrant.Role)
				permission["dataActions"] = append(permission["dataActions"].([]string), "Microsoft.App/sandboxGroups/sandboxes/executeCommand/action")
			case "execution-import":
				permission := rolePermission(w, s.Authority.ControlGrant.Role)
				permission["dataActions"] = append(permission["dataActions"].([]string), "Microsoft.App/sandboxGroups/diskimages/write")
			case "registry-push":
				props(s.Authority.RegistryAssignment)["roleDefinitionId"] = "/subscriptions/" + s.Worker.Subscription + "/providers/Microsoft.Authorization/roleDefinitions/8311e382-0749-4cb8-b61a-304f252e45ec"
			case "role-exception":
				rolePermission(w, s.Authority.StopGrant.Role)["notDataActions"] = []string{"Microsoft.App/sandboxGroups/sandboxes/stop/action"}
			}
			reader, err := NewSDKAuthority(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
				t.Fatal("incorrect selected authority was admitted", err)
			}
		})
	}
}

func TestSandboxAuthoritySharesBoundedEvidenceAndRetainsFailures(t *testing.T) {
	s := runtimeSettings()
	w := newPlacementWorld(t, s)
	w.resources[s.Authority.RegistryAssignment]["properties"].(map[string]any)["roleDefinitionId"] = pullRole
	credential := &runtimeCredential{token: "placement-metadata-canary"}
	reader, err := NewSDKAuthority(s, credential, w)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	now := time.Now()
	reader.now = func() time.Time { return now }
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			if err := reader.Check(t.Context(), s.Worker); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if len(w.calls) != 15 {
		t.Fatal("selected authority scope changed", len(w.calls))
	}
	for _, count := range w.calls {
		if count != 1 {
			t.Fatal("concurrent authority repeated metadata reads")
		}
	}
	now = now.Add(metadataRefresh)
	w.status = 404
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable || strings.Contains(err.Error(), "canary") {
		t.Fatal("missing authority became worker absence or exposed diagnostics", err)
	}
	w.status = 0
	rolePermission(w, s.Authority.StopGrant.Role)["dataActions"] = []string{"*"}
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
		t.Fatal("failed authority was cached or broadened role accepted", err)
	}
	before := len(credential.scopes)
	if _, err := reader.get(t.Context(), s.Worker.groupID()); err != hostedexecution.ErrOwnership || len(credential.scopes) != before {
		t.Fatal("authority reader escaped its selected resources", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := reader.Check(ctx, s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("canceled check accepted authority", err)
	}
	reader.Close()
	if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed authority reader returned cached success", err)
	}
}

func TestSandboxAuthorityBindingsRejectBeforeCredentialConstruction(t *testing.T) {
	for _, mode := range []string{"missing", "identity-reuse", "client-reuse", "principal-reuse", "role-reuse", "broad-assignment", "foreign-role"} {
		t.Run(mode, func(t *testing.T) {
			f := newControllerFixture(t, "go", "")
			s := runtimeSettings()
			switch mode {
			case "missing":
				s.Authority = AuthoritySettings{}
			case "identity-reuse":
				s.Authority.Metadata.Resource = s.Authority.Control.Resource
			case "client-reuse":
				s.Authority.StopClientID = s.ControlClientID
			case "principal-reuse":
				s.Authority.StopPrincipalID = s.Authority.Registry.PrincipalID
			case "role-reuse":
				s.Authority.StopGrant.Role = s.Authority.ControlGrant.Role
			case "broad-assignment":
				s.Authority.ControlGrant.Assignment = s.Authority.MetadataGrant.Assignment
			case "foreign-role":
				s.Authority.ControlGrant.Role = strings.Replace(s.Authority.ControlGrant.Role, s.Worker.Subscription, "foreign", 1)
			}
			_, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: f.controls, Credentials: func(string) (azcore.TokenCredential, error) {
				t.Error("invalid authority constructed a credential")
				return nil, nil
			}})
			if err != hostedexecution.ErrConfiguration {
				t.Fatal("invalid authority binding accepted", err)
			}
		})
	}
}

func TestSandboxRuntimeAuthorityGateCannotBeOmittedOrHideProtectionLoss(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings("discovery")
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
	protection.denied.Store(true)
	if err := r.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "instance", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrUnavailable {
		t.Fatal("authority cache hid uncached protection", err)
	}
	protection.denied.Store(false)
	r.authority.now = func() time.Time { return time.Now().Add(metadataRefresh) }
	w.resources[s.Authority.WorkflowGrant.Assignment]["properties"].(map[string]any)["scope"] = resourceGroupID(s.Worker.groupID())
	if err := r.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || protection.imports.Load() != 0 {
		t.Fatal("factory omitted selected timer authority", err)
	}
}
