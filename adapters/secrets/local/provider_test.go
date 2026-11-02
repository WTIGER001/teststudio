package local

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
)

func TestBindingsResolveOnlyTheirProjectAndPurposeWithoutExposingSecretValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webhook")
	value := "https://receiver.invalid/private-token"
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := New([]Binding{{ID: "alerts", Project: "one", Purpose: secrets.NotificationWebhook, File: path}, {ID: "alerts", Project: "two", Purpose: secrets.NotificationWebhook, Environment: "TESTSTUDIO_SECOND_WEBHOOK", Network: networkguard.Loopback}})
	if err != nil {
		t.Fatal(err)
	}
	p.lookup = func(name string) (string, bool) {
		return "http://127.0.0.1/other-private-token", name == "TESTSTUDIO_SECOND_WEBHOOK"
	}
	refs, err := p.References(t.Context(), "one", secrets.NotificationWebhook)
	if err != nil || len(refs) != 1 {
		t.Fatal("missing references", err)
	}
	encoded, _ := json.Marshal(refs)
	if strings.Contains(string(encoded), "token") || strings.Contains(string(encoded), path) || strings.Contains(string(encoded), "TESTSTUDIO_SECOND_WEBHOOK") {
		t.Fatal("reference listing exposed secret sources or values")
	}
	first, err := p.Resolve(t.Context(), "one", "alerts", secrets.NotificationWebhook)
	if err != nil || string(first.Bytes) != value || first.Network != networkguard.Public {
		t.Fatal("file binding did not resolve")
	}
	second, err := p.Resolve(t.Context(), "two", "alerts", secrets.NotificationWebhook)
	if err != nil || second.Network != networkguard.Loopback || string(second.Bytes) == value {
		t.Fatal("project bindings crossed scope")
	}
	encoded, _ = json.Marshal(first)
	if string(encoded) != "{}" {
		t.Fatal("resolved secret entered a JSON payload")
	}
	for _, scope := range []struct{ project, purpose string }{{"other", secrets.NotificationWebhook}, {"one", "source-access"}} {
		if _, err := p.Resolve(t.Context(), scope.project, "alerts", scope.purpose); !errors.Is(err, secrets.ErrUnavailable) {
			t.Fatal("unregistered scope resolved a secret")
		}
	}
}
func TestBindingsRejectAmbiguousSourcesAndBoundSecretReads(t *testing.T) {
	for _, binding := range []Binding{
		{ID: "alerts", Project: "one", Purpose: secrets.NotificationWebhook},
		{ID: "alerts", Project: "one", Purpose: secrets.NotificationWebhook, Environment: "BAD NAME"},
		{ID: "alerts", Project: "one", Purpose: secrets.NotificationWebhook, File: "relative"},
		{ID: "alerts", Project: "one", Purpose: secrets.NotificationWebhook, Environment: "VALUE", File: "/file"},
		{ID: "alerts", Project: "one", Purpose: secrets.NotificationWebhook, Environment: "VALUE", Network: "anywhere"},
	} {
		if _, err := New([]Binding{binding}); err == nil {
			t.Fatalf("invalid binding accepted: %+v", binding)
		}
	}
	p, err := New([]Binding{{ID: "alerts", Project: "one", Purpose: secrets.NotificationWebhook, Environment: "VALUE"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", strings.Repeat("x", 65537)} {
		p.lookup = func(string) (string, bool) { return value, true }
		if _, err := p.Resolve(t.Context(), "one", "alerts", secrets.NotificationWebhook); err == nil || strings.Contains(err.Error(), value) && value != "" {
			t.Fatal("unbounded or exposed secret")
		}
	}
}
