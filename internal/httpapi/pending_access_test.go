package httpapi_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func pendingSetup(t *testing.T) *client {
	t.Helper()
	c := assignmentSetup(t)
	provider := providerInput("pending-provider", false)
	provider.Name = "Allowed sign-in provider"
	accessCall(t, c, "POST", "/api/v1/identity-providers", provider, "", 200)
	return c
}

func pendingInput() access.PendingAccessInput {
	return access.PendingAccessInput{ProviderID: "pending-provider", ProviderRevision: 1, Email: "Pending@EXAMPLE.TEST", RoleID: "builtin.teststudio.viewer", Reason: "Approve project access before first sign-in"}
}

func pendingProof(t *testing.T, c *client, subject, email string) authoidc.SignInIdentity {
	t.Helper()
	user, err := c.api.Auth.Directory.ResolveProviderIdentity(t.Context(), "pending-provider", 1, "https://identity.example.test", subject, "Pending user", email, true, true)
	if err != nil {
		t.Fatal(err)
	}
	return authoidc.SignInIdentity{UserID: user.ID, ProviderID: "pending-provider", ProviderRevision: 1, Issuer: "https://identity.example.test", Subject: subject, Email: email, EmailVerified: true, ExpiresAt: time.Now().Add(time.Hour)}
}

