package httpapi_test

import "testing"

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
