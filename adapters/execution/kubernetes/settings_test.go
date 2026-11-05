package kubernetes

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerSettingsUseMountedTrustAndRotatingControllerTokens(t *testing.T) {
	f := fresh(t, "go")
	directory := t.TempDir()
	config := f.provider.config
	config.TokenFile = filepath.Join(directory, "token")
	config.CAFile = filepath.Join(directory, "ca.crt")
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.server.Certificate().Raw})
	if err := os.WriteFile(config.TokenFile, []byte(f.token), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.CAFile, certificate, 0600); err != nil {
		t.Fatal(err)
	}
	settings := Settings{Worker: config, Registry: RegistrySettings{Origin: f.server.URL, RepositoryPrefix: "teststudio"}}
	data, _ := json.Marshal(settings)
	lookup := func(name string) (string, bool) { return string(data), name == "TESTSTUDIO_WORKERS" }
	loaded, err := LoadSettings("", "TESTSTUDIO_WORKERS", lookup)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := loaded.Open(f.store, func(string) (string, bool) { return "", false }, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	var response map[string]any
	if err := provider.request(t.Context(), http.MethodGet, "/api/v1/namespaces/studio-workers", nil, nil, &response); err != nil {
		t.Fatal("mounted certificate or token was not used", err)
	}
	f.mu.Lock()
	f.token = "rotated-controller-token"
	f.mu.Unlock()
	if err := os.WriteFile(config.TokenFile, []byte("rotated-controller-token"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := provider.request(t.Context(), http.MethodGet, "/api/v1/namespaces/studio-workers", nil, nil, &response); err != nil {
		t.Fatal("rotated mounted token was not read", err)
	}
	settings.Registry.BearerEnvironment = "TESTSTUDIO_DATABASE"
	data, _ = json.Marshal(settings)
	if _, err := LoadSettings("", "TESTSTUDIO_WORKERS", lookup); err == nil || strings.Contains(err.Error(), "rotated-controller-token") {
		t.Fatal("unsafe credential environment reference was admitted", err)
	}
	if _, err := settings.Open(f.store, func(string) (string, bool) { return "private-canary", true }, nil, nil); err == nil {
		t.Fatal("direct settings bypassed credential validation")
	}
	if _, err := LoadSettings("", "", func(string) (string, bool) { t.Fatal("disabled workers read environment"); return "", false }); err != nil {
		t.Fatal(err)
	}
}

func TestRenderedWorkerSettingsMatchTheProtectedPolicy(t *testing.T) {
	file := os.Getenv("TESTSTUDIO_KUBERNETES_SETTINGS")
	if file == "" {
		t.Skip("rendered worker configuration was not selected")
	}
	settings, err := LoadSettings(file, "", func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	policy := os.Getenv("TESTSTUDIO_KUBERNETES_POLICY")
	if policy == "" {
		policy = filepath.Join(filepath.Dir(file), "private-workers-policy.json")
	}
	data, err := os.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	var actual map[string]any
	if json.Unmarshal(data, &actual) != nil {
		t.Fatal("rendered network policy is invalid")
	}
	provider := &Provider{config: settings.Worker}
	expected, _ := json.Marshal(provider.policy())
	normalized, _ := json.Marshal(actual)
	if string(expected) != string(normalized) {
		t.Fatal("chart policy differs from provider admission")
	}
}
