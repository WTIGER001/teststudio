package httpapi_test

import (
	"testing"

	permissions "github.com/wtiger001/go-permissions"
)

func TestBulkMetadataPlanRejectsExecutableFieldsAndRevokedRead(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	input := map[string]any{"action": "metadata", "tests": []any{map[string]any{"id": "HEALTH-1", "revision": 1}}, "metadata": map[string]any{"selector": "@unreviewed"}}
	if status, _ := c.call("POST", base+"/tests/bulk-preview", input, nil); status != 422 {
		t.Fatal("metadata accepted executable source fields")
	}
	input["metadata"] = map[string]any{"owner": "Example team"}
	status, plan := c.call("POST", base+"/tests/bulk-preview", input, nil)
	if status != 200 {
		t.Fatal(status, plan)
	}
	object := "example/test/health/HEALTH-1"
	if err := c.api.Auth.Engine.CreateGrant(t.Context(), permissions.Grant{OwnerKind: permissions.PrincipalUser, OwnerID: "admin", TeamScope: "example", ObjectScope: &object, PermissionName: "teststudio.catalog.read", Effect: permissions.EffectDeny}); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("POST", base+"/tests/bulk-preview", input, nil); status != 403 {
		t.Fatal("new preview ignored an exact read denial")
	}
	if status, _ := c.call("GET", base+"/catalog-imports/"+plan["id"].(string), nil, nil); status != 403 {
		t.Fatal("stored preview exposed revoked test data")
	}
	if status, _ := c.call("POST", base+"/catalog-imports/"+plan["id"].(string)+"/apply", nil, map[string]string{"If-Match": "1"}); status != 403 {
		t.Fatal("apply ignored an exact read denial")
	}
	row, err := c.api.Store.Get(t.Context(), "tests/example/health", "HEALTH-1")
	if err != nil || row.Revision != 1 {
		t.Fatal("rejected preview changed a test", row, err)
	}
}
