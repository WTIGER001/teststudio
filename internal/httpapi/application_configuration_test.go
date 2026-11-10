package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/secrets"
	"slices"
	"strconv"
	"strings"
	"testing"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestApplicationConfigurationOwnsDefaultSuiteAndPreservesConflicts(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/applications/app/configuration"
	status, value := c.call("GET", base, nil, nil)
	if status != 200 {
		t.Fatal(status, value)
	}
	app := value["application"].(map[string]any)
	if app["data"].(map[string]any)["defaultSuiteId"] != "app-default" {
		t.Fatal("default suite missing")
	}
	suite, err := c.api.Store.Get(t.Context(), projects.Bucket("suites", "example"), "app-default")
	if err != nil {
		t.Fatal(err)
	}
	initial, _ := storage.Decode[projects.Suite](suite)
	if initial.Name != "Default" || len(initial.Applications) != 1 || initial.Applications[0] != "app" {
		t.Fatal("default suite owner missing")
	}
	tests, err := c.api.Store.All(t.Context(), "tests/example/app-default", "example")
	if err != nil || len(tests) != 0 {
		t.Fatal("default suite contains tests")
	}
	requirements := []projects.Requirement{{Name: "tenantId", Kind: "parameter", ValueType: "integer", Required: true}, {Name: "apiToken", Kind: "secret"}}
	revision := int64(app["revision"].(float64))
	body := map[string]any{"requirements": requirements}
	header := map[string]string{"If-Match": strconv.FormatInt(revision, 10)}
	if status, value := c.call("PUT", base, body, header); status != 200 {
		t.Fatal(status, value)
	}
	if status, _ := c.call("PUT", base, body, header); status != 409 {
		t.Fatal("stale contract did not conflict", status)
	}
	_, value = c.call("GET", base, nil, nil)
	environments := value["environments"].([]any)
	if len(environments) != 1 || environments[0].(map[string]any)["readiness"].(map[string]any)["ready"] != false {
		t.Fatal("missing requirements appear ready")
	}
	e := environments[0].(map[string]any)
	envHeader := map[string]string{"If-Match": strconv.Itoa(int(e["revision"].(float64)))}
	endpoints := e["data"].(map[string]any)["endpoints"]
	envBody := map[string]any{"endpoints": endpoints, "parameters": map[string]string{"tenantId": "7"}}
	if status, result := c.call("PUT", strings.TrimSuffix(base, "/configuration")+"/environments/local/configuration", envBody, envHeader); status != 200 {
		t.Fatal(status, result)
	}
	if status, _ := c.call("PUT", strings.TrimSuffix(base, "/configuration")+"/environments/local/configuration", envBody, envHeader); status != 409 {
		t.Fatal("stale environment did not conflict")
	}
	shared := projects.Suite{ID: "shared-new", Name: "Shared", Applications: []string{"app", "app"}}
	if status, _ := c.call("PUT", "/api/v1/projects/example/suites/shared-new", shared, map[string]string{"If-Match": "0"}); status != 422 {
		t.Fatal("shared suite accepted", status)
	}
	if status, _ := c.call("PUT", "/api/v1/projects/example/suites/unowned", projects.Suite{ID: "unowned", Name: "Unowned"}, map[string]string{"If-Match": "0"}); status != 422 {
		t.Fatal("unassigned suite accepted", status)
	}
}

func TestApplicationConfigurationScopesEnvironmentsAndExactAuthority(t *testing.T) {
	c, _ := preparedRun(t)
	if err := c.api.Projects.SaveApplication(t.Context(), "admin", "example", projects.Application{ID: "other", Name: "Other"}, 0); err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/projects/example/applications/other"
	if status, _ := c.call("PUT", base+"/environments/local/configuration", map[string]any{"parameters": map[string]string{"tenantId": "foreign"}}, map[string]string{"If-Match": "1"}); status != 404 {
		t.Fatal("foreign application changed environment", status)
	}
	status, result := c.call("GET", base+"/configuration", nil, nil)
	if status != 200 || len(result["environments"].([]any)) != 0 {
		t.Fatal("foreign environment in matrix", status)
	}
	object := "example/environment/local"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.environments.read", TeamScope: "example", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	status, result = c.call("GET", "/api/v1/projects/example/applications/app/configuration", nil, nil)
	if status != 200 || len(result["environments"].([]any)) != 0 {
		t.Fatal("denied environment in matrix")
	}
	object = "example/application/app"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.applications.manage", TeamScope: "example", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("PUT", "/api/v1/projects/example/applications/app/configuration", map[string]any{"requirements": []projects.Requirement{}}, map[string]string{"If-Match": "1"}); status != 403 {
		t.Fatal("exact deny bypassed", status)
	}
}

