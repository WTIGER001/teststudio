package httpapi_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/recording"
)

func TestCompanionAPIUsesScopedCapabilitiesWithoutBrowserSessionAuthority(t *testing.T) {
	c := fixture(t, func(api *httpapi.Server) {
		api.Recordings = &recording.Service{Store: api.Store, Auth: api.Auth, CompanionEnabled: true}
	})
	c.signIn()
	if status, data := c.call("POST", "/api/v1/projects", map[string]string{"id": "recording", "name": "Recording"}, nil); status != 200 {
		t.Fatal(data)
	}
	suite := "/api/v1/projects/recording/suites/browser"
	if status, data := c.call("PUT", suite, map[string]any{"id": "browser", "name": "Browser", "requiresTarget": true}, map[string]string{"If-Match": "0"}); status != 200 {
		t.Fatal(data)
	}
	base := suite + "/browser-recordings"
	if status, data := c.call("POST", base, recording.Input{ID: "session", TestID: "REC-001", Title: "Continue", URL: "https://example.test"}, nil); status != 200 {
		t.Fatal(data)
	}
	if status, _ := c.call("POST", base+"/session/pairing", nil, map[string]string{"X-CSRF-Token": ""}); status != 403 {
		t.Fatal("pairing omitted CSRF")
	}
	status, pairing := c.call("POST", base+"/session/pairing", nil, nil)
	if status != 200 {
		t.Fatal("owner could not pair", status)
	}
	device := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	pair := recording.PairRequest{PairingToken: pairing["token"].(string), Instance: "desktop", DeviceToken: device}
	raw := func(method, path string, input any, headers map[string]string) (int, map[string]any) {
		t.Helper()
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		r, err := http.NewRequest(method, c.origin+path, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Content-Type", "application/json")
		for key, value := range headers {
			r.Header.Set(key, value)
		}
		response, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if len(response.Cookies()) != 0 || response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("companion response issued a cookie or allowed caching")
		}
		var result map[string]any
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, result
	}
	capBase := "/api/v1/recording-companion/sessions/session"
	for _, headers := range []map[string]string{{"Origin": c.origin}, {"Cookie": "session=unused"}, {"Sec-Fetch-Site": "same-origin"}} {
		if status, _ := raw("POST", capBase+"/pair", pair, headers); status != 403 {
			t.Fatal("browser pair request accepted", status)
		}
	}
	if status, _ := raw("POST", capBase+"/pair", pair, nil); status != 200 {
		t.Fatal("desktop pairing rejected", status)
	}
	headers := map[string]string{"Authorization": "Bearer " + device}
	if status, _ := raw("POST", capBase+"/control", map[string]string{"instance": "desktop"}, headers); status != 200 {
		t.Fatal("paired control rejected", status)
	}
	if status, _ := raw("POST", "/api/v1/recording-companion/sessions/other/control", map[string]string{"instance": "desktop"}, headers); status != 403 {
		t.Fatal("capability crossed session", status)
	}
	if status, _ := raw("GET", "/api/v1/projects", nil, headers); status != 401 {
		t.Fatal("device became application session", status)
	}
	if status, _ := raw("GET", base+"/session", nil, headers); status != 401 {
		t.Fatal("device gained private owner API", status)
	}
	code := "import { test } from '@playwright/test';\ntest('test', async ({ page }) => { await page.goto('https://example.test'); });\n"
	if status, _ := raw("POST", capBase+"/snapshots", map[string]any{"instance": "desktop", "snapshot": recording.CompanionSnapshot{Sequence: 1, State: "ready", Code: code}}, headers); status != 200 {
		t.Fatal("captured source rejected", status)
	}
	if status, _ := c.call("GET", base+"/session/source", nil, nil); status != 200 {
		t.Fatal("hosted recording could not use reviewed draft conversion", status)
	}
	if status, data := c.call("GET", "/api/v1/projects/recording/runs", nil, nil); status != 200 || len(data["items"].([]any)) != 0 {
		t.Fatal("pairing or capture executed a test")
	}
}
