package checks

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStatus(t *testing.T) {
	origin := os.Getenv("TESTSTUDIO_API_BASE_URL")
	if origin == "" {
		t.Fatal("The reviewed request relay is required")
	}
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Get(origin + "/api/private/status")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]string
	err = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&result)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || result["status"] != "ready" {
		t.Fatal("The registered HTTPS service did not return its authenticated status")
	}
	if !strings.Contains(result["receivedCredential"], "[REDACTED]") || !strings.Contains(response.Header.Get("X-Credential-Echo"), "[REDACTED]") {
		t.Fatal("The gateway did not redact the reflected fixture credential")
	}
	for _, rejected := range []struct{ method, path string }{{"POST", "/api/private/status"}, {"GET", "/api/status"}} {
		request, err := http.NewRequest(rejected.method, origin+rejected.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		denied, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		denied.Body.Close()
		if denied.StatusCode != http.StatusForbidden {
			t.Fatal("An unreviewed method or path reached the fixture")
		}
	}
	connection, err := net.DialTimeout("tcp", "gateway-target.teststudio-local.svc:8443", 500*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("The uploaded worker bypassed its trusted gateway")
	}
	t.Log("Authenticated HTTPS status, reflected-secret redaction, reviewed policy, and direct-egress denial pass")
}
