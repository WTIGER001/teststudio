package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func preparedFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	poison := filepath.Join(root, "initialization-ran")
	source := `package arithmetic
import("os";"testing")
func init(){if err:=os.WriteFile(` + "`" + poison + "`" + `,[]byte("ran"),0600);err!=nil{panic(err)}}
func TestAddition(t *testing.T){if 6+8!=14{t.Fatal("arithmetic")}}
func TestRows(t *testing.T){t.Run("first",func(t *testing.T){})}
`
	manifest := suitespec.Manifest{Schema: 1, Suite: "fresh-go", Runner: "go", Type: "unit", ModuleRoot: "service", Catalog: "testing/catalog.json", BuildProfile: "go-module", Tests: suitespec.Tests{Packages: []string{"./unit"}}}
	origin := suitespec.Origin{SnapshotDigest: "sha256:" + strings.Repeat("a", 64)}
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{}}
	for _, item := range []struct{ id, selector string }{{"SUM", "./unit::TestAddition"}, {"ROW", "./unit::TestRows/first"}} {
		document.Tests = append(document.Tests, catalog.Test{ID: item.id, Title: item.id, Category: "Unit", Runner: "go", Type: "unit", State: "enabled", Safety: "safe", Rationale: "Reads only in-memory values", TimeoutSeconds: 5, Source: "service/unit/arithmetic_test.go", Selector: item.selector})
	}
	files := map[string]any{suitespec.ManifestFile: manifest, suitespec.OriginFile: origin, manifest.Catalog: document}
	for name, value := range files {
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
	if err := os.MkdirAll(filepath.Join(root, "service/unit"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "service/go.mod"), []byte("module teststudio.local/discovery\n\ngo 1.24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "service/unit/arithmetic_test.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPreparedGoDiscoveryCompilesWithoutInitializationOrTestExecution(t *testing.T) {
	root := preparedFixture(t)
	discovery, err := Discover(t.Context(), root)
	if err != nil {
		t.Fatalf("discovery: %v logs %s", err, discovery.Log)
	}
	if len(discovery.Tests) != 2 || discovery.Tests[0].ParentOnly || !discovery.Tests[1].ParentOnly || discovery.Tests[0].Line != 4 {
		t.Fatalf("source facts: %+v", discovery.Tests)
	}
	if err := suitespec.ValidateDiscovery(discovery, "sha256:"+strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "initialization-ran")); !os.IsNotExist(err) {
		t.Fatal("discovery executed package initialization")
	}
	var input Input
	input.Test.ID, input.Test.Selector, input.TimeoutMs = "SUM", "./unit::TestAddition", 5000
	output := t.TempDir()
	report := Execute(t.Context(), input, root, output)
	if report.Result.State != "passed" {
		t.Fatalf("prepared module execution: %+v", report.Result)
	}
	if _, err := os.Stat(filepath.Join(root, "initialization-ran")); err != nil {
		t.Fatal("normal execution did not use prepared module context")
	}
}

func TestPreparedGoDiscoveryRejectsMissingDeclarationsAndBuildFailures(t *testing.T) {
	for _, kind := range []string{"missing", "build", "helper", "test-main", "non-running-example"} {
		t.Run(kind, func(t *testing.T) {
			root := preparedFixture(t)
			if kind == "helper" || kind == "test-main" || kind == "non-running-example" {
				name, body := "Testhelper", "func Testhelper(t *testing.T){}"
				if kind == "test-main" {
					name, body = "TestMain", "func TestMain(m *testing.M){}"
				}
				if kind == "non-running-example" {
					name, body = "ExampleAddition", "func Addition(){}\nfunc ExampleAddition(){}"
				}
				source := filepath.Join(root, "service/unit/arithmetic_test.go")
				data, _ := os.ReadFile(source)
				if err := os.WriteFile(source, append(data, []byte(body+"\n")...), 0600); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(root, "testing/catalog.json")
				data, _ = os.ReadFile(file)
				if err := os.WriteFile(file, []byte(strings.ReplaceAll(string(data), "TestAddition", name)), 0600); err != nil {
					t.Fatal(err)
				}
			} else if kind == "missing" {
				file := filepath.Join(root, "testing/catalog.json")
				data, _ := os.ReadFile(file)
				if err := os.WriteFile(file, []byte(strings.ReplaceAll(string(data), "TestAddition", "TestAbsent")), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(filepath.Join(root, "service/unit/broken.go"), []byte("package arithmetic\nvar Broken = MissingDeclaration\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Discover(t.Context(), root); err == nil {
				t.Fatal("invalid artifact discovery succeeded")
			}
		})
	}
}
