package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func accessCall(t *testing.T, c *client, method, path string, value any, revision string, want int) map[string]any {
	t.Helper()
	headers := map[string]string{}
	if revision != "" {
		headers["If-Match"] = revision
	}
	status, result := c.call(method, path, value, headers)
	if status != want {
		t.Fatalf("%s %s: got %d want %d: %v", method, path, status, want, result)
	}
	return result
}

func TestAccessUsersKeepServerMetadataAndRevokeSessionsAcrossRestore(t *testing.T) {
	c := fixture(t)
	accessCall(t, c, "GET", "/api/v1/users", nil, "", 401)
	c.signIn()
	definition := map[string]any{"id": "operator", "name": "Operator", "email": "operator@example.test"}
	created := accessCall(t, c, "POST", "/api/v1/users", definition, "", 200)
	if created["revision"] != float64(1) || created["data"].(map[string]any)["emailVerified"] != false {
		t.Fatal(created)
	}
	accessCall(t, c, "POST", "/api/v1/users", definition, "", 409)
	accessCall(t, c, "PUT", "/api/v1/users/operator", map[string]any{"id": "operator", "name": "Changed", "disabled": true}, "1", 422)
	accessCall(t, c, "PUT", "/api/v1/users/operator", definition, "0", 422)
	accessCall(t, c, "PUT", "/api/v1/users/other", definition, "1", 422)
	w := httptest.NewRecorder()
	v, err := c.api.Sessions.Create(t.Context(), w, "operator", "example", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	cookie := w.Result().Cookies()[0]
	view := accessCall(t, c, "GET", "/api/v1/users/operator/sessions", nil, "", 200)
	encoded, _ := json.Marshal(view)
	if strings.Contains(string(encoded), v.CSRF) || strings.Contains(string(encoded), "csrfToken") || strings.Contains(string(encoded), cookie.Value) {
		t.Fatal("session list leaked credentials")
	}
	if len(view["items"].([]any)) != 1 {
		t.Fatal(view)
	}
	profile := accessCall(t, c, "GET", "/api/v1/users/operator", nil, "", 200)
	if profile["revision"] != float64(2) || profile["data"].(map[string]any)["lastSignIn"] == nil {
		t.Fatal("missing durable sign-in time")
	}
	accessCall(t, c, "POST", "/api/v1/users/operator/disable", map[string]string{"reason": "Account suspended"}, "1", 409)
	disabled := accessCall(t, c, "POST", "/api/v1/users/operator/disable", map[string]string{"reason": "Account suspended"}, "2", 200)
	if disabled["data"].(map[string]any)["sessionEpoch"] != float64(1) {
		t.Fatal(disabled)
	}
	request := httptest.NewRequest(http.MethodGet, c.origin+"/auth/session", nil)
	request.AddCookie(cookie)
	if _, err := c.api.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatalf("disabled session: %v", err)
	}
	accessCall(t, c, "POST", "/api/v1/users/operator/restore", map[string]string{"reason": "Account reviewed"}, "3", 200)
	if _, err := c.api.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatalf("restoration revived session: %v", err)
	}
	if _, err := c.api.Sessions.Validate(t.Context(), v.ID); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatalf("artifact session survived restore: %v", err)
	}
	definition["name"] = "Reviewed operator"
	updated := accessCall(t, c, "PUT", "/api/v1/users/operator", definition, "4", 200)
	if updated["data"].(map[string]any)["createdAt"] != created["data"].(map[string]any)["createdAt"] {
		t.Fatal("profile edit changed creation time")
	}
	admin := accessCall(t, c, "GET", "/api/v1/users/admin", nil, "", 200)
	if admin["revision"] != float64(2) {
		t.Fatal(admin)
	}
	blocked := accessCall(t, c, "POST", "/api/v1/users/admin/disable", map[string]string{"reason": "Would remove final administrator"}, "2", 422)
	if !strings.Contains(blocked["message"].(string), "last usable") {
		t.Fatal(blocked)
	}
	rows, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if strings.Contains(string(row.Data), "user.disable") {
			found = true
			if !strings.Contains(string(row.Data), "Account suspended") {
				t.Fatal("missing audit reason")
			}
		}
	}
	if !found {
		t.Fatal("user change has no audit record")
	}
}

