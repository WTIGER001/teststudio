package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

const initialPassword = "silver meadow river sunrise 42"
const changedPassword = "violet ocean quiet evening 51"
const resetPassword = "golden forest clear morning 87"

func passwordFixture(t *testing.T, loopback ...bool) *client {
	t.Helper()
	return fixture(t, func(api *httpapi.Server) {
		if len(loopback) > 0 && !loopback[0] {
			api.LocalUser = ""
		}
		api.Sessions.LocalAccounts = true
		service, err := localauth.New(t.Context(), api.Sessions)
		if err != nil {
			t.Fatal(err)
		}
		api.LocalAccounts = service
	})
}

func passwordClient(t *testing.T, c *client) *client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &client{http: &http.Client{Jar: jar}, origin: c.origin, t: t, api: c.api}
}

func passwordLogin(t *testing.T, c *client, user, password string, required bool) {
	t.Helper()
	result := accessCall(t, c, "POST", "/auth/password", map[string]string{"userId": user, "password": password}, "", 200)
	if result["mustChangePassword"] != required {
		t.Fatal(result)
	}
	session := accessCall(t, c, "GET", "/auth/session", nil, "", 200)["session"].(map[string]any)
	c.csrf = session["csrfToken"].(string)
	if session["authenticationMethod"] != "password" {
		t.Fatal("password method was inferred from a provider name", session)
	}
}

func TestLocalAccountsRequirePasswordChangeRotateSessionsAndRedactSecrets(t *testing.T) {
	c := passwordFixture(t)
	c.signIn()
	providers := accessCall(t, c, "GET", "/auth/providers", nil, "", 200)
	if providers["localAccountsEnabled"] != true || providers["localEnabled"] != true {
		t.Fatal(providers)
	}
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "local-reviewer", "name": "Local reviewer"}, "", 200)
	path := "/api/v1/users/local-reviewer/local-credential"
	empty := accessCall(t, c, "GET", path, nil, "", 200)
	if empty["revision"] != float64(0) || empty["data"].(map[string]any)["configured"] != false {
		t.Fatal(empty)
	}
	accessCall(t, c, "PUT", path, map[string]any{"password": initialPassword, "reason": "Cannot bypass required change", "mustChangePassword": false}, "0", 422)
	created := accessCall(t, c, "PUT", path, map[string]string{"password": initialPassword, "reason": "Provision local sign-in"}, "0", 200)
	if created["revision"] != float64(1) || created["data"].(map[string]any)["mustChangePassword"] != true {
		t.Fatal(created)
	}
	accessCall(t, c, "PUT", path, map[string]string{"password": resetPassword, "reason": "Reject stale reset"}, "0", 409)
	user := passwordClient(t, c)
	passwordLogin(t, user, "local-reviewer", initialPassword, true)
	accessCall(t, user, "GET", "/api/v1/projects", nil, "", 403)
	accessCall(t, user, "GET", "/api/v1/users", nil, "", 403)
	origin, _ := url.Parse(c.origin)
	oldCookie := user.http.Jar.Cookies(origin)[0]
	accessCall(t, user, "POST", "/auth/password/change", map[string]string{"currentPassword": "incorrect", "newPassword": changedPassword}, "", 422)
	accessCall(t, user, "POST", "/auth/password/change", map[string]string{"currentPassword": initialPassword, "newPassword": initialPassword}, "", 422)
	status, _ := user.call("POST", "/auth/password/change", map[string]string{"currentPassword": initialPassword, "newPassword": changedPassword}, map[string]string{"X-CSRF-Token": "wrong"})
	if status != 403 {
		t.Fatal("password change accepted missing session proof", status)
	}
	accessCall(t, user, "POST", "/auth/password/change", map[string]string{"currentPassword": initialPassword, "newPassword": changedPassword}, "", 200)
	current := accessCall(t, user, "GET", "/auth/session", nil, "", 200)
	currentSession := current["session"].(map[string]any)
	user.csrf = currentSession["csrfToken"].(string)
	if currentSession["mustChangePassword"] == true || oldCookie.Value == user.http.Jar.Cookies(origin)[0].Value {
		t.Fatal("password change retained a restricted session or cookie")
	}
	request := httptest.NewRequest("GET", c.origin+"/api/v1/me", nil)
	request.AddCookie(oldCookie)
	if _, err := c.api.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("old cookie survived password change", err)
	}
	accessCall(t, user, "GET", "/api/v1/me", nil, "", 200)
	own := accessCall(t, user, "GET", "/api/v1/me/local-credential", nil, "", 200)
	if own["data"].(map[string]any)["canManage"] != false {
		t.Fatal("self-service credential read implied installation authority", own)
	}
	accessCall(t, user, "GET", "/api/v1/users/admin/local-credential", nil, "", 403)
	projects := accessCall(t, user, "GET", "/api/v1/projects", nil, "", 200)
	if len(projects["items"].([]any)) != 0 {
		t.Fatal("local authentication implicitly granted project access", projects)
	}
	anonymous := passwordClient(t, c)
	failed := accessCall(t, anonymous, "POST", "/auth/password", map[string]string{"userId": "local-reviewer", "password": initialPassword}, "", 401)
	unknown := accessCall(t, anonymous, "POST", "/auth/password", map[string]string{"userId": "unknown-account", "password": initialPassword}, "", 401)
	if failed["message"] != unknown["message"] {
		t.Fatal("login errors disclose credential existence")
	}
	passwordLogin(t, anonymous, "local-reviewer", changedPassword, false)
	active := accessCall(t, c, "GET", path, nil, "", 200)
	accessCall(t, c, "POST", path+"/disable", map[string]string{"reason": "Disable only local sign-in"}, "2", 200)
	if _, err := c.api.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal(err)
	}
	accessCall(t, user, "GET", "/api/v1/me", nil, "", 401)
	accessCall(t, anonymous, "GET", "/api/v1/me", nil, "", 401)
	accessCall(t, c, "POST", path+"/enable", map[string]string{"reason": "Enable reviewed local sign-in"}, "3", 200)
	passwordLogin(t, anonymous, "local-reviewer", changedPassword, false)
	accessCall(t, c, "PUT", path, map[string]string{"password": resetPassword, "reason": "Reset a forgotten password"}, "4", 200)
	accessCall(t, anonymous, "GET", "/api/v1/me", nil, "", 401)
	passwordLogin(t, anonymous, "local-reviewer", resetPassword, true)
	accessCall(t, anonymous, "POST", "/auth/logout", map[string]any{}, "", 200)
	var encoded strings.Builder
	for _, value := range []any{created, active, accessCall(t, c, "GET", path, nil, "", 200), accessCall(t, c, "GET", "/api/v1/users/local-reviewer", nil, "", 200)} {
		data, _ := json.Marshal(value)
		encoded.Write(data)
	}
	audit, err := c.api.Store.All(t.Context(), "audit", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range audit {
		encoded.Write(row.Data)
	}
	for _, secret := range []string{initialPassword, changedPassword, resetPassword, "$argon2id$", "\"hash\""} {
		if strings.Contains(encoded.String(), secret) {
			t.Fatal("credentials leaked into administration responses or audit")
		}
	}
}

