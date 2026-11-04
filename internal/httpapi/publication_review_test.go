package httpapi_test

import (
	"testing"
)

func TestPublicationReviewThroughAPIAndStaleRecovery(t *testing.T) {
	c, _ := preparedRun(t)
	base := "/api/v1/projects/example/suites/health"
	status, draft := c.call("POST", base+"/catalog-drafts", map[string]any{}, nil)
	if status != 200 {
		t.Fatal(status, draft)
	}
	id := draft["id"].(string)
	route := base + "/catalog-drafts/" + id
	if status, _ := c.call("POST", route+"/publication-reviews", map[string]any{}, nil); status != 422 {
		t.Fatal(status)
	}
	status, review := c.call("POST", route+"/publication-reviews", map[string]any{}, map[string]string{"If-Match": "1"})
	if status != 200 || review["activeReleaseId"] == nil || review["afterSettings"] == nil {
		t.Fatal(status, review)
	}
	reviewID := review["id"].(string)
	if status, _ := c.call("GET", base+"/publication-reviews/"+reviewID, nil, nil); status != 200 {
		t.Fatal(status)
	}
	status, test := c.call("GET", base+"/tests/HEALTH-1", nil, nil)
	if status != 200 {
		t.Fatal(status, test)
	}
	value := test["data"].(map[string]any)
	value["title"] = "Concurrent UI review change"
	if status, _ := c.call("PUT", base+"/tests/HEALTH-1", value, map[string]string{"If-Match": "1"}); status != 200 {
		t.Fatal(status)
	}
	if status, _ := c.call("POST", route+"/publish", map[string]any{"activate": true, "reviewId": reviewID}, map[string]string{"If-Match": "1", "Idempotency-Key": "stale-review"}); status != 409 {
		t.Fatal("stale reviewed publication", status)
	}
	status, draft = c.call("POST", base+"/catalog-drafts", map[string]any{}, nil)
	if status != 200 {
		t.Fatal(status, draft)
	}
	route = base + "/catalog-drafts/" + draft["id"].(string)
	status, review = c.call("POST", route+"/publication-reviews", map[string]any{}, map[string]string{"If-Match": "1"})
	if status != 200 {
		t.Fatal(status, review)
	}
	body := map[string]any{"activate": true, "reviewId": review["id"]}
	headers := map[string]string{"If-Match": "1", "Idempotency-Key": "current-review"}
	status, first := c.call("POST", route+"/publish", body, headers)
	if status != 200 {
		t.Fatal(status, first)
	}
	status, replay := c.call("POST", route+"/publish", body, headers)
	if status != 200 || replay["id"] != first["id"] {
		t.Fatal(status, replay)
	}
	if status, _ := c.call("GET", "/api/v1/projects/foreign/suites/health/publication-reviews/"+reviewID, nil, nil); status != 404 {
		t.Fatal(status)
	}
}
