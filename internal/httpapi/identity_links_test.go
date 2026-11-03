package httpapi_test

import (
	"fmt"
	"strings"
	"testing"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/accountlinks"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestIdentityLinkAPIRequiresSessionCSRFAndCurrentRevision(t *testing.T) {
	c := fixture(t)
	provider, _ := storage.Put("providers", "new-provider", "", 0, authoidc.Provider{ID: "new-provider", Name: "New provider", Issuer: "https://identity.example", ClientID: "private-client", SecretRef: "env:PRIVATE_SECRET"})
	if err := c.api.Store.Commit(t.Context(), provider); err != nil {
		t.Fatal(err)
	}
	accessCall(t, c, "GET", "/api/v1/me/identity-linking", nil, "", 401)
	c.signIn()
	options := accessCall(t, c, "GET", "/api/v1/me/identity-linking", nil, "", 200)
	if options["canStart"] != true || len(options["providers"].([]any)) != 1 {
		t.Fatal("missing choices", options)
	}
	for _, secret := range []string{"private-client", "PRIVATE_SECRET", "secretRef", "clientId"} {
		if strings.Contains(fmt.Sprint(options), secret) {
			t.Fatal("provider credential configuration leaked", secret)
		}
	}
	input := accountlinks.StartInput{ProviderID: "new-provider", Reason: "Attach a provider", ConfirmLocal: true}
	for _, headers := range []map[string]string{{"Origin": "https://other.example"}, {"X-CSRF-Token": "wrong"}} {
		if status, _ := c.call("POST", "/api/v1/me/identity-links", input, headers); status != 403 {
			t.Fatal("cross-origin or missing CSRF accepted", status)
		}
	}
	op := accessCall(t, c, "POST", "/api/v1/me/identity-links", input, "", 200)
	path := "/api/v1/me/identity-links/" + op["id"].(string)
	accessCall(t, c, "POST", path+"/confirm", nil, "0", 409)
	accessCall(t, c, "POST", path+"/confirm", nil, "1", 422)
	accessCall(t, c, "POST", path+"/cancel", nil, "", 422)
	accessCall(t, c, "POST", path+"/cancel", nil, "0", 409)
	accessCall(t, c, "POST", path+"/cancel", nil, "2", 409)
	other := passwordClient(t, c)
	other.signIn()
	accessCall(t, other, "GET", path, nil, "", 404)
	accessCall(t, other, "POST", path+"/cancel", nil, "1", 404)
	cancelled := accessCall(t, c, "POST", path+"/cancel", nil, "1", 200)
	if cancelled["state"] != "cancelled" || cancelled["revision"] != float64(2) {
		t.Fatal("incorrect cancellation", cancelled)
	}
	accessCall(t, c, "POST", path+"/cancel", nil, "1", 409)
	accessCall(t, c, "POST", path+"/confirm", nil, "2", 422)
}