func TestAccessGroupScopesProvenanceAndMembershipRevisions(t *testing.T) {
	c := fixture(t)
	c.signIn()
	for _, id := range []string{"alpha", "beta"} {
		accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": id, "name": id}, "", 200)
	}
	for _, id := range []string{"manager", "member"} {
		accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": id, "name": id}, "", 200)
	}
	for _, d := range []map[string]string{{"id": "alpha-group", "name": "Alpha group", "projectId": "alpha"}, {"id": "beta-group", "name": "Beta group", "projectId": "beta"}, {"id": "shared-group", "name": "Shared group"}} {
		accessCall(t, c, "POST", "/api/v1/groups", d, "", 200)
	}
	accessCall(t, c, "PUT", "/api/v1/groups/alpha-group/members/member", map[string]any{}, "0", 200)
	single := accessCall(t, c, "GET", "/api/v1/groups/alpha-group/members/member", nil, "", 200)
	if single["revision"] != float64(1) || single["data"].(map[string]any)["source"] != "local" {
		t.Fatal(single)
	}
	accessCall(t, c, "PUT", "/api/v1/groups/alpha-group/members/member", map[string]any{}, "0", 409)
	page := accessCall(t, c, "GET", "/api/v1/groups/alpha-group/members", nil, "", 200)
	member := page["items"].([]any)[0].(map[string]any)["data"].(map[string]any)
	if member["source"] != "local" || member["readOnly"] != false || member["active"] != true {
		t.Fatal(member)
	}
	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	accessCall(t, c, "PUT", "/api/v1/groups/alpha-group/members/member", map[string]string{"expiresAt": expires}, "1", 200)
	accessCall(t, c, "DELETE", "/api/v1/groups/alpha-group/members/member", map[string]string{"reason": "Remove local membership"}, "1", 409)
	accessCall(t, c, "DELETE", "/api/v1/groups/alpha-group/members/member", map[string]string{"reason": "Remove local membership"}, "2", 200)
	if err := c.api.Auth.Directory.SaveGroup(t.Context(), identity.Group{ID: "external", Name: "External group", Project: "alpha", ManagedBy: "oidc:example"}, 0); err != nil {
		t.Fatal(err)
	}
	provider, err := storage.Put("providers", "example", "", 0, map[string]any{"issuer": "https://identity.example.test", "clientId": "teststudio", "disabled": false})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.api.Store.Commit(t.Context(), provider); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Directory.SyncProviderGroups(t.Context(), "member", "example", 1, []string{"external"}, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	page = accessCall(t, c, "GET", "/api/v1/groups/external/members", nil, "", 200)
	external := accessCall(t, c, "GET", "/api/v1/groups/external/members/member?source=oidc:example", nil, "", 200)
	if external["data"].(map[string]any)["readOnly"] != true {
		t.Fatal(external)
	}
	if page["items"].([]any)[0].(map[string]any)["data"].(map[string]any)["readOnly"] != true {
		t.Fatal(page)
	}
	groups := accessCall(t, c, "GET", "/api/v1/users/member/memberships", nil, "", 200)
	if len(groups["items"].([]any)) != 1 || groups["items"].([]any)[0].(map[string]any)["data"].(map[string]any)["groupName"] != "External group" {
		t.Fatal(groups)
	}
	accessCall(t, c, "PUT", "/api/v1/groups/external/members/member", map[string]any{}, "0", 422)
	accessCall(t, c, "DELETE", "/api/v1/groups/external/members/member", map[string]string{"reason": "Attempt source override"}, "1", 422)
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "manager"}, "builtin.teststudio.project-admin", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	if _, err := c.api.Sessions.Create(t.Context(), w, "manager", "example", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	origin, err := url.Parse(c.origin)
	if err != nil {
		t.Fatal(err)
	}
	c.http.Jar.SetCookies(origin, w.Result().Cookies())
	_, session := c.call("GET", "/auth/session", nil, nil)
	c.csrf = session["session"].(map[string]any)["csrfToken"].(string)
	me := accessCall(t, c, "GET", "/api/v1/me", nil, "", 200)
	if me["user"].(map[string]any)["id"] != "manager" || me["localAccount"] != false {
		t.Fatal(me)
	}
	accessCall(t, c, "GET", "/api/v1/users", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/users/member", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/groups", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/groups?project=beta", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/groups/beta-group", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/groups/beta-group/members/member", nil, "", 403)
	accessCall(t, c, "GET", "/api/v1/groups/shared-group", nil, "", 403)
	page = accessCall(t, c, "GET", "/api/v1/groups?project=alpha", nil, "", 200)
	if len(page["items"].([]any)) != 2 {
		t.Fatal(page)
	}
	accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": "escape", "name": "Escape", "projectId": "beta"}, "", 403)
	accessCall(t, c, "PUT", "/api/v1/groups/beta-group", map[string]string{"id": "beta-group", "name": "Reparent", "projectId": "alpha"}, "1", 403)
	accessCall(t, c, "PUT", "/api/v1/groups/alpha-group", map[string]string{"id": "alpha-group", "name": "Reparent", "projectId": "beta"}, "1", 422)
	accessCall(t, c, "PUT", "/api/v1/groups/alpha-group/members/member", map[string]any{}, "0", 200)
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("group membership silently granted project access", err)
	}
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: "alpha-group"}, "builtin.teststudio.viewer", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "catalog.read"); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "POST", "/api/v1/groups/alpha-group/archive", map[string]string{"reason": "Group retired"}, "1", 200)
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "catalog.read"); !errors.Is(err, authorization.ErrForbidden) {
		t.Fatal("archived group retained access", err)
	}
	accessCall(t, c, "PUT", "/api/v1/groups/alpha-group/members/member", map[string]any{}, "1", 422)
	accessCall(t, c, "POST", "/api/v1/groups/alpha-group/restore", map[string]string{"reason": "Group needed again"}, "2", 200)
	if err := c.api.Auth.Check(t.Context(), "member", "alpha", "", "catalog.read"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.api.Store.Get(t.Context(), "memberships", identity.MembershipID("member", "alpha-group", "local")); err != nil {
		t.Fatal(err)
	}
}

