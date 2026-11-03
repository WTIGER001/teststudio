package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/sources"
)

func TestPlaywrightEditorAPIRequiresSessionCSRFAndExplicitRevision(t *testing.T) {
	c := fixture(t, func(api *httpapi.Server) {
		api.Sources = &sources.Service{Store: api.Store, Auth: api.Auth, Location: "local", BundleRunners: []string{"playwright"}}
	})
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", map[string]string{"id": "editor", "name": "Editor"}, nil); status != 200 {
		t.Fatal(data)
	}
	base := "/api/v1/projects/editor/suites/browser"
	if status, data := c.call("PUT", base, map[string]any{"id": "browser", "name": "Browser", "applications": []string{}, "requiresTarget": false}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	base += "/playwright-drafts"
	status, starter := c.call("GET", base+"/starter", nil, nil)
	if status != 200 {
		t.Fatal(starter)
	}
	starter["id"] = "welcome"
	for _, endpoint := range []string{base, base + "/starter", base + "/welcome"} {
		response, err := http.Get(c.origin + endpoint)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 401 {
			t.Fatal("anonymous source read accepted")
		}
	}
	if status, _ := c.call("PUT", base+"/welcome", starter, map[string]string{"If-Match": "0", "X-CSRF-Token": ""}); status != 403 {
		t.Fatal("source mutation accepted missing CSRF")
	}
	if status, _ := c.call("PUT", base+"/welcome", starter, nil); status != 422 {
		t.Fatal("implicit revision accepted")
	}
	if status, data := c.call("PUT", base+"/welcome", starter, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	if status, _ := c.call("POST", base+"/welcome/prepare", nil, map[string]string{"If-Match": "1", "Idempotency-Key": "once", "X-CSRF-Token": ""}); status != 403 {
		t.Fatal("preparation accepted missing CSRF")
	}
	if status, _ := c.call("POST", base+"/welcome/prepare", nil, map[string]string{"If-Match": "1"}); status != 422 {
		t.Fatal("implicit idempotency key accepted")
	}
	if status, data := c.call("POST", base+"/welcome/prepare", nil, map[string]string{"If-Match": "1", "Idempotency-Key": "once"}); status != 200 {
		t.Fatal(data)
	}
	if status, _ := c.call("GET", base+"?limit=101", nil, nil); status != 422 {
		t.Fatal("unbounded draft scan accepted")
	}
}
