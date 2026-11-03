package github_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type webhookSecret struct{ value string }

func (s webhookSecret) References(context.Context, string, string) ([]secrets.Reference, error) {
	return []secrets.Reference{{ID: "hook", Name: "Webhook"}}, nil
}
func (s webhookSecret) Resolve(_ context.Context, project, id, purpose string) (secrets.Value, error) {
	if project != "alpha" || id != "hook" || purpose != sourceprovider.WebhookPurpose {
		return secrets.Value{}, secrets.ErrUnavailable
	}
	return secrets.Value{Bytes: []byte(s.value)}, nil
}
func TestGitHubWebhookAuthenticatesExactBytesAndScopedSecretBeforeParsing(t *testing.T) {
	secret := strings.Repeat("secret", 8)
	client := &github.Client{Secrets: webhookSecret{value: secret}}
	body, _ := json.Marshal(map[string]any{"ref": "refs/heads/main", "after": strings.Repeat("a", 40), "repository": map[string]any{"id": 42, "full_name": "owner/suite"}})
	sign := func(data []byte) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(data)
		return "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}
	request := sourceprovider.WebhookRequest{Delivery: "delivery-one", Event: "push", Signature: sign(body), Body: body}
	access := sourceprovider.Access{Project: "alpha", Repository: "owner/suite", RepositoryID: "42"}
	event, err := client.VerifyWebhook(t.Context(), access, "hook", request)
	if err != nil || event.Reference != "heads/main" || event.RepositoryID != "42" || event.Commit != strings.Repeat("a", 40) {
		t.Fatal(event, err)
	}
	for _, change := range []func(*sourceprovider.WebhookRequest){
		func(r *sourceprovider.WebhookRequest) { r.Signature = "" },
		func(r *sourceprovider.WebhookRequest) { r.Signature = "sha1=" + strings.Repeat("0", 40) },
		func(r *sourceprovider.WebhookRequest) { r.Body = append(append([]byte{}, r.Body...), ' ') },
		func(r *sourceprovider.WebhookRequest) { r.Body = []byte("invalid"); r.Signature = sign(r.Body) },
		func(r *sourceprovider.WebhookRequest) { r.Delivery = "bad/delivery" },
	} {
		altered := request
		change(&altered)
		if _, err := client.VerifyWebhook(t.Context(), access, "hook", altered); !errors.Is(err, sourceprovider.ErrWebhookRejected) {
			t.Fatal("unsafe delivery accepted", err)
		}
	}
	access.Project = "other"
	if _, err := client.VerifyWebhook(t.Context(), access, "hook", request); !errors.Is(err, sourceprovider.ErrWebhookRejected) {
		t.Fatal("cross-project secret accepted", err)
	}
}
