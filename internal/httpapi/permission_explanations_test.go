package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
)

func explanationURL(user, project, object, permission string) string {
	path := "/api/v1/me/effective-permissions"
	if user != "" {
		path = "/api/v1/users/" + user + "/effective-permissions"
	}
	q := url.Values{"project": {project}, "object": {object}, "permission": {"teststudio." + permission}}
	return path + "?" + q.Encode()
}
func explanationItems(value map[string]any) []map[string]any {
	items := []map[string]any{}
	for _, item := range value["items"].([]any) {
		items = append(items, item.(map[string]any))
	}
	return items
}

func TestPermissionExplanationUsesRealDeniesInheritedGroupsAndRetainedExpiry(t *testing.T) {
	c := roleSetup(t)
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "reviewers", "name": "Reviewers", "projectId": "alpha"}, "", 200)
	accessCall(t, c, "PUT", "/api/v1/groups/reviewers/members/member", map[string]any{}, "0", 200)
	accessCall(t, c, "POST", "/api/v1/roles", roleInput("leaf-reader", "alpha", roleRule("catalog.read", permissions.EffectAllow, "suite/health")), "", 200)
	parent := roleInput("composite-reader", "alpha")
	parent.Inherits = []string{"leaf-reader"}
	accessCall(t, c, "POST", "/api/v1/roles", parent, "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", map[string]string{"principalKind": "group", "principalId": "reviewers", "roleId": parent.ID, "reason": "Verify inherited source paths"}, "", 200)
	base := "/api/v1/projects/alpha/permissions"
	accessCall(t, c, "POST", base, directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
	deny := accessCall(t, c, "POST", base, directGrant("group", "reviewers", "catalog.read", "suite/health", permissions.EffectDeny), "", 200)
	expired := directGrant("user", "member", "catalog.read", "suite/health", permissions.EffectDeny)
	expiry := time.Now().Add(-time.Hour)
	expired.ExpiresAt = &expiry
	accessCall(t, c, "POST", base, expired, "", 200)
	version, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	audit, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	path := explanationURL("member", "alpha", "suite/health", "catalog.read")
	result := accessCall(t, c, "GET", path, nil, "", 200)
	if result["allowed"] != false || result["reason"] != "explicit-deny" || result["partial"] != false || len(explanationItems(result)) != 4 {
		t.Fatal(result)
	}
	foundRole, foundExpired := false, false
	for _, source := range explanationItems(result) {
		if source["restricted"] != false {
			t.Fatal("root source unexpectedly restricted", source)
		}
		if source["status"] == "expired" {
			foundExpired = true
		}
		if source["kind"] == "role" {
			foundRole = true
			if source["roleId"] != "leaf-reader" || source["assignedRoleId"] != parent.ID || source["principalId"] != "reviewers" || source["scopeKnown"] != true {
				t.Fatal(source)
			}
		}
	}
	if !foundRole || !foundExpired {
		t.Fatal(result)
	}
	current, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || version.Revision != current.Revision {
		t.Fatal("read-only explanation advanced directory", err)
	}
	afterAudit, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil || len(audit) != len(afterAudit) {
		t.Fatal("explanation wrote an audit mutation", err)
	}
	accessCall(t, c, "PUT", base+"/"+deny["id"].(string), access.GrantUpdate{ExpiresAt: &expiry, Reason: "Expire the matching group deny"}, assignmentRevision(deny), 200)
	result = accessCall(t, c, "GET", path, nil, "", 200)
	if result["allowed"] != true || result["reason"] != "allowed" || len(explanationItems(result)) != 4 {
		t.Fatal(result)
	}
	accessCall(t, c, "POST", "/api/v1/groups/reviewers/archive", map[string]string{"reason": "Withdraw inherited group paths"}, "1", 200)
	result = accessCall(t, c, "GET", path, nil, "", 200)
	if result["allowed"] != true || len(explanationItems(result)) != 2 {
		t.Fatal("archived group remained a source", result)
	}
	u := accessCall(t, c, "GET", "/api/v1/users/member", nil, "", 200)
	accessCall(t, c, "POST", "/api/v1/users/member/disable", map[string]string{"reason": "Disable the inspected subject"}, assignmentRevision(u), 200)
	result = accessCall(t, c, "GET", path, nil, "", 200)
	if result["allowed"] != false || result["userDisabled"] != true || result["reason"] != "user-disabled" || len(explanationItems(result)) != 0 {
		t.Fatal(result)
	}
}