func TestLocalAccountsPreserveLastAdministratorAndMixedIdentitySessions(t *testing.T) {
	c := passwordFixture(t, false)
	assignmentSignInAs(t, c, "admin")
	path := "/api/v1/users/admin/local-credential"
	accessCall(t, c, "PUT", path, map[string]string{"password": initialPassword, "reason": "Prepare managed local administration"}, "0", 200)
	local := passwordClient(t, c)
	passwordLogin(t, local, "admin", initialPassword, true)
	accessCall(t, local, "POST", "/auth/password/change", map[string]string{"currentPassword": initialPassword, "newPassword": changedPassword}, "", 200)
	current := accessCall(t, local, "GET", "/auth/session", nil, "", 200)
	local.csrf = current["session"].(map[string]any)["csrfToken"].(string)
	before, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	failure := accessCall(t, local, "POST", path+"/disable", map[string]string{"reason": "Cannot disable final local sign-in"}, "2", 422)
	if !strings.Contains(failure["message"].(string), "last usable") {
		t.Fatal(failure)
	}
	after, err := c.api.Auth.Directory.Revision(t.Context())
	if err != nil || after.Revision != before.Revision {
		t.Fatal("rejected credential change advanced the directory clock", after, err)
	}
	user := accessCall(t, local, "GET", "/api/v1/users/admin", nil, "", 200)
	accessCall(t, local, "POST", "/api/v1/users/admin/disable", map[string]string{"reason": "Protect the last local account"}, assignmentRevision(user), 422)
	assignment := accessCall(t, local, "GET", "/api/v1/role-assignments/user/admin/builtin.teststudio.installation-admin", nil, "", 200)
	accessCall(t, local, "DELETE", "/api/v1/role-assignments/user/admin/builtin.teststudio.installation-admin", map[string]string{"reason": "Protect the last local role"}, assignmentRevision(assignment), 422)
	w := httptest.NewRecorder()
	oidcSession, err := c.api.Sessions.Create(t.Context(), w, "admin", "password", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal("an OIDC provider identifier collided with password authentication", err)
	}
	if _, err := c.api.Sessions.Validate(t.Context(), oidcSession.ID); err != nil {
		t.Fatal(err)
	}
	if err := localauth.Configure(t.Context(), c.api.Auth.Directory, false); err != nil {
		t.Fatal(err)
	}
	c.api.Sessions.LocalAccounts = false
	origin, _ := url.Parse(c.origin)
	request := httptest.NewRequest("GET", c.origin+"/auth/session", nil)
	request.AddCookie(local.http.Jar.Cookies(origin)[0])
	if _, err := c.api.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("disabled local account mode preserved a password cookie", err)
	}
	if _, err := c.api.Sessions.Validate(t.Context(), oidcSession.ID); err != nil {
		t.Fatal("disabled password mode invalidated an OIDC provider with the same name", err)
	}
	if err := localauth.Configure(t.Context(), c.api.Auth.Directory, true); err != nil {
		t.Fatal(err)
	}
	c.api.Sessions.LocalAccounts = true
	if _, err := c.api.Sessions.Read(t.Context(), request); !errors.Is(err, sessions.ErrUnauthenticated) {
		t.Fatal("reenabling local accounts revived an old password cookie", err)
	}

}

