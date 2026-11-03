package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/cliauth"
)

func TestCLIBrowserConfirmationProtectsConsentAndReturnsProofBoundSessionOnlyOnce(t *testing.T) {
	c := fixture(t)
	c.signIn()
	id, secret, code, err := cliauth.NewRequest(time.Now(), "Local publisher")
	if err != nil {
		t.Fatal(err)
	}
	path := "/auth/cli/requests/" + id
	status, view := c.call("GET", path+"?clientName="+url.QueryEscape("Local publisher"), nil, nil)
	if status != 200 || view["state"] != "pending" || strings.Contains(string(mustJSON(t, view)), secret) {
		t.Fatal(status, view)
	}
	confirm := cliauth.Confirmation{Client: "Local publisher", Code: code, Approve: true}
	if status, _ := c.call("POST", path, confirm, map[string]string{"X-CSRF-Token": ""}); status != 403 {
		t.Fatal("CLI consent skipped CSRF", status)
	}
	if status, value := c.call("POST", path, confirm, nil); status != 200 || value["state"] != "approved" {
		t.Fatal(status, value)
	}
	proof := cliauth.Proof{Request: id, Client: "Local publisher", Verifier: secret}
	exchange := func(proof cliauth.Proof) cliauth.Exchange {
		t.Helper()
		request, err := http.NewRequestWithContext(t.Context(), "POST", c.origin+"/auth/cli/exchange", bytes.NewReader(mustJSON(t, proof)))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Origin", c.origin)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatal("anonymous CLI proof exchange", response.StatusCode)
		}
		if len(response.Cookies()) != 0 {
			t.Fatal("CLI exchange replaced a browser cookie")
		}
		var value cliauth.Exchange
		if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	result := exchange(proof)
	if result.State != "authorized" || result.Session == nil {
		t.Fatal(result)
	}
	child := result.Session
	request, err := http.NewRequestWithContext(t.Context(), "GET", c.origin+"/api/v1/projects", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "teststudio_session", Value: child.Cookie})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("CLI session did not use normal API authorization", response.StatusCode)
	}
	if repeated := exchange(proof); repeated.State != "consumed" || repeated.Session != nil {
		t.Fatal("CLI proof replayed", repeated)
	}
	if status, value := c.call("GET", "/auth/session", nil, nil); status != 200 || value["session"].(map[string]any)["csrfToken"] != c.csrf {
		t.Fatal("CLI sign-in changed the browser session")
	}
}
func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