func TestApplicationSuiteAssignmentPreservesHistoryAndRejectsForeignTests(t *testing.T) {
	c, runID := preparedRun(t)
	row, err := c.api.Store.Get(t.Context(), projects.Bucket("suites", "example"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, _ := storage.Decode[projects.Suite](row)
	suite.Applications = []string{"app", "other"}
	change, _ := storage.Put(row.Bucket, row.ID, "example", row.Revision, suite)
	if err := c.api.Store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Projects.SaveApplication(t.Context(), "admin", "example", projects.Application{ID: "other", Name: "Other"}, 0); err != nil {
		t.Fatal(err)
	}
	before, _ := c.api.Store.Get(t.Context(), "runs", runID)
	input := projects.SuiteSettings{Name: suite.Name, Applications: []string{"other"}, RequiresTarget: true}
	if _, err := c.api.Projects.UpdateSuiteSettings(t.Context(), "admin", "example", "health", input, row.Revision+1); err == nil {
		t.Fatal("foreign test silently reassigned")
	}
	input.Applications = []string{"app"}
	if _, err := c.api.Projects.UpdateSuiteSettings(t.Context(), "admin", "example", "health", input, row.Revision+1); err != nil {
		t.Fatal(err)
	}
	after, _ := c.api.Store.Get(t.Context(), "runs", runID)
	var a, b map[string]any
	json.Unmarshal(before.Data, &a)
	json.Unmarshal(after.Data, &b)
	if a["release"].(map[string]any)["id"] != b["release"].(map[string]any)["id"] {
		t.Fatal("assignment rewrote run release")
	}
}

func TestApplicationDefaultSuiteCreationIsAtomicAndLegacyCreationIsExplicit(t *testing.T) {
	c, _ := preparedRun(t)
	collision, _ := storage.Put(projects.Bucket("suites", "example"), "collision-default", "example", 0, projects.Suite{ID: "collision-default", Name: "Existing", Applications: []string{"app"}})
	if err := c.api.Store.Commit(t.Context(), collision); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Projects.SaveApplication(t.Context(), "admin", "example", projects.Application{ID: "collision", Name: "Collision"}, 0); err == nil {
		t.Fatal("default collision accepted")
	}
	if _, err := c.api.Store.Get(t.Context(), projects.Bucket("applications", "example"), "collision"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("partial application committed", err)
	}
	legacy, _ := storage.Put(projects.Bucket("applications", "example"), "legacy", "example", 0, projects.Application{ID: "legacy", Name: "Legacy"})
	if err := c.api.Store.Commit(t.Context(), legacy); err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/projects/example/applications/legacy"
	status, response := c.call("GET", base+"/configuration", nil, nil)
	if status != 200 || response["requirements"] == nil || len(response["requirements"].([]any)) != 0 {
		t.Fatal("empty contract is not an array", status)
	}
	if status, result := c.call("POST", base+"/default-suite", nil, map[string]string{"If-Match": "1"}); status != 200 || result["data"].(map[string]any)["defaultSuiteId"] != "legacy-default" {
		t.Fatal("explicit default creation failed", status)
	}
	if status, _ := c.call("POST", base+"/default-suite", nil, map[string]string{"If-Match": "2"}); status != 200 {
		t.Fatal("confirmed default retry created duplicate", status)
	}
}

type applicationCredentialMetadata struct{ origin string }

func (p applicationCredentialMetadata) References(context.Context, string, string) ([]secrets.Reference, error) {
	return []secrets.Reference{{ID: "owned", Name: "Owned target credential", Origins: []string{p.origin}}, {ID: "foreign", Name: "Other target credential", Origins: []string{"https://other-target.test"}}}, nil
}
func (p applicationCredentialMetadata) Resolve(context.Context, string, string, string) (secrets.Value, error) {
	return secrets.Value{}, secrets.ErrUnavailable
}

func TestApplicationConfigurationCredentialMetadataRespectsApplicationTargets(t *testing.T) {
	c, _ := preparedRun(t)
	row, err := c.api.Store.Get(t.Context(), projects.Bucket("environments", "example"), "local")
	if err != nil {
		t.Fatal(err)
	}
	environment, _ := storage.Decode[projects.Environment](row)
	c.api.Projects.Secrets = applicationCredentialMetadata{origin: environment.Endpoints["api"]}
	if err := c.api.Auth.Directory.SaveUser(t.Context(), identity.User{ID: "app-manager", Name: "Application manager"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "app-manager"}, authorization.ContributorRole, map[string]any{"team": "example", "application": "app"}); err != nil {
		t.Fatal(err)
	}
	refs, err := c.api.Projects.ApplicationCredentialReferences(t.Context(), "app-manager", "example", "app")
	if err != nil || len(refs) != 1 || refs[0].ID != "owned" {
		t.Fatal("application credential listing crosses target scope", err)
	}
	admin, err := c.api.Projects.ApplicationCredentialReferences(t.Context(), "admin", "example", "app")
	if err != nil || len(admin) != 2 {
		t.Fatal("project operator lost approved reference catalog", err)
	}
}

func TestApplicationConfigurationIncludesOnlyReadableBrowserRequirements(t *testing.T) {
	c, _ := preparedRun(t)
	test := catalog.Test{ID: "BROWSER-1", Title: "Browser", Runner: "playwright", Type: "browser", State: "enabled", Safety: "safe", TimeoutSeconds: 10, Category: "Browser", Rationale: "Reads a fixture", Source: "browser.spec.ts", Selector: "@BROWSER-1"}
	if err := c.api.Catalog.SaveTest(t.Context(), "admin", "example", "app-default", test, 0); err != nil {
		t.Fatal(err)
	}
	row, _ := c.api.Store.Get(t.Context(), projects.Bucket("suites", "example"), "app-default")
	suite, _ := storage.Decode[projects.Suite](row)
	suite.ActiveRelease = "browser-version"
	release, _ := storage.Put("releases/example/app-default", "browser-version", "example", 0, catalog.Release{ID: "browser-version", Draft: catalog.Draft{Artifact: catalog.Artifact{Browser: &browserpolicy.Spec{Endpoint: "website"}}}})
	changed, _ := storage.Put(row.Bucket, row.ID, "example", row.Revision, suite)
	if err := c.api.Store.Commit(t.Context(), release, changed); err != nil {
		t.Fatal(err)
	}
	result, err := c.api.Projects.Configuration(t.Context(), "admin", "example", "app")
	if err != nil || !slices.ContainsFunc(result.Requirements, func(r projects.Requirement) bool { return r.Kind == "endpoint" && r.Name == "website" }) {
		t.Fatal("saved browser endpoint absent from readiness", err)
	}
	object := "example/test/app-default/BROWSER-1"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.catalog.read", TeamScope: "example", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	result, err = c.api.Projects.Configuration(t.Context(), "admin", "example", "app")
	if err != nil || slices.ContainsFunc(result.Requirements, func(r projects.Requirement) bool { return r.Name == "website" }) {
		t.Fatal("denied browser test exposed its endpoint", err)
	}
}

func TestApplicationConfigurationRemovedParameterDeclarationStaysVisibleAsInvalid(t *testing.T) {
	c, _ := preparedRun(t)
	if err := c.api.Projects.SaveRequirements(t.Context(), "admin", "example", "app", []projects.Requirement{{Name: "tenantId", Kind: "parameter"}}, 1); err != nil {
		t.Fatal(err)
	}
	row, _ := c.api.Store.Get(t.Context(), catalog.Bucket("tests", "example", "health"), "HEALTH-1")
	test, _ := storage.Decode[catalog.Test](row)
	test.RequiredParameters = []string{"tenantId"}
	if err := c.api.Catalog.SaveTest(t.Context(), "admin", "example", "health", test, row.Revision); err != nil {
		t.Fatal(err)
	}
	if err := c.api.Projects.SaveRequirements(t.Context(), "admin", "example", "app", nil, 2); err != nil {
		t.Fatal(err)
	}
	configuration, err := c.api.Projects.Configuration(t.Context(), "admin", "example", "app")
	if err != nil || len(configuration.Environments) != 1 {
		t.Fatal(err)
	}
	ready := configuration.Environments[0].Readiness
	if ready.Ready || !slices.ContainsFunc(ready.Items, func(item projects.ConfigurationItem) bool { return item.Name == "tenantId" && item.Status == "invalid" }) {
		t.Fatal("removed parameter disappeared from readiness")
	}
}
