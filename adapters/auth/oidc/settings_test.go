package oidc_test

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/adapters/auth/oidc"
)

func TestInitialProviderConfigurationValidatesTheCompleteList(t *testing.T) {
	one := `{"id":"primary","name":"Primary","issuer":"https://identity.example.test","clientId":"studio","secretRef":"env:TESTSTUDIO_OIDC_CLIENT_ONE"}`
	two := `{"id":"secondary","name":"Secondary","issuer":"https://other.example.test","clientId":"studio","secretRef":"env:TESTSTUDIO_OIDC_CLIENT_TWO"}`
	for _, fixture := range []struct {
		document string
		valid    bool
		count    int
	}{
		{"[" + one + "," + two + "]", true, 2},
		{"[" + one + "," + one + "]", false, 0},
		{"[" + one + ",{}]", false, 0},
		{`[{"id":"private","name":"Private","issuer":"https://identity.example.test","clientId":"studio","clientSecret":"must-not-echo"}]`, false, 0},
		{`null`, false, 0}, {`[]`, false, 0}, {`{}`, false, 0},
	} {
		list, err := oidc.InitialProviders("", "PROVIDERS", func(string) (string, bool) { return fixture.document, true })
		if (err == nil) != fixture.valid || len(list) != fixture.count || (err != nil && strings.Contains(err.Error(), "must-not-echo")) {
			t.Fatal("incorrect initial provider outcome", err)
		}
	}
}
