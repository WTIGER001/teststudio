package httpapi_test

import (
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/evidence"
)

func TestEvidencePinsAccessAndAdministrativeDeletionThroughAPI(t *testing.T) {
	c, id := preparedRun(t)
	base := "/api/v1/projects/example/runs/" + id
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, r := c.call("GET", base, nil, nil)
		if r["data"].(map[string]any)["finishedAt"] != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := c.api.Evidence.ProcessRetention(t.Context(), 10); err != nil {
		t.Fatal(err)
	}
	a, err := c.api.Evidence.SaveBytes(t.Context(), evidence.Artifact{ID: "investigation", Project: "example", Run: id, Kind: "attachment", Filename: "investigation.txt", ContentType: "text/plain", ExpiresAt: time.Now().Add(-time.Hour)}, []byte("diagnostic evidence"), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	access := base + "/artifacts/" + a.ID + "/access"
	if status, _ := c.call("POST", access, map[string]string{"action": "view"}, nil); status != 410 {
		t.Fatalf("expired access: %d", status)
	}
	if status, data := c.call("PUT", base+"/retention/pin", map[string]any{"pinned": true, "reason": "Investigate the retained evidence"}, map[string]string{"If-Match": "0"}); status != 200 || data["pin"].(map[string]any)["pinned"] != true {
		t.Fatalf("pin: %d %v", status, data)
	}
	status, grant := c.call("POST", access, map[string]string{"action": "view"}, nil)
	if status != 200 {
		t.Fatal(grant)
	}
	response, err := http.Get(grant["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || string(data) != "diagnostic evidence" {
		t.Fatalf("pinned access: %d %q", response.StatusCode, data)
	}
	status, list := c.call("GET", base+"/artifacts", nil, nil)
	if status != 200 {
		t.Fatal(list)
	}
	var revision float64
	for _, value := range list["items"].([]any) {
		row := value.(map[string]any)
		if row["id"] == a.ID {
			revision = row["revision"].(float64)
		}
	}
	deletePath := base + "/artifacts/" + a.ID
	if status, _ := c.call("DELETE", deletePath, map[string]string{"reason": "Remove investigation file"}, map[string]string{"If-Match": strconv.Itoa(int(revision))}); status != 422 {
		t.Fatalf("pinned delete: %d", status)
	}
	if status, _ := c.call("PUT", base+"/retention/pin", map[string]any{"pinned": false, "reason": "Investigation complete"}, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatalf("unpin: %d", status)
	}
	if status, result := c.call("DELETE", deletePath, map[string]string{"reason": "Remove investigation file"}, map[string]string{"If-Match": strconv.Itoa(int(revision))}); status != 202 || result["data"].(map[string]any)["state"] != "delete-pending" {
		t.Fatalf("delete: %d %v", status, result)
	}
	if err := c.api.Evidence.Sweep(t.Context(), 100); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("POST", access, map[string]string{"action": "view"}, nil); status != 410 {
		t.Fatalf("deleted access: %d", status)
	}
	status, result := c.call("GET", base+"/retention", nil, nil)
	if status != 200 || result["pin"].(map[string]any)["deletionsInProgress"] != float64(0) || result["status"].(map[string]any)["state"] != "ready" {
		t.Fatalf("retention read: %d %v", status, result)
	}
}

func TestEvidenceRetentionMutationsRespectExplicitDeny(t *testing.T) {
	c, id := preparedRun(t)
	base := "/api/v1/projects/example/runs/" + id
	for _, permission := range []string{"teststudio.evidence.retain", "teststudio.evidence.delete"} {
		if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "example", PermissionName: permission, Effect: permissions.EffectDeny}); err != nil {
			t.Fatal(err)
		}
	}
	if status, _ := c.call("PUT", base+"/retention/pin", map[string]any{"pinned": true, "reason": "Not permitted"}, map[string]string{"If-Match": "0"}); status != 403 {
		t.Fatalf("pin bypassed deny: %d", status)
	}
	if status, _ := c.call("DELETE", base+"/artifacts/any", map[string]string{"reason": "Not permitted"}, map[string]string{"If-Match": "1"}); status != 403 {
		t.Fatalf("delete bypassed deny: %d", status)
	}
	if status, _ := c.call("GET", base+"/retention", nil, nil); status != 200 {
		t.Fatalf("read was coupled to mutation permission: %d", status)
	}
}
