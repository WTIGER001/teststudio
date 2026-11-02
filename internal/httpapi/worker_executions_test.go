package httpapi_test

import (
	"encoding/json"
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
	"strings"
	"testing"
	"time"
)

func TestWorkerHistoryScopesAccessAndHidesInternalClaimTokens(t *testing.T) {
	c, id := preparedRun(t)
	ledger := workerstate.Ledger{Store: c.api.Store}
	installation, err := ledger.Installation(t.Context(), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, executionID := range []string{"one", "two"} {
		ref := execution.Reference{ID: executionID, Installation: installation, Project: "example", Run: id, Test: "HEALTH-1", Attempt: 1, Worker: "worker", Profile: "playwright-local-isolated-v1"}
		record, changes, err := ledger.Begin(ref)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.api.Store.Commit(t.Context(), changes...); err != nil {
			t.Fatal(err)
		}
		_, row, err := ledger.Get(t.Context(), id, executionID)
		if err != nil {
			t.Fatal(err)
		}
		record.Claim = "private-claim-token"
		until := time.Now().Add(time.Minute)
		record.ClaimUntil = &until
		record.State = "recovering"
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
		if err == nil {
			err = c.api.Store.Commit(t.Context(), change)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	base := "/api/v1/projects/example/runs/" + id + "/executions"
	status, page := c.call("GET", base+"?limit=1", nil, nil)
	if status != 200 || len(page["items"].([]any)) != 1 || page["cursor"] == nil {
		t.Fatalf("history: %d %v", status, page)
	}
	data, _ := json.Marshal(page)
	if strings.Contains(string(data), "private-claim-token") || strings.Contains(string(data), "claimToken") {
		t.Fatal("worker history exposed internal claim ownership")
	}
	if status, next := c.call("GET", base+"?limit=1&cursor="+page["cursor"].(string), nil, nil); status != 200 || len(next["items"].([]any)) != 1 {
		t.Fatal("worker history pagination failed")
	}
	if status, _ := c.call("GET", "/api/v1/projects/other/runs/"+id+"/executions", nil, nil); status != 404 {
		t.Fatal("worker history crossed projects")
	}
	if err := c.api.Auth.Engine.UnassignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	if status, _ := c.call("GET", base, nil, nil); status != 403 {
		t.Fatal("worker history ignored permission revocation")
	}
}