func TestPendingAccessUsesVerifiedProviderClaimsAndNeverMergesAccounts(t *testing.T) {
	c := pendingSetup(t)
	base := "/api/v1/projects/alpha/pending-access"
	before, _ := c.api.Store.All(t.Context(), "users", "")
	created := accessCall(t, c, "POST", base, pendingInput(), "", 200)
	data := created["data"].(map[string]any)
	if data["email"] != "Pending@example.test" || data["state"] != "pending" || data["requireVerifiedEmail"] != true {
		t.Fatal("invalid pending contract", created)
	}
	after, _ := c.api.Store.All(t.Context(), "users", "")
	if len(after) != len(before) {
		t.Fatal("pending access created a user")
	}
	accessCall(t, c, "POST", base, pendingInput(), "", 409)
	encoded, _ := json.Marshal(created)
	if string(encoded) == "" || data["roleDigest"] != nil {
		t.Fatal("private approval fingerprint leaked")
	}
	resource := base + "/" + created["id"].(string)
	proof := pendingProof(t, c, "new-subject", "Pending@example.test")
	service := c.api.OIDC.ProjectAccess
	unverified := proof
	unverified.EmailVerified = false
	if err := service.ApplyPendingAccess(t.Context(), unverified); err != nil {
		t.Fatal(err)
	}
	roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", false)
	mismatch := proof
	mismatch.Email = "pending@example.test"
	if err := service.ApplyPendingAccess(t.Context(), mismatch); err != nil {
		t.Fatal(err)
	}
	roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", false)
	anotherProvider := proof
	anotherProvider.ProviderID = "different-provider"
	if err := service.ApplyPendingAccess(t.Context(), anotherProvider); err != nil {
		t.Fatal(err)
	}
	roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", false)
	old, err := c.api.Auth.Directory.ResolveExternal(t.Context(), "https://another.example.test", "existing-user", "Existing user", proof.Email, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ApplyPendingAccess(t.Context(), proof); err != nil {
		t.Fatal(err)
	}
	roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", true)
	roleAllowed(t, c, proof.UserID, "beta", "", "projects.read", false)
	roleAllowed(t, c, proof.UserID, "", "", "members.manage", false)
	roleAllowed(t, c, old.ID, "alpha", "", "projects.read", false)
	link, err := c.api.Auth.Directory.ExternalLink(t.Context(), proof.Issuer, proof.Subject)
	if err != nil || link.UserID != proof.UserID || proof.UserID == old.ID {
		t.Fatal("pending access merged identity owners", err)
	}
	activated := accessCall(t, c, "GET", resource, nil, "", 200)
	activatedData := activated["data"].(map[string]any)
	if activatedData["state"] != "activated" || activatedData["activatedUserId"] != proof.UserID || activatedData["assignmentId"] == nil || activatedData["canCancel"] != false {
		t.Fatal("activation metadata missing", activated)
	}
	assignments := accessCall(t, c, "GET", "/api/v1/projects/alpha/role-assignments", nil, "", 200)
	row := assignments["items"].([]any)[0].(map[string]any)
	assignmentPath := "/api/v1/projects/alpha/role-assignments/user/" + proof.UserID + "/builtin.teststudio.viewer"
	accessCall(t, c, "DELETE", assignmentPath, map[string]string{"reason": "Revoke activated access"}, assignmentRevision(row), 200)
	if err := service.ApplyPendingAccess(t.Context(), proof); err != nil {
		t.Fatal(err)
	}
	roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", false)

	object := "alpha/user/" + proof.UserID
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.members.read", Effect: permissions.EffectDeny, TeamScope: "alpha", ObjectScope: &object}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "GET", resource, nil, "", 403)
	filtered := accessCall(t, c, "GET", base, nil, "", 200)
	if len(filtered["items"].([]any)) != 0 {
		t.Fatal("activated recipient deny leaked its pending history", filtered)
	}
	audits, err := c.api.Store.All(t.Context(), "audit", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, row := range audits {
		var event struct{ Action string }
		_ = json.Unmarshal(row.Data, &event)
		counts[event.Action]++
	}
	if counts["project-access.prepare"] != 1 || counts["project-access.activate"] != 1 || counts["role.assign"] != 1 {
		t.Fatal("pending access or role assignment audit repeated", counts)
	}
}

func TestPendingAccessRestrictsProjectManagersAndPreservesCancellationRevisions(t *testing.T) {
	c := pendingSetup(t)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	manager := passwordClient(t, c)
	assignmentSignInAs(t, manager, "manager")
	base := "/api/v1/projects/alpha/pending-access"
	choices := accessCall(t, manager, "GET", base+"/providers", nil, "", 200)
	raw, _ := json.Marshal(choices)
	var decoded map[string]any
	_ = json.Unmarshal(raw, &decoded)
	if len(decoded["items"].([]any)) != 1 {
		t.Fatal("missing provider choices")
	}
	choice := decoded["items"].([]any)[0].(map[string]any)
	if choice["clientId"] != nil || choice["secretRef"] != nil || choice["revision"] != float64(1) {
		t.Fatal("private provider configuration leaked", choice)
	}
	accessCall(t, manager, "GET", "/api/v1/identity-providers", nil, "", 403)
	created := accessCall(t, manager, "POST", base, pendingInput(), "", 200)
	resource := base + "/" + created["id"].(string)
	foreign := "/api/v1/projects/beta/pending-access"
	for _, method := range []string{"GET", "POST"} {
		var body any
		if method == "POST" {
			body = pendingInput()
		}
		accessCall(t, manager, method, foreign, body, "", 403)
	}
	accessCall(t, manager, "GET", foreign+"/providers", nil, "", 403)
	accessCall(t, manager, "GET", foreign+"/"+created["id"].(string), nil, "", 403)
	accessCall(t, manager, "POST", foreign+"/"+created["id"].(string)+"/cancel", map[string]string{"reason": "Reject cross-project cancellation"}, "1", 403)
	invalid := pendingInput()
	invalid.RoleID = "builtin.teststudio.installation-admin"
	accessCall(t, manager, "POST", base, invalid, "", 422)
	invalid = pendingInput()
	invalid.ProviderRevision = 0
	accessCall(t, manager, "POST", base, invalid, "", 422)
	invalid.ProviderRevision = 2
	accessCall(t, manager, "POST", base, invalid, "", 409)
	invalid = pendingInput()
	invalid.Email = "Display <Pending@example.test>"
	accessCall(t, manager, "POST", base, invalid, "", 422)
	for _, headers := range []map[string]string{{"Origin": "https://other.example"}, {"X-CSRF-Token": "wrong"}} {
		if status, _ := manager.call("POST", base, pendingInput(), headers); status != 403 {
			t.Fatal("mutation protection bypassed", status)
		}
	}
	accessCall(t, manager, "POST", resource+"/cancel", map[string]string{"reason": "Cancel pending access"}, "0", 409)
	cancelled := accessCall(t, manager, "POST", resource+"/cancel", map[string]string{"reason": "Cancel pending access"}, "1", 200)
	if cancelled["data"].(map[string]any)["state"] != "cancelled" {
		t.Fatal(cancelled)
	}
	accessCall(t, manager, "POST", resource+"/cancel", map[string]string{"reason": "Reject stale cancellation"}, "1", 409)
	recreated := accessCall(t, manager, "POST", base, pendingInput(), "", 200)
	if recreated["id"] == created["id"] {
		t.Fatal("new approval overwrote cancelled history")
	}
	accessCall(t, manager, "POST", resource+"/cancel", map[string]string{"reason": "Reject old cancellation"}, "2", 422)
	if latest := accessCall(t, manager, "GET", base+"/"+recreated["id"].(string), nil, "", 200); latest["data"].(map[string]any)["state"] != "pending" {
		t.Fatal("old cancellation affected new access")
	}
}

func TestPendingAccessBlocksChangedApprovalAndDoesNotRestoreItSilently(t *testing.T) {
	for _, change := range []string{"owner-revoked", "provider-changed", "role-changed", "inherited-role-changed", "project-archived", "recipient-deny"} {
		t.Run(change, func(t *testing.T) {
			c := pendingSetup(t)
			roleAssignment := accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
			manager := passwordClient(t, c)
			assignmentSignInAs(t, manager, "manager")
			base := "/api/v1/projects/alpha/pending-access"
			input := pendingInput()
			if change == "role-changed" || change == "inherited-role-changed" {
				policy := roleInput("pending-reader", "alpha", roleRule("projects.read", permissions.EffectAllow, ""))
				accessCall(t, c, "POST", "/api/v1/roles", policy, "", 200)
				input.RoleID = policy.ID
				if change == "inherited-role-changed" {
					parent := roleInput("pending-composite", "alpha")
					parent.Inherits = []string{policy.ID}
					accessCall(t, c, "POST", "/api/v1/roles", parent, "", 200)
					input.RoleID = parent.ID
				}
			}
			created := accessCall(t, manager, "POST", base, input, "", 200)
			proof := pendingProof(t, c, "new-subject", "Pending@example.test")
			expected := "authority-changed"
			switch change {
			case "owner-revoked":
				accessCall(t, c, "DELETE", "/api/v1/projects/alpha/role-assignments/user/manager/builtin.teststudio.project-admin", map[string]string{"reason": "Revoke the approving owner"}, assignmentRevision(roleAssignment), 200)
			case "provider-changed":
				provider := providerInput("pending-provider", false)
				provider.Name = "Reviewed provider"
				accessCall(t, c, "PUT", "/api/v1/identity-providers/pending-provider", provider, "1", 200)
				proof.ProviderRevision = 2
				expected = "provider-changed"
			case "role-changed", "inherited-role-changed":
				policy := roleInput("pending-reader", "alpha", roleRule("projects.read", permissions.EffectAllow, ""), roleRule("runs.read", permissions.EffectAllow, ""))
				accessCall(t, c, "PUT", "/api/v1/roles/pending-reader", policy, "1", 200)
				expected = "role-changed"
			case "project-archived":
				row, err := c.api.Store.Get(t.Context(), "projects", "alpha")
				if err != nil {
					t.Fatal(err)
				}
				state, err := storage.Decode[map[string]any](row)
				if err != nil {
					t.Fatal(err)
				}
				state["archived"] = true
				value, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, state)
				if err := c.api.Store.Commit(t.Context(), value); err != nil {
					t.Fatal(err)
				}
				expected = "project-unavailable"
			case "recipient-deny":
				deny := permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "manager", PermissionName: "teststudio.members.manage", Effect: permissions.EffectDeny, TeamScope: "alpha", ObjectScope: ptr("alpha/user/" + proof.UserID)}
				if err := c.api.Auth.Engine.CreateGrant(t.Context(), deny); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.api.OIDC.ProjectAccess.ApplyPendingAccess(t.Context(), proof); err != nil {
				t.Fatal("ineligible approval should withhold access without blocking sign-in", err)
			}
			roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", false)
			status := accessCall(t, c, "GET", base+"/"+created["id"].(string), nil, "", 200)["data"].(map[string]any)
			if status["state"] != "blocked" || status["blockReason"] != expected {
				t.Fatal("approval not blocked", status)
			}
			if err := c.api.OIDC.ProjectAccess.ApplyPendingAccess(t.Context(), proof); err != nil {
				t.Fatal(err)
			}
			roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", false)
		})
	}
}

func ptr(value string) *string { return &value }

func TestPendingAccessExactObjectDenyFiltersRowsAndBoundCursors(t *testing.T) {
	c := pendingSetup(t)
	first := accessCall(t, c, "POST", "/api/v1/projects/alpha/pending-access", pendingInput(), "", 200)
	input := pendingInput()
	input.Email = "Another@example.test"
	accessCall(t, c, "POST", "/api/v1/projects/alpha/pending-access", input, "", 200)
	object := "pending-access/" + first["id"].(string)
	query := explanationURL("admin", "alpha", object, "members.read")
	accessCall(t, c, "GET", query, nil, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "admin", "members.read", object, permissions.EffectDeny), "", 200)
	accessCall(t, c, "GET", query, nil, "", 403)
	page := accessCall(t, c, "GET", "/api/v1/projects/alpha/pending-access?limit=1", nil, "", 200)
	if len(page["items"].([]any)) != 0 || page["cursor"] == nil {
		t.Fatal("denied first row did not preserve bounded cursor", page)
	}
	cursor := fmt.Sprint(page["cursor"])
	accessCall(t, c, "GET", "/api/v1/projects/beta/pending-access?cursor="+cursor, nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/projects/alpha/pending-access/"+first["id"].(string), nil, "", 403)
	var forbidden = authorization.ErrForbidden
	if !errors.Is(c.api.Auth.Check(t.Context(), "admin", "alpha", "pending-access/"+first["id"].(string), "members.read"), forbidden) {
		t.Fatal("pending object deny was bypassed")
	}
}

func TestPendingAccessCancellationAndActivationSerializeAcrossInstances(t *testing.T) {
	c := pendingSetup(t)
	base := "/api/v1/projects/alpha/pending-access"
	created := accessCall(t, c, "POST", base, pendingInput(), "", 200)
	proof := pendingProof(t, c, "racing-subject", "Pending@example.test")
	local := ""
	if c.api.Store.Dialect == "sqlite" {
		if err := c.api.Store.DB.QueryRowContext(t.Context(), "SELECT file FROM pragma_database_list WHERE name='main'").Scan(&local); err != nil {
			t.Fatal(err)
		}
	}
	db, err := storage.Open(t.Context(), testkit.ConnectionString(t, c.api.Store, local))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), &identity.Directory{Store: db})
	if err != nil {
		t.Fatal(err)
	}
	second := &access.Service{Store: db, Auth: auth}
	first := &access.Service{Store: c.api.Store, Auth: c.api.Auth, LocalUser: c.api.LocalUser}
	ready := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-ready
		_, err := first.CancelPendingAccess(t.Context(), "admin", "alpha", created["id"].(string), "Cancel competing pending approval", 1)
		results <- err
	}()
	go func() { <-ready; results <- second.ApplyPendingAccess(t.Context(), proof) }()
	close(ready)
	for range 2 {
		err := <-results
		if err != nil && !errors.Is(err, storage.ErrConflict) && !strings.Contains(err.Error(), "only pending or blocked") {
			t.Fatal("unexpected competing access error", err)
		}
	}
	value := accessCall(t, c, "GET", base+"/"+created["id"].(string), nil, "", 200)["data"].(map[string]any)
	if value["state"] != "cancelled" && value["state"] != "activated" {
		t.Fatal("no operation completed", value)
	}
	roleAllowed(t, c, proof.UserID, "alpha", "", "projects.read", value["state"] == "activated")
}

