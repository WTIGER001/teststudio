package httpapi_test

import (
	"encoding/json"
	"strings"
	"testing"

	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/secrets"
)

func TestAPICredentialReferencesKeepProjectPurposeOriginAndValuesPrivate(t *testing.T) {
	c, _ := preparedRun(t)
	provider, err := localsecrets.New([]localsecrets.Binding{
		{ID: "reader", Name: "Approved API reader", Project: "example", Purpose: secrets.APICredential, Environment: "UNRESOLVED_API_PRIVATE_VALUE", Origins: []string{"https://approved.test"}},
		{ID: "foreign", Project: "other", Purpose: secrets.APICredential, Environment: "UNRESOLVED_API_PRIVATE_VALUE", Origins: []string{"https://foreign.test"}},
		{ID: "source", Project: "example", Purpose: "source-access", Environment: "UNRESOLVED_API_PRIVATE_VALUE"},
	})
	if err != nil {
		t.Fatal(err)
	}
	c.api.Projects.Secrets = provider
	status, refs := c.call("GET", "/api/v1/projects/example/api-credential-references", nil, nil)
	encoded, _ := json.Marshal(refs)
	if status != 200 || len(refs["items"].([]any)) != 1 || strings.Contains(string(encoded), "UNRESOLVED") || strings.Contains(string(encoded), "foreign") || strings.Contains(string(encoded), "source") {
		t.Fatal("credential reference listing crosses scope or exposes a source", status)
	}
	if _, err := c.api.Projects.CredentialReferences(t.Context(), "unknown-user", "example"); err == nil {
		t.Fatal("unprivileged principal reads credential references")
	}
	environment := projects.Environment{ID: "authenticated", Name: "Authenticated API", Application: "app", Stage: "development", Endpoints: map[string]string{"api": "https://approved.test"}, Credentials: map[string]string{"api-login": "reader"}}
	if status, data := c.call("PUT", "/api/v1/projects/example/environments/authenticated", environment, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal("approved reference rejected", status, data)
	}
	for _, reference := range []string{"foreign", "source", "unregistered"} {
		environment.ID = "invalid-" + reference
		environment.Credentials["api-login"] = reference
		if status, _ := c.call("PUT", "/api/v1/projects/example/environments/"+environment.ID, environment, map[string]string{"If-Match": "0"}); status != 422 {
			t.Fatal("foreign or wrong-purpose reference admitted", reference, status)
		}
	}
	environment.ID = "wrong-origin"
	environment.Credentials["api-login"] = "reader"
	environment.Endpoints["api"] = "https://elsewhere.test"
	if status, _ := c.call("PUT", "/api/v1/projects/example/environments/wrong-origin", environment, map[string]string{"If-Match": "0"}); status != 422 {
		t.Fatal("unapproved origin accepts credential binding")
	}
}
