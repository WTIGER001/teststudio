package httpapi_test

import (
	"encoding/json"
	"testing"
)

func TestVersionIsPublicReadOnlyAndUncached(t *testing.T) {
	c := fixture(t)
	response, err := c.http.Get(c.origin + "/api/v1/version")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var value map[string]any
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || response.Header.Get("Cache-Control") != "no-store" || value["version"] == "" {
		t.Fatal(response.StatusCode, response.Header, value)
	}
	if value["user"] != nil || value["session"] != nil {
		t.Fatal("version includes identity data")
	}
}
