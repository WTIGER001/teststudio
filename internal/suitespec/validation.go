package suitespec

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/gosuite"
)

var commitID = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)
var snapshotDigest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var playwrightSelector = regexp.MustCompile(`^@[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func RelativePath(value string, root bool) bool {
	return value != "" && len(value) <= 512 && !strings.ContainsAny(value, "\\:\x00") && !strings.ContainsFunc(value, unicode.IsControl) && !path.IsAbs(value) && path.Clean(value) == value && value != ".." && !strings.HasPrefix(value, "../") && (root || value != ".")
}

func Validate(manifest Manifest) error {
	if err := browserpolicy.Validate(manifest.Browser); err != nil {
		return err
	}
	if manifest.Browser != nil && manifest.Browser.Requests != nil && manifest.Type != "api" {
		return errors.New("request-restricted suites require API type")
	}
	environmentProfile := manifest.BuildProfile == "playwright-environment" || manifest.BuildProfile == "go-environment"
	if (manifest.Browser != nil) != environmentProfile || manifest.Browser != nil && (manifest.RequestPolicy != "" || manifest.Runner == "playwright" && manifest.BuildProfile != "playwright-environment" || manifest.Runner == "go" && (manifest.BuildProfile != "go-environment" || manifest.Type != "api" || manifest.Browser.Requests == nil) || manifest.Runner != "playwright" && manifest.Runner != "go") {
		return errors.New("environment suites require their runner's environment profile, a registered endpoint, and reviewed request rules for Go API clients")
	}
	if manifest.Browser != nil {
		if err := apipolicy.ValidateBindings(manifest.Browser.Requests, manifest.RequiredBindings); err != nil {
			return err
		}
		if manifest.Browser.Requests != nil && manifest.Browser.Requests.Authentication != nil && len(manifest.RequiredBindings) != 1 {
			return errors.New("API authentication must declare its manifest credential binding")
		}
	}
	if manifest.Schema != SchemaVersion {
		return errors.New("unsupported suite manifest schema")
	}
	if !domain.ValidID(manifest.Suite) || !domain.ValidID(manifest.BuildProfile) {
		return errors.New("valid suite and build profile identifiers are required")
	}
	if !RelativePath(manifest.ModuleRoot, true) || !RelativePath(manifest.Catalog, false) || (manifest.RequestPolicy != "" && !RelativePath(manifest.RequestPolicy, false)) {
		return errors.New("suite paths must stay within the source context")
	}
	if !slices.Contains([]string{".json", ".yaml", ".yml"}, path.Ext(manifest.Catalog)) {
		return errors.New("suite catalog requires JSON or YAML")
	}
	seen := map[string]bool{}
	for _, id := range append(slices.Clone(manifest.Applications), manifest.RequiredBindings...) {
		if !domain.ValidID(id) {
			return errors.New("manifest references require valid permanent identifiers")
		}
	}
	for _, id := range manifest.Applications {
		if seen[id] {
			return errors.New("duplicate application identifier")
		}
		seen[id] = true
	}
	seen = map[string]bool{}
	for _, id := range manifest.RequiredBindings {
		if seen[id] {
			return errors.New("duplicate required binding")
		}
		seen[id] = true
	}
	switch manifest.Runner {
	case "go":
		if !slices.Contains([]string{"unit", "integration", "api"}, manifest.Type) || (manifest.Tests.Project != "" || manifest.Tests.Config != "") || len(manifest.Tests.Packages) < 1 || len(manifest.Tests.Packages) > 100 {
			return errors.New("Go suites require a compatible type and 1 through 100 explicit packages")
		}
		seen = map[string]bool{}
		for _, pkg := range manifest.Tests.Packages {
			if _, err := gosuite.Parse(pkg + "::TestManifest"); err != nil {
				return err
			}
			if seen[pkg] {
				return errors.New("duplicate Go package")
			}
			seen[pkg] = true
		}
	case "playwright":
		if manifest.Tests.Config != "" && !RelativePath(manifest.Tests.Config, false) {
			return errors.New("Playwright configuration must stay within its module root")
		}
		if !slices.Contains([]string{"browser", "api", "accessibility", "visual", "performance"}, manifest.Type) || len(manifest.Tests.Packages) != 0 || (len(manifest.Tests.Project) > 200 || strings.ContainsAny(manifest.Tests.Project, "[]\r\n›>\x00")) {
			return errors.New("Playwright suites require a compatible type and at most one exact project")
		}
	default:
		return errors.New("prepared suites require a supported coded runner")
	}
	return nil
}

func ValidateOrigin(origin Origin) error {
	if origin.SnapshotDigest != "" && !snapshotDigest.MatchString(origin.SnapshotDigest) {
		return errors.New("source snapshot requires a SHA-256 digest")
	}
	if origin.Repository == "" {
		if origin.Commit != "" || origin.Reference != "" || origin.SnapshotDigest == "" {
			return errors.New("local source provenance requires a snapshot digest without Git fields")
		}
		return nil
	}
	repository, err := url.Parse(origin.Repository)
	if len(origin.Repository) > 2048 || err != nil || repository.Scheme != "https" || repository.Hostname() == "" || repository.User != nil || repository.RawQuery != "" || repository.ForceQuery || repository.Fragment != "" || strings.Contains(origin.Repository, "#") || repository.RawPath != "" || repository.Path == "" || repository.Path == "/" || path.Clean(repository.Path) != repository.Path || strings.ContainsFunc(origin.Repository, unicode.IsControl) {
		return errors.New("source repository requires an exact credential-free HTTPS repository URL")
	}
	if !commitID.MatchString(origin.Commit) || len(origin.Reference) < 1 || len(origin.Reference) > 256 || strings.TrimSpace(origin.Reference) != origin.Reference || strings.ContainsAny(origin.Reference, "\x00\\ ~^:?*[") || strings.ContainsFunc(origin.Reference, unicode.IsControl) || strings.HasPrefix(origin.Reference, "-") || strings.Contains(origin.Reference, "..") || strings.Contains(origin.Reference, "@{") {
		return errors.New("Git provenance requires a bounded reference and exact resolved commit")
	}
	return nil
}

func ValidateCatalog(manifest Manifest, document catalog.Document) error {
	if err := Validate(manifest); err != nil {
		return err
	}
	if document.Schema != catalog.SchemaVersion || len(document.Tests) == 0 && !manifest.Discover || len(document.Tests) > 1000 {
		return errors.New("prepared catalog requires the supported schema and 1 through 1000 tests")
	}
	if document.Suite != "" && document.Suite != manifest.Suite {
		return errors.New("catalog belongs to another suite")
	}
	for _, test := range document.Tests {
		if test.Runner != manifest.Runner || !RelativePath(test.Source, false) {
			return fmt.Errorf("test %s requires its declared runner and a source path within the suite context", test.ID)
		}
		if test.Application != "" && !slices.Contains(manifest.Applications, test.Application) {
			return fmt.Errorf("test %s application is absent from the manifest", test.ID)
		}
		for _, binding := range test.RequiredBindings {
			if !slices.Contains(manifest.RequiredBindings, binding) {
				return fmt.Errorf("test %s binding is absent from the manifest", test.ID)
			}
		}
		if manifest.Runner == "go" {
			entry, err := gosuite.Parse(test.Selector)
			if err != nil {
				return err
			}
			if !slices.Contains(manifest.Tests.Packages, entry.Package) || path.Dir(test.Source) != path.Join(manifest.ModuleRoot, entry.Package) || !strings.HasSuffix(test.Source, "_test.go") {
				return fmt.Errorf("test %s source or package does not match its manifest", test.ID)
			}
		} else if !playwrightSelector.MatchString(test.Selector) {
			return errors.New("Playwright discovery requires one exact permanent test tag")
		}
	}
	return nil
}

func ValidateDiscovery(discovery Discovery, image string) error {
	if discovery.Schema != DiscoverySchema || len(discovery.Log) > 128<<10 {
		return errors.New("invalid or oversized discovery report")
	}
	if err := ValidateOrigin(discovery.Origin); err != nil {
		return err
	}
	if err := ValidateCatalog(discovery.Manifest, discovery.Catalog); err != nil {
		return err
	}
	if len(discovery.Tests) != len(discovery.Catalog.Tests) {
		return errors.New("discovered tests do not match the complete declared catalog")
	}
	found := map[string]DiscoveredTest{}
	selectors := map[string]string{}
	for _, test := range discovery.Tests {
		if _, exists := found[test.ID]; exists || !RelativePath(test.File, false) || test.Line < 1 {
			return errors.New("discovered tests require unique identifiers and exact source locations")
		}
		if test.ParentOnly && discovery.Manifest.Runner != "go" {
			return errors.New("only declared Go subtests can use parent-only discovery")
		}
		found[test.ID], selectors[test.ID] = test, test.Selector
	}
	for _, test := range discovery.Catalog.Tests {
		actual, exists := found[test.ID]

		if !exists || actual.Selector != test.Selector || actual.File != test.Source {
			return fmt.Errorf("test %s differs from its discovered source or selector", test.ID)
		}
		if discovery.Manifest.Runner == "go" {
			entry, _ := gosuite.Parse(test.Selector)
			if actual.ParentOnly != strings.Contains(entry.Test, "/") {
				return errors.New("Go discovery must identify unresolved child paths")
			}
		}

	}
	issues := catalog.Validate(catalog.Draft{Tests: discovery.Catalog.Tests, Artifact: catalog.Artifact{Image: image, Discovered: selectors, Browser: discovery.Manifest.Browser}, Capture: catalog.DefaultCapture()})
	if len(issues) > 0 {
		return fmt.Errorf("discovered catalog is invalid: %s: %s", issues[0].Field, issues[0].Message)
	}
	return nil
}