func TestLocalAccountsRejectCrossOriginRateLimitAndProjectAdministration(t *testing.T) {
	c := passwordFixture(t)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "operator", "name": "Operator"}, "", 200)
	accessCall(t, c, "PUT", "/api/v1/users/operator/local-credential", map[string]string{"password": initialPassword, "reason": "Configure local account"}, "0", 200)
	anonymous := passwordClient(t, c)
	for _, origin := range []string{"", "https://untrusted.example"} {
		status, _ := anonymous.call("POST", "/auth/password", map[string]string{"userId": "operator", "password": initialPassword}, map[string]string{"Origin": origin})
		if status != 403 {
			t.Fatal("login accepted an invalid origin", status)
		}
	}
	for range 10 {
		accessCall(t, anonymous, "POST", "/auth/password", map[string]string{"userId": "nonexistent-user", "password": initialPassword}, "", 401)
	}
	accessCall(t, anonymous, "POST", "/auth/password", map[string]string{"userId": "nonexistent-user", "password": initialPassword}, "", 429)
	rows, err := c.api.Store.All(t.Context(), "local-auth-attempts", "")
	if err != nil || len(rows) != 2 {
		t.Fatal("limiter did not use bounded hashed slots", len(rows), err)
	}
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "alpha", "name": "Alpha"}, "", 200)
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "operator"}, "builtin.teststudio.project-admin", map[string]any{"team": "alpha"}); err != nil {
		t.Fatal(err)
	}
	assignmentSignInAs(t, c, "operator")
	accessCall(t, c, "GET", "/api/v1/users/admin/local-credential", nil, "", 403)
	accessCall(t, c, "PUT", "/api/v1/users/admin/local-credential", map[string]string{"password": resetPassword, "reason": "Reject project takeover"}, "0", 403)
	accessCall(t, c, "POST", "/api/v1/users/admin/local-credential/disable", map[string]string{"reason": "Reject project takeover"}, "1", 403)
}

func TestLocalAccountsPasswordResetInvalidatesProofBeforeSessionCreation(t *testing.T) {
	c := passwordFixture(t)
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/users", map[string]string{"id": "operator", "name": "Operator"}, "", 200)
	path := "/api/v1/users/operator/local-credential"
	accessCall(t, c, "PUT", path, map[string]string{"password": initialPassword, "reason": "Configure local account"}, "0", 200)
	accessCall(t, c, "PUT", path, map[string]string{"password": resetPassword, "reason": "Revoke earlier password proof"}, "1", 200)
	if _, cookie, err := c.api.Sessions.CreatePassword(t.Context(), "operator", 1, ""); !errors.Is(err, sessions.ErrUnauthenticated) || cookie != nil {
		t.Fatal("old password proof created a session after reset", err)
	}
	record, err := c.api.Store.Get(t.Context(), "local-credentials", "operator")
	if err != nil {
		t.Fatal(err)
	}
	if record.Revision != 2 {
		t.Fatal(record)
	}
	if _, err := c.api.Store.Get(t.Context(), "sessions", "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal(err)
	}
}
