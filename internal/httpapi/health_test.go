package httpapi_test

import (
	"net/http/httptest"
	"testing"
)

func TestLivenessRemainsAvailableWhenDatabaseReadinessFails(t *testing.T) {
	client := fixture(t)
	for _, path := range []string{"/livez", "/healthz"} {
		status, body := client.call("GET", path, nil, nil)
		if status != 200 || body["service"] != "teststudio" || body["status"] != "up" {
			t.Fatal(path, status, body)
		}
	}
	if err := client.api.Store.Close(); err != nil {
		t.Fatal(err)
	}
	status, body := client.call("GET", "/livez", nil, nil)
	if status != 200 || body["status"] != "up" {
		t.Fatal("liveness depends on database readiness", status, body)
	}
	status, body = client.call("GET", "/healthz", nil, nil)
	if status != 503 || body["status"] != "unavailable" {
		t.Fatal("unavailable storage remained ready", status, body)
	}
}

func TestArtifactLivenessAndReadinessExposeNoApplicationRoutes(t *testing.T) {
	client := fixture(t)
	handler := client.api.ArtifactHandler()
	call := func(path string, expected int) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != expected {
			t.Fatal(path, response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("artifact response caching was enabled")
		}
	}
	call("/livez", 200)
	call("/healthz", 200)
	call("/", 404)
	call("/auth/session", 404)
	call("/api/v1/projects", 404)
	if err := client.api.Store.Close(); err != nil {
		t.Fatal(err)
	}
	call("/livez", 200)
	call("/healthz", 503)
}
