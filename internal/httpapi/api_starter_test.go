package httpapi_test

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/sources"
)

func TestAPIStarterEndpointKeepsExplicitTypeAndNormalDraftContracts(t *testing.T) {
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", BundleRunners: []string{"playwright"}}
	})
	base := "/api/v1/projects/api-authoring/suites/requests/playwright-drafts"
	if status, _ := c.call("GET", base+"/starter?type=api", nil, nil); status != 401 {
		t.Fatal("anonymous API source read succeeds")
	}
	c.signIn()
	accessCall(t, c, "POST", "/api/v1/projects", map[string]string{"id": "api-authoring", "name": "API authoring"}, "", 200)
	accessCall(t, c, "PUT", "/api/v1/projects/api-authoring/suites/requests", map[string]any{"id": "requests", "name": "Request checks", "applications": []string{}, "requiresTarget": true}, "0", 200)
	starter := accessCall(t, c, "GET", base+"/starter?type=api", nil, "", 200)
	if starter["manifestPath"] != "teststudio-suite.json" || len(starter["files"].([]any)) != 6 {
		t.Fatal("API starter is not a complete workspace")
	}
	for _, query := range []string{"type=unknown", "type=", "type=api&type=browser"} {
		accessCall(t, c, "GET", base+"/starter?"+query, nil, "", 422)
	}
	starter["id"] = "request-source"
	accessCall(t, c, "PUT", base+"/request-source", starter, "0", 200)
	row := accessCall(t, c, "GET", base+"/request-source", nil, "", 200)
	if !strings.Contains(row["data"].(map[string]any)["files"].([]any)[0].(map[string]any)["content"].(string), "async ({ request })") {
		t.Fatal("saved API starter changes its direct request fixture")
	}
}