func TestAccessSessionRevocationChecksOwnershipAndDoesNotLeakSecrets(t *testing.T) {
	c := fixture(t)
	c.signIn()
	page := accessCall(t, c, "GET", "/api/v1/users/admin/sessions", nil, "", 200)
	rows := page["items"].([]any)
	if len(rows) != 1 {
		t.Fatal(page)
	}
	row := rows[0].(map[string]any)
	id := row["id"].(string)
	accessCall(t, c, "DELETE", "/api/v1/users/other/sessions/"+id, map[string]string{"reason": "Wrong owner"}, "1", 404)
	accessCall(t, c, "DELETE", "/api/v1/users/admin/sessions/"+id, map[string]string{"reason": "End current session"}, "0", 409)
	accessCall(t, c, "DELETE", "/api/v1/users/admin/sessions/"+id, map[string]string{"reason": "End current session"}, "1", 200)
	accessCall(t, c, "GET", "/api/v1/users", nil, "", 401)
	if _, err := c.api.Store.Get(t.Context(), "sessions", id); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestAccessPagesFilterObjectDeniesAndBindOpaqueCursors(t *testing.T) {
	c := fixture(t)
	c.signIn()
	for _, id := range []string{"alpha", "beta"} {
		accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": id, "name": id}, "", 200)
	}
	for _, id := range []string{"hidden", "visible"} {
		accessCall(t, c, "POST", "/api/v1/groups", map[string]string{"id": id, "name": id, "projectId": "alpha"}, "", 200)
	}
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "member", "name": "Member"}, "", 200)
	accessCall(t, c, "PUT", "/api/v1/groups/hidden/members/member", map[string]any{}, "0", 200)
	accessCall(t, c, "PUT", "/api/v1/groups/visible/members/member", map[string]any{}, "0", 200)
	memberObject := "alpha/group/visible"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "alpha", ObjectScope: &memberObject, PermissionName: "teststudio.members.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "GET", "/api/v1/groups/visible/members", nil, "", 403)
	object := "alpha/group/hidden"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "alpha", ObjectScope: &object, PermissionName: "teststudio.groups.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	memberships := accessCall(t, c, "GET", "/api/v1/users/member/memberships", nil, "", 200)
	if len(memberships["items"].([]any)) != 0 {
		t.Fatal("user detail leaked a denied group", memberships)
	}
	page := accessCall(t, c, "GET", "/api/v1/groups?project=alpha&limit=1", nil, "", 200)
	if len(page["items"].([]any)) != 0 || page["partial"] != true {
		t.Fatal("filtered scan coverage was not explicit", page)
	}
	cursor := page["cursor"].(string)
	page = accessCall(t, c, "GET", "/api/v1/groups?project=alpha&limit=1&cursor="+url.QueryEscape(cursor), nil, "", 200)
	if len(page["items"].([]any)) != 1 || page["items"].([]any)[0].(map[string]any)["id"] != "visible" {
		t.Fatal(page)
	}
	accessCall(t, c, "GET", "/api/v1/groups?project=beta&limit=1&cursor="+url.QueryEscape(cursor), nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/groups?project=alpha&archived=true&limit=1&cursor="+url.QueryEscape(cursor), nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/users?cursor="+url.QueryEscape(cursor), nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/groups?project=alpha&cursor=forged", nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/groups?project=alpha&limit=0", nil, "", 422)
	accessCall(t, c, "GET", "/api/v1/groups?project=alpha&limit=1001", nil, "", 422)
	status, _ := c.call("POST", "/api/v1/users", map[string]string{"id": "csrf-rejected", "name": "Rejected"}, map[string]string{"X-CSRF-Token": ""})
	if status != 403 {
		t.Fatal("missing CSRF token accepted", status)
	}
	if _, err := c.api.Store.Get(t.Context(), "users", "csrf-rejected"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("rejected mutation wrote a user", err)
	}
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "member"}, "builtin.teststudio.project-admin", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	if _, err := c.api.Sessions.Create(t.Context(), w, "member", "example", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	origin, err := url.Parse(c.origin)
	if err != nil {
		t.Fatal(err)
	}
	c.http.Jar.SetCookies(origin, w.Result().Cookies())
	accessCall(t, c, "GET", "/api/v1/groups?project=alpha&limit=1&cursor="+url.QueryEscape(cursor), nil, "", 422)
}

func TestAccessLegacyUserViewsIncludeSessionGenerationWithoutRewritingHistory(t *testing.T) {
	c := fixture(t)
	c.signIn()
	change, err := storage.Put("users", "legacy-user", "", 0, map[string]any{"id": "legacy-user", "name": "Legacy user", "disabled": false, "emailVerified": false, "createdAt": time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	view := accessCall(t, c, "GET", "/api/v1/users/legacy-user", nil, "", 200)
	if view["data"].(map[string]any)["sessionEpoch"] != float64(0) || view["revision"] != float64(1) {
		t.Fatal(view)
	}
	page := accessCall(t, c, "GET", "/api/v1/users", nil, "", 200)
	for _, value := range page["items"].([]any) {
		row := value.(map[string]any)
		if row["id"] == "legacy-user" && row["data"].(map[string]any)["sessionEpoch"] != float64(0) {
			t.Fatal(row)
		}
	}
	raw, err := c.api.Store.Get(t.Context(), "users", "legacy-user")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw.Data), "sessionEpoch") || raw.Revision != 1 {
		t.Fatal("compatible reads rewrote the original record")
	}
}
