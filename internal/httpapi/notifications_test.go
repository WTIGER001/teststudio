package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	localsecrets "github.com/wtiger001/teststudio/adapters/secrets/local"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/secrets"
)

func TestNotificationAPIKeepsReferencesPrivateAndRetriesExistingDeliveries(t *testing.T) {
	c, _ := preparedRun(t)
	var calls atomic.Int32
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(400)
			w.Write([]byte("sensitive-notification-token"))
		} else {
			w.WriteHeader(204)
		}
	}))
	t.Cleanup(receiver.Close)
	t.Setenv("TESTSTUDIO_API_NOTIFICATION_URL", receiver.URL+"/sensitive-notification-token")
	provider, err := localsecrets.New([]localsecrets.Binding{{ID: "alerts", Name: "Availability receiver", Project: "example", Purpose: secrets.NotificationWebhook, Environment: "TESTSTUDIO_API_NOTIFICATION_URL", Network: networkguard.Loopback}, {ID: "source-only", Project: "example", Purpose: "source-access", Environment: "TESTSTUDIO_API_NOTIFICATION_URL"}})
	if err != nil {
		t.Fatal(err)
	}
	c.api.Notifications.Secrets = provider
	base := "/api/v1/projects/example/notifications"
	status, refs := c.call("GET", base+"/references", nil, nil)
	encoded, _ := json.Marshal(refs)
	if status != 200 || len(refs["items"].([]any)) != 1 || strings.Contains(string(encoded), "sensitive-notification-token") || strings.Contains(string(encoded), "TESTSTUDIO_API_NOTIFICATION_URL") {
		t.Fatalf("private reference listing: %d %s", status, encoded)
	}
	d := notifications.Definition{ID: "availability", Name: "Availability", Enabled: true, Kind: "webhook", Reference: "alerts", Outcomes: []string{"passed"}, MaxAttempts: 2}
	if status, data := c.call("POST", base+"/channels", d, nil); status != 200 {
		t.Fatalf("channel: %d %v", status, data)
	}
	if status, _ := c.call("PUT", base+"/channels/availability", d, map[string]string{"If-Match": "9"}); status != 409 {
		t.Fatal("stale notification configuration overwrote another revision")
	}
	foreign := d
	foreign.ID = "invalid-purpose"
	foreign.Reference = "source-only"
	if status, _ := c.call("POST", base+"/channels", foreign, nil); status != 422 {
		t.Fatal("notification borrowed a different secret purpose")
	}
	if status, _ := c.call("GET", "/api/v1/projects/other/notifications/channels/availability", nil, nil); status != 404 {
		t.Fatal("foreign project exposed a channel")
	}
	status, created := c.call("POST", "/api/v1/projects/example/runs", runs.Request{Suite: "health", Environment: "local"}, map[string]string{"Idempotency-Key": "notification-run"})
	if status != 200 {
		t.Fatal(created)
	}
	id := created["id"].(string)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, data := c.call("GET", "/api/v1/projects/example/runs/"+id, nil, nil)
		if runs.Terminal(data["data"].(map[string]any)["state"].(string)) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := c.api.Notifications.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	status, page := c.call("GET", base+"/deliveries?run="+id, nil, nil)
	if status != 200 || len(page["items"].([]any)) != 1 {
		t.Fatalf("deliveries: %d %v", status, page)
	}
	row := page["items"].([]any)[0].(map[string]any)
	deliveryID := row["id"].(string)
	delivery := row["data"].(map[string]any)
	if delivery["state"] != "failed" || delivery["httpStatus"] != float64(400) || calls.Load() != 1 {
		t.Fatalf("permanent failure: %v", delivery)
	}
	encoded, _ = json.Marshal(page)
	if strings.Contains(string(encoded), "sensitive-notification-token") || strings.Contains(string(encoded), receiver.URL) {
		t.Fatal("delivery metadata exposed a destination or response body")
	}
	resource := base + "/deliveries/" + deliveryID
	if status, _ := c.call("POST", resource+"/retry", map[string]any{}, nil); status != 422 {
		t.Fatal("retry omitted its revision")
	}
	if status, _ := c.call("POST", resource+"/retry", map[string]any{}, map[string]string{"If-Match": "1"}); status != 409 {
		t.Fatal("stale retry bypassed revision checks")
	}
	revision := fmt.Sprint(int64(row["revision"].(float64)))
	if status, data := c.call("POST", resource+"/retry", map[string]any{}, map[string]string{"If-Match": revision}); status != 200 {
		t.Fatalf("retry: %d %v", status, data)
	}
	if _, err := c.api.Notifications.Tick(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	status, row = c.call("GET", resource, nil, nil)
	if status != 200 || row["data"].(map[string]any)["state"] != "delivered" || calls.Load() != 2 {
		t.Fatalf("delivery retry: %d %v", status, row)
	}
	status, history := c.call("GET", resource+"/attempts", nil, nil)
	if status != 200 || len(history["items"].([]any)) != 2 {
		t.Fatal("retry erased original attempt history")
	}
	object := "example/run/" + id
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", PermissionName: "teststudio.runs.read", TeamScope: "example", ObjectScope: &object, Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("GET", resource, nil, nil); status != 403 {
		t.Fatal("delivery exposed a denied run")
	}
	status, page = c.call("GET", base+"/deliveries", nil, nil)
	if status != 200 || len(page["items"].([]any)) != 0 {
		t.Fatal("delivery listing exposed denied run metadata")
	}
}
