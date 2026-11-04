package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func TestGoAPIInputMatchesPreparedPolicyAndAuthentication(t *testing.T) {
	root := t.TempDir()
	policy := &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "login", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/health", ReadOnly: true}}, MaxRequests: 3, MaxResponseBytes: 1024}
	manifest := suitespec.Manifest{Runner: "go", Type: "api", BuildProfile: "go-environment", Browser: &browserpolicy.Spec{Endpoint: "api", Requests: policy}, Catalog: "catalog.json"}
	document := catalog.Document{Tests: []catalog.Test{{ID: "GO-API-1", Selector: "./checks::TestHealth", RequiredBindings: []string{"login"}}}}
	for name, value := range map[string]any{suitespec.ManifestFile: manifest, "catalog.json": document} {
		data, _ := json.Marshal(value)
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	input := Input{Browser: &browserpolicy.Target{Origin: "https://approved.test", Requests: policy, Authenticated: true}}
	input.Test.ID, input.Test.Selector, input.Test.Type, input.Test.RequiredBindings = "GO-API-1", "./checks::TestHealth", "api", []string{"login"}
	if err := validateEnvironment(root, input); err != nil {
		t.Fatal("reviewed Go API input rejected", err)
	}
	for _, mutate := range []func(*Input){func(i *Input) { i.Browser = nil }, func(i *Input) { i.Browser.Authenticated = false }, func(i *Input) { i.Test.RequiredBindings = nil }, func(i *Input) { i.Test.Type = "unit" }, func(i *Input) { copy := *policy; copy.MaxRequests = 4; i.Browser.Requests = &copy }} {
		candidate := input
		copy := *input.Browser
		candidate.Browser = &copy
		mutate(&candidate)
		if validateEnvironment(root, candidate) == nil {
			t.Fatal("mismatched Go API input accepted")
		}
	}
}
