package suitespec_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func manifest() suitespec.Manifest {
	return suitespec.Manifest{Schema: 1, Suite: "unit", Type: "unit", Runner: "go", ModuleRoot: "service", Catalog: "testing/catalog.json", Tests: suitespec.Tests{Packages: []string{"./unit"}}, BuildProfile: "go-module"}
}

func TestAPIAuthenticationManifestAndCatalogShareTheDeclaredBinding(t *testing.T) {
	m := suitespec.Manifest{Schema: 1, Suite: "requests", Type: "api", Runner: "playwright", ModuleRoot: ".", Catalog: "catalog.json", Tests: suitespec.Tests{Config: "playwright.config.ts"}, BuildProfile: "playwright-environment", Browser: &browserpolicy.Spec{Endpoint: "api", Requests: &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "login", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/", ReadOnly: true}}, MaxRequests: 2, MaxResponseBytes: 1024}}}
	if suitespec.Validate(m) == nil {
		t.Fatal("authentication without manifest binding accepted")
	}
	m.RequiredBindings = []string{"login"}
	d := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: "API-1", Runner: "playwright", Type: "api", Source: "checks/api.spec.ts", Selector: "@API-1", RequiredBindings: []string{"login"}}}}
	if err := suitespec.ValidateCatalog(m, d); err != nil {
		t.Fatal("declared API credential rejected", err)
	}
	d.Tests[0].RequiredBindings = []string{"foreign"}
	if suitespec.ValidateCatalog(m, d) == nil {
		t.Fatal("unreviewed test binding accepted")
	}
	d.Tests[0].RequiredBindings = nil
	if err := suitespec.ValidateCatalog(m, d); err != nil {
		t.Fatal("missing-credential scenario rejected", err)
	}
}
func descriptor() suitespec.Discovery {
	spec := manifest()
	return suitespec.Discovery{Schema: suitespec.DiscoverySchema, Manifest: spec, Origin: suitespec.Origin{SnapshotDigest: "sha256:" + strings.Repeat("a", 64)}, Catalog: catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{{ID: "UNIT-1", Title: "Addition", Description: "Independent arithmetic check", Category: "Unit", Runner: "go", Type: "unit", State: "enabled", Safety: "safe", Rationale: "Computes in memory", TimeoutSeconds: 5, Source: "service/unit/arithmetic_test.go", Selector: "./unit::TestAddition"}}}, Tests: []suitespec.DiscoveredTest{{ID: "UNIT-1", Selector: "./unit::TestAddition", File: "service/unit/arithmetic_test.go", Line: 7}}}
}
func TestSuiteManifestOriginAndDiscoveredCatalogContracts(t *testing.T) {
	image := "sha256:" + strings.Repeat("b", 64)
	if err := suitespec.ValidateDiscovery(descriptor(), image); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name   string
		mutate func(*suitespec.Discovery)
	}{
		{"schema", func(d *suitespec.Discovery) { d.Manifest.Schema = 2 }},
		{"root escape", func(d *suitespec.Discovery) { d.Manifest.ModuleRoot = "../service" }},
		{"catalog escape", func(d *suitespec.Discovery) { d.Manifest.Catalog = "/etc/catalog.json" }},
		{"catalog URL", func(d *suitespec.Discovery) { d.Manifest.Catalog = "https://example.invalid/catalog.json" }},
		{"package traversal", func(d *suitespec.Discovery) { d.Manifest.Tests.Packages = []string{"../unit"} }},
		{"recursive packages", func(d *suitespec.Discovery) { d.Manifest.Tests.Packages = []string{"./..."} }},
		{"runner type", func(d *suitespec.Discovery) { d.Manifest.Type = "browser" }},
		{"project injection", func(d *suitespec.Discovery) { d.Manifest.Tests.Project = "--config=other" }},
		{"duplicate packages", func(d *suitespec.Discovery) { d.Manifest.Tests.Packages = []string{"./unit", "./unit"} }},
		{"missing origin", func(d *suitespec.Discovery) { d.Origin = suitespec.Origin{} }},
		{"credentials", func(d *suitespec.Discovery) {
			d.Origin = suitespec.Origin{Repository: "https://user:password@git.example/team/repo", Reference: "main", Commit: strings.Repeat("a", 40)}
		}},
		{"tokens in query", func(d *suitespec.Discovery) {
			d.Origin = suitespec.Origin{Repository: "https://git.example/team/repo?token=private", Reference: "main", Commit: strings.Repeat("a", 40)}
		}},
		{"unresolved commit", func(d *suitespec.Discovery) {
			d.Origin = suitespec.Origin{Repository: "https://git.example/team/repo", Reference: "main", Commit: "main"}
		}},
		{"binding absent", func(d *suitespec.Discovery) { d.Catalog.Tests[0].RequiredBindings = []string{"database"} }},
		{"application absent", func(d *suitespec.Discovery) { d.Catalog.Tests[0].Application = "foreign" }},
		{"source mismatch", func(d *suitespec.Discovery) { d.Tests[0].File = "service/unit/other_test.go" }},
		{"selector mismatch", func(d *suitespec.Discovery) { d.Tests[0].Selector = "./unit::TestAnother" }},
		{"duplicate discovery", func(d *suitespec.Discovery) { d.Tests = append(d.Tests, d.Tests[0]) }},
		{"missing discovery", func(d *suitespec.Discovery) { d.Tests = nil }},
		{"child claim", func(d *suitespec.Discovery) {
			d.Catalog.Tests[0].Selector = "./unit::TestAddition/child"
			d.Tests[0].Selector = d.Catalog.Tests[0].Selector
		}},
	} {
		t.Run(item.name, func(t *testing.T) {
			d := descriptor()
			item.mutate(&d)
			if err := suitespec.ValidateDiscovery(d, image); err == nil {
				t.Fatal("invalid discovery accepted")
			}
		})
	}
	if err := suitespec.ValidateDiscovery(descriptor(), "mutable:latest"); err == nil {
		t.Fatal("mutable image accepted")
	}
	origin := suitespec.Origin{Repository: "https://github.com/wtiger001/fresh-tests", Reference: "refs/heads/main", Commit: strings.Repeat("c", 40)}
	if err := suitespec.ValidateOrigin(origin); err != nil {
		t.Fatal(err)
	}
}

func TestSuiteDeclarationReadsRejectEscapesUnknownFieldsAndSpecialFiles(t *testing.T) {
	root := t.TempDir()
	file := func(name string, value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		filename := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	d := descriptor()
	file(suitespec.ManifestFile, d.Manifest)
	file(suitespec.OriginFile, d.Origin)
	file(d.Manifest.Catalog, d.Catalog)
	if _, err := suitespec.Load(root); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.json")); err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := suitespec.Read(root, "escape.json", &value); err == nil {
		t.Fatal("escaping link read")
	}
	if err := suitespec.Read(root, "testing", &value); err == nil {
		t.Fatal("directory declaration read")
	}
	if err := definitions.Decode("json", `{"schemaVersion":1,"suiteId":"unit","unknown":true}`, &suitespec.Manifest{}); err == nil {
		t.Fatal("unknown manifest field accepted")
	}
	if err := definitions.Decode("yaml", "schemaVersion: 1\nschemaVersion: 1", &suitespec.Manifest{}); err == nil {
		t.Fatal("duplicate YAML field accepted")
	}
}
