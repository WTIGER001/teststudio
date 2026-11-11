package github_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func TestGitHubTokenReadsPrivateSourceAndReplacesWithoutAppExchange(t *testing.T) {
	f := fixture(t)
	expiry := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	replace := func() {
		data, err := json.Marshal(map[string]any{"token": f.apiToken, "repositories": []string{"owner/suite"}, "expiresAt": expiry})
		if err != nil {
			t.Fatal(err)
		}
		f.client.Secrets = credentials{data: data}
	}
	replace()
	access := sourceprovider.Access{Project: "alpha", Repository: "owner/suite", Credential: "app"}
	revision, err := f.client.Resolve(t.Context(), access, "heads/main")
	if err != nil || !revision.Private || revision.Commit != f.commit {
		t.Fatal("private token authentication", err)
	}
	archive, err := f.client.Archive(t.Context(), access, revision)
	if err != nil || len(archive.Content) == 0 || f.downloads != 1 || f.tokenRequests != 0 {
		t.Fatal("token archive retrieval or redirect isolation", err)
	}
	info, err := f.client.CredentialInfo(t.Context(), access)
	if err != nil || info.ExpiresAt == nil || !info.ExpiresAt.Equal(expiry) || info.CanRotate || info.Fingerprint != "" || info.Renewal != "operator token replacement" {
		t.Fatal("token lifecycle metadata", err)
	}
	encoded, _ := json.Marshal(info)
	if strings.Contains(string(encoded), f.apiToken) {
		t.Fatal("token leaked into credential status")
	}
	f.apiToken = "fixture-replacement-token"
	replace()
	if _, err := f.client.Resolve(t.Context(), access, "heads/main"); err != nil || f.tokenRequests != 0 {
		t.Fatal("replacement token was not read", err)
	}
}

func TestGitHubTokenRejectsScopeExpiryAndMixedCredentialsBeforeNetwork(t *testing.T) {
	f := fixture(t)
	for _, kind := range []string{"unapproved-repository", "write-disabled", "expired", "mixed-app", "newline", "too-long", "empty"} {
		t.Run(kind, func(t *testing.T) {
			input := map[string]any{"token": "fixture-private-token", "repositories": []string{"owner/suite"}}
			access := sourceprovider.Access{Project: "alpha", Repository: "owner/suite", Credential: "app"}
			switch kind {
			case "unapproved-repository":
				input["repositories"] = []string{"other/repository"}
			case "write-disabled":
				access.Write = true
			case "expired":
				input["expiresAt"] = time.Now().UTC().Add(-time.Hour)
			case "mixed-app":
				input["clientId"] = "fixture-client"
			case "newline":
				input["token"] = "fixture-private-token\nInjected: header"
			case "too-long":
				input["token"] = strings.Repeat("x", 4097)
			case "empty":
				input["token"] = ""
			}
			data, _ := json.Marshal(input)
			f.client.Secrets = credentials{data: data}
			before := f.apiRequests
			_, err := f.client.Resolve(t.Context(), access, "heads/main")
			if err == nil || strings.Contains(err.Error(), "fixture-private-token") || f.apiRequests != before {
				t.Fatal("invalid token reached the network or leaked", err)
			}
		})
	}
}
