package httpapi_test

import (
	"encoding/json"
	"strings"
	"testing"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestSignInOptionsSeparatePasswordsVisibleSSOAndBypass(t *testing.T) {
	for _, mode := range []struct {
		name                   string
		development, passwords bool
	}{
		{"sso-only", false, false}, {"mixed", false, true}, {"development", true, false}, {"both-local", true, true},
	} {
		t.Run(mode.name, func(t *testing.T) {
			var c *client
			if mode.passwords {
				c = passwordFixture(t, mode.development)
			} else {
				c = fixture(t, func(api *httpapi.Server) {
					if !mode.development {
						api.LocalUser = ""
					}
				})
			}
			providers := []authoidc.Provider{
				{ID: "organization", Name: "Organization sign-in", LoginBypassEnabled: true},
				{ID: "bootstrap", Name: "Private recovery provider", HiddenFromSignIn: true, LoginBypassEnabled: true},
				{ID: "disabled", Name: "Disabled provider", Disabled: true, LoginBypassEnabled: true},
			}
			for _, provider := range providers {
				provider.Issuer = "https://identity.example.test"
				provider.ClientID = "private-client-reference"
				provider.SecretRef = "env:PRIVATE_SECRET"
				change, err := storage.Put("providers", provider.ID, "", 0, provider)
				if err != nil {
					t.Fatal(err)
				}
				if err = c.api.Store.Commit(t.Context(), change); err != nil {
					t.Fatal(err)
				}
			}
			value := accessCall(t, c, "GET", "/auth/providers", nil, "", 200)
			list := value["providers"].([]any)
			if len(list) != 1 || list[0].(map[string]any)["id"] != "organization" {
				t.Fatal(value)
			}
			expected := ""
			if !mode.passwords && !mode.development {
				expected = "organization"
			}
			if value["autoRedirectProvider"] != expected || value["localAccountsEnabled"] != mode.passwords || value["localEnabled"] != mode.development {
				t.Fatal(value)
			}
			encoded, _ := json.Marshal(value)
			for _, private := range []string{"PRIVATE_SECRET", "private-client-reference", "identity.example.test", "Private recovery provider"} {
				if strings.Contains(string(encoded), private) {
					t.Fatal("private provider configuration reached public choices")
				}
			}
			extra, err := storage.Put("providers", "another", "", 0, authoidc.Provider{ID: "another", Name: "Another SSO", LoginBypassEnabled: true})
			if err != nil {
				t.Fatal(err)
			}
			if err := c.api.Store.Commit(t.Context(), extra); err != nil {
				t.Fatal(err)
			}
			value = accessCall(t, c, "GET", "/auth/providers", nil, "", 200)
			if value["autoRedirectProvider"] != "" {
				t.Fatal("ambiguous bypass chose a provider", value)
			}
		})
	}
}

func TestProviderSignInOptionsRequireNormalAuthorityAndRetainRevisions(t *testing.T) {
	c := fixture(t)
	c.signIn()
	input := providerInput("organization", false)
	input.LoginBypassEnabled = true
	value := accessCall(t, c, "POST", "/api/v1/identity-providers", input, "", 200)
	if value["data"].(map[string]any)["loginBypassEnabled"] != true {
		t.Fatal(value)
	}
	input.HiddenFromSignIn = true
	accessCall(t, c, "PUT", "/api/v1/identity-providers/organization", input, "1", 422)
	input.LoginBypassEnabled = false
	value = accessCall(t, c, "PUT", "/api/v1/identity-providers/organization", input, "1", 200)
	if value["data"].(map[string]any)["hiddenFromSignIn"] != true {
		t.Fatal(value)
	}
	value = accessCall(t, c, "GET", "/auth/providers", nil, "", 200)
	if len(value["providers"].([]any)) != 0 {
		t.Fatal(value)
	}
	accessCall(t, c, "PUT", "/api/v1/identity-providers/organization", input, "1", 409)
}
