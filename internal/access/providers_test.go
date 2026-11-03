package access_test

import (
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/storage"
)

func providerDefinition(id, issuer string, disabled bool) access.ProviderInput {
	return access.ProviderInput{ID: id, Name: id, Issuer: issuer, ClientID: "teststudio", Disabled: &disabled, AutoProvision: true, Scopes: []string{}, GroupMappings: map[string]string{}, Reason: "Verify managed provider lifecycle"}
}

func TestProviderChangesPreserveTheLastUsableAdministratorAndRollBackAudit(t *testing.T) {
	s := fixture(t)
	input := providerDefinition("example", "https://identity.example.test", false)
	if _, err := s.SaveProvider(t.Context(), "admin", input, 0); err != nil {
		t.Fatal(err)
	}
	backup := providerUser(t, s, "provider-administrator")
	if err := s.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: backup.ID}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	s.LocalUser = ""
	for _, mode := range []string{"disable", "issuer"} {
		value := input
		if mode == "disable" {
			disabled := true
			value.Disabled = &disabled
		} else {
			value.Issuer = "https://replacement.example.test"
		}
		before, err := s.Auth.Directory.Revision(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		audit, err := s.Store.All(t.Context(), "audit", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.SaveProvider(t.Context(), "admin", value, 1); err == nil || !strings.Contains(err.Error(), "last usable") {
			t.Fatal("provider change removed the only usable login", mode, err)
		}
		after, err := s.Auth.Directory.Revision(t.Context())
		if err != nil || after.Revision != before.Revision {
			t.Fatal("rejected provider changed directory", err)
		}
		current, err := s.Store.All(t.Context(), "audit", "")
		if err != nil || len(current) != len(audit) {
			t.Fatal("rejected provider retained audit writes", err)
		}
		row, err := s.Store.Get(t.Context(), "providers", "example")
		if err != nil || row.Revision != 1 {
			t.Fatal("rejected provider advanced revision", err)
		}
	}
	if _, err := s.SaveProvider(t.Context(), "admin", providerDefinition("alternate", input.Issuer, false), 0); err != nil {
		t.Fatal(err)
	}
	disabled := true
	input.Disabled = &disabled
	if _, err := s.SaveProvider(t.Context(), "admin", input, 1); err != nil {
		t.Fatal("alternate issuer login was not retained", err)
	}
	row, err := s.Store.Get(t.Context(), "providers", "example")
	if err != nil {
		t.Fatal(err)
	}
	value, err := storage.Decode[struct {
		Disabled bool `json:"disabled"`
	}](row)
	if err != nil || !value.Disabled || row.Revision != 2 {
		t.Fatal("provider lifecycle was not saved", err)
	}
}