func TestPermissionExplanationSelfInspectionAndManagementBoundaries(t *testing.T) {
	c := roleSetup(t)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "member", "viewer"), "", 200)
	assignmentSignInAs(t, c, "member")
	result := accessCall(t, c, "GET", explanationURL("", "alpha", "suite/health", "catalog.read"), nil, "", 200)
	if result["allowed"] != true || result["canInspectOthers"] != false || len(explanationItems(result)) != 1 {
		t.Fatal(result)
	}
	source := explanationItems(result)[0]
	if source["restricted"] != true || source["roleId"] != nil || source["principalId"] != nil {
		t.Fatal("self inspection leaked an unreadable definition", source)
	}
	accessCall(t, c, "GET", explanationURL("manager", "alpha", "suite/health", "catalog.read"), nil, "", 403)
	accessCall(t, c, "GET", explanationURL("", "beta", "suite/health", "catalog.read"), nil, "", 403)
	missing := accessCall(t, c, "GET", explanationURL("", "", "", "system.manage"), nil, "", 200)
	if missing["allowed"] != false || missing["reason"] != "no-matching-allow" || len(explanationItems(missing)) != 0 {
		t.Fatal(missing)
	}
	c.signIn()
	viewer := accessCall(t, c, "PUT", "/api/v1/predefined-roles/builtin.teststudio.viewer", builtinChange(true), "1", 200)
	missing = accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/health", "catalog.read"), nil, "", 200)
	if missing["allowed"] != false || missing["reason"] != "no-matching-allow" || len(explanationItems(missing)) != 0 {
		t.Fatal("disabled role remained an active policy path", missing)
	}
	accessCall(t, c, "PUT", "/api/v1/predefined-roles/builtin.teststudio.viewer", builtinChange(false), assignmentRevision(viewer), 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/permissions", directGrant("user", "manager", "catalog.read", "suite/health", permissions.EffectAllow), "", 200)
	assignmentSignInAs(t, c, "manager")
	result = accessCall(t, c, "GET", explanationURL("", "alpha", "suite/health", "catalog.read"), nil, "", 200)
	if result["allowed"] != true {
		t.Fatal("exact-object self access required unrelated project permission", result)
	}
	accessCall(t, c, "GET", explanationURL("", "alpha", "", "catalog.read"), nil, "", 403)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	assignmentSignInAs(t, c, "manager")
	accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/health", "catalog.read"), nil, "", 200)
	accessCall(t, c, "GET", explanationURL("member", "beta", "suite/health", "catalog.read"), nil, "", 403)
	accessCall(t, c, "GET", explanationURL("member", "", "", "system.manage"), nil, "", 403)
	accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/*", "catalog.read"), nil, "", 422)
	accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/missing", "catalog.read"), nil, "", 404)
	accessCall(t, c, "GET", explanationURL("member", "alpha", "", "system.manage"), nil, "", 422)
}

func TestPermissionExplanationRedactsForeignAndInstallationPolicySources(t *testing.T) {
	c := roleSetup(t)
	private := "private-beta-policy"
	accessCall(t, c, "POST", "/api/v1/roles", roleInput(private, "beta", roleRule("catalog.read", permissions.EffectAllow, "")), "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/beta/role-assignments", map[string]string{"principalKind": "user", "principalId": "member", "roleId": private, "reason": "Verify cross-scope source privacy"}, "", 200)
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: private, PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow, TeamScope: "*"}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", "/api/v1/role-assignments", assignBody("user", "member", "installation-admin"), "", 200)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	path := explanationURL("member", "alpha", "suite/health", "catalog.read")
	root := accessCall(t, c, "GET", path, nil, "", 200)
	if root["allowed"] != true || len(explanationItems(root)) != 2 {
		t.Fatal(root)
	}
	boundSource := false
	for _, source := range explanationItems(root) {
		if source["assignedRoleId"] == private {
			boundSource = true
			if source["assignmentProject"] != "beta" {
				t.Fatal("authorized explanation lost the actual foreign assignment binding", source)
			}
		}
	}
	if !boundSource {
		t.Fatal("foreign assignment path was not explained", root)
	}
	assignmentSignInAs(t, c, "manager")
	result := accessCall(t, c, "GET", path, nil, "", 200)
	for _, source := range explanationItems(result) {
		if source["restricted"] != true || source["roleId"] != nil || source["teamScope"] != nil || source["assignmentProject"] != nil {
			t.Fatal("foreign source leaked", source)
		}
	}
	raw, _ := json.Marshal(result)
	for _, secret := range []string{private, "builtin.teststudio.installation-admin"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("foreign source identifier leaked", secret)
		}
	}
	if explanationItems(result)[0]["id"] == explanationItems(root)[0]["id"] {
		t.Fatal("source identifiers correlate across inspectors")
	}
}

func TestPermissionExplanationLeavesFieldDenySemanticsToTheLibrary(t *testing.T) {
	c := roleSetup(t)
	object := "alpha/suite/health"
	for _, effect := range []permissions.Effect{permissions.EffectAllow, permissions.EffectDeny} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio.catalog.read", Effect: effect, TeamScope: "alpha", ObjectScope: &object, RestrictedFields: []string{"title"}}); err != nil {
			t.Fatal(err)
		}
	}
	result := accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/health", "catalog.read"), nil, "", 200)
	if result["allowed"] != true || result["reason"] != "allowed" || len(explanationItems(result)) != 1 {
		t.Fatal("field-only deny reinterpreted as permission deny", result)
	}
	if explanationItems(result)[0]["effect"] != "allow" {
		t.Fatal(result)
	}
}

func TestPermissionExplanationBoundedScansBindContextActorAndDirectoryVersion(t *testing.T) {
	c := roleSetup(t)
	for start := 0; start < 1101; start += 100 {
		grants := []permissions.Grant{}
		for index := start; index < start+100 && index < 1101; index++ {
			object := fmt.Sprintf("alpha/suite/unrelated-%04d", index)
			if index == 1100 {
				object = "alpha/suite/health"
			}
			grants = append(grants, permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow, TeamScope: "alpha", ObjectScope: &object})
		}
		if err := c.api.Auth.Engine.CreateGrants(t.Context(), grants); err != nil {
			t.Fatal(err)
		}
	}
	path := explanationURL("member", "alpha", "suite/health", "catalog.read")
	first := accessCall(t, c, "GET", path+"&limit=1000", nil, "", 200)
	if first["allowed"] != true || first["reason"] != "allowed" || len(explanationItems(first)) != 0 || first["partial"] != true {
		t.Fatal("bounded empty scan misreported access", first)
	}
	cursor := url.QueryEscape(first["cursor"].(string))
	last := accessCall(t, c, "GET", path+"&limit=1000&cursor="+cursor, nil, "", 200)
	if last["partial"] != false || len(explanationItems(last)) != 1 {
		t.Fatal(last)
	}
	accessCall(t, c, "GET", explanationURL("member", "alpha", "suite/health", "runs.read")+"&cursor="+cursor, nil, "", 422)
	accessCall(t, c, "GET", explanationURL("member", "beta", "suite/health", "catalog.read")+"&cursor="+cursor, nil, "", 422)
	accessCall(t, c, "POST", "/api/v1/projects/alpha/role-assignments", assignBody("user", "manager", "project-admin"), "", 200)
	fresh := accessCall(t, c, "GET", path+"&limit=1", nil, "", 200)
	assignmentSignInAs(t, c, "manager")
	accessCall(t, c, "GET", path+"&cursor="+url.QueryEscape(fresh["cursor"].(string)), nil, "", 422)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "later-directory-change", "name": "Later directory change"}, "", 200)
	accessCall(t, c, "GET", path+"&cursor="+url.QueryEscape(fresh["cursor"].(string)), nil, "", 409)
}

func TestPermissionExplanationCursorRejectsExpiryWithoutDirectoryWrites(t *testing.T) {
	c := roleSetup(t)
	object := "alpha/suite/health"
	expiry := time.Now().Add(500 * time.Millisecond)
	for _, deadline := range []*time.Time{&expiry, nil} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "member", PermissionName: "teststudio.catalog.read", Effect: permissions.EffectAllow, TeamScope: "alpha", ObjectScope: &object, ExpiresAt: deadline}); err != nil {
			t.Fatal(err)
		}
	}
	path := explanationURL("member", "alpha", "suite/health", "catalog.read")
	first := accessCall(t, c, "GET", path+"&limit=1", nil, "", 200)
	if len(explanationItems(first)) != 1 || first["partial"] != true {
		t.Fatal(first)
	}
	time.Sleep(time.Until(expiry) + 10*time.Millisecond)
	accessCall(t, c, "GET", path+"&cursor="+url.QueryEscape(first["cursor"].(string)), nil, "", 409)
	result := accessCall(t, c, "GET", path, nil, "", 200)
	if result["allowed"] != true || len(explanationItems(result)) != 2 {
		t.Fatal(result)
	}
}