func TestPendingAccessBatchFailureRollsBackNativeRolesAndActivationMetadata(t *testing.T) {
	c := pendingSetup(t)
	for _, project := range []string{"alpha", "beta"} {
		accessCall(t, c, "POST", "/api/v1/projects/"+project+"/pending-access", pendingInput(), "", 200)
	}
	proof := pendingProof(t, c, "rollback-subject", "Pending@example.test")
	rows, err := c.api.Store.DB.QueryContext(t.Context(), "SELECT bucket,id FROM records WHERE bucket LIKE 'pending-access-matches/%' ORDER BY id ASC")
	if err != nil {
		t.Fatal(err)
	}
	var bucket, id string
	count := 0
	for rows.Next() {
		if err := rows.Scan(&bucket, &id); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if count != 2 {
		t.Fatal("missing pending indexes", count)
	}
	index, err := c.api.Store.Get(t.Context(), bucket, id)
	if err != nil {
		t.Fatal(err)
	}
	broken, _ := storage.Put(bucket, id, index.Project, index.Revision, map[string]string{"id": "unavailable-approval", "projectId": index.Project})
	if err := c.api.Store.Commit(t.Context(), broken); err != nil {
		t.Fatal(err)
	}
	before, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.api.OIDC.ProjectAccess.ApplyPendingAccess(t.Context(), proof); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("expected late batch rejection", err)
	}
	after, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || before.Revision != after.Revision {
		t.Fatal("failed batch changed directory revision", err)
	}
	for _, project := range []string{"alpha", "beta"} {
		roleAllowed(t, c, proof.UserID, project, "", "projects.read", false)
		page := accessCall(t, c, "GET", "/api/v1/projects/"+project+"/pending-access", nil, "", 200)
		if len(page["items"].([]any)) != 1 || page["items"].([]any)[0].(map[string]any)["data"].(map[string]any)["state"] != "pending" {
			t.Fatal("partial activation metadata escaped rollback", page)
		}
	}
	assignments, err := c.api.Auth.Engine.RoleAssignmentsForPrincipal(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: proof.UserID})
	if err != nil || len(assignments) != 0 {
		t.Fatal("native assignment escaped rollback", len(assignments), err)
	}
}
