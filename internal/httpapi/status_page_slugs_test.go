package httpapi_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/internal/probes"
	"github.com/wtiger001/teststudio/internal/statuspages"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestStatusVanityURLsReserveGloballyAndFollowPublishedSnapshot(t *testing.T) {
	c, _, binding := probeFixture(t)
	base := "/api/v1/projects/example/status-pages/vanity-one"
	d := statuspages.Definition{ID: "vanity-one", Slug: " Customer-Health ", Title: "Health", Background: "#ffffff", Window: "24h", MaxAge: 900, Probes: []statuspages.Entry{{Probe: binding.ID, Title: "Website"}}}
	check := func(name string, available bool) {
		t.Helper()
		value := accessCall(t, c, "GET", "/api/v1/projects/example/status-page-slugs/"+name+"?pageId="+d.ID, nil, "", 200)
		if value["available"] != available {
			t.Fatal(value)
		}
	}
	check("CUSTOMER-HEALTH", true)
	row := accessCall(t, c, "PUT", base, d, "0", 200)
	token := row["data"].(map[string]any)["publicToken"].(string)
	if row["data"].(map[string]any)["slug"] != "customer-health" {
		t.Fatal(row)
	}
	accessCall(t, c, "GET", "/api/v1/public/status-pages/customer-health", nil, "", 404)
	check("customer-health", true)
	d.ID = "another-page"
	check("customer-health", false)
	accessCall(t, c, "PUT", "/api/v1/projects/example/status-pages/another-page", d, "0", 409)
	d.ID = "vanity-one"
	accessCall(t, c, "POST", base+"/publish", nil, "1", 200)
	for _, address := range []string{"customer-health", "CUSTOMER-HEALTH", token} {
		accessCall(t, c, "GET", "/api/v1/public/status-pages/"+address, nil, "", 200)
	}
	d.Slug = "new-health"
	accessCall(t, c, "PUT", base, d, "2", 200)
	accessCall(t, c, "GET", "/api/v1/public/status-pages/customer-health", nil, "", 200)
	accessCall(t, c, "GET", "/api/v1/public/status-pages/new-health", nil, "", 404)
	accessCall(t, c, "POST", base+"/publish", nil, "3", 200)
	accessCall(t, c, "GET", "/api/v1/public/status-pages/customer-health", nil, "", 404)
	accessCall(t, c, "GET", "/api/v1/public/status-pages/new-health", nil, "", 200)
	accessCall(t, c, "GET", "/api/v1/public/status-pages/"+token, nil, "", 200)
	accessCall(t, c, "POST", base+"/unpublish", nil, "4", 200)
	accessCall(t, c, "GET", "/api/v1/public/status-pages/new-health", nil, "", 404)
	accessCall(t, c, "DELETE", base, nil, "5", 200)
	accessCall(t, c, "PUT", base, d, "0", 409)
	accessCall(t, c, "GET", "/api/v1/projects/example/status-pages/vanity-one", nil, "", 404)
	for _, bad := range []string{"-bad", "bad-", "a/b", "hello world", strings.Repeat("a", 64), strings.Repeat("a", 40)} {
		d.ID = "invalid-name"
		d.Slug = bad
		accessCall(t, c, "PUT", "/api/v1/projects/example/status-pages/invalid-name", d, "0", 422)
	}
	changes := []storage.Change{}
	for _, bucket := range []string{"projects", "applications/example", "environments/example", "suites/example", "tests/example/health"} {
		rows, err := c.api.Store.All(t.Context(), bucket, func() string {
			if bucket == "projects" {
				return ""
			}
			return "example"
		}())
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range rows {
			if bucket == "projects" && record.ID != "example" {
				continue
			}
			target := strings.Replace(bucket, "/example", "/other", 1)
			id := record.ID
			if bucket == "projects" {
				id = "other"
			}
			change, err := storage.Put(target, id, func() string {
				if bucket == "projects" {
					return ""
				}
				return "other"
			}(), 0, record.Data)
			if err != nil {
				t.Fatal(err)
			}
			changes = append(changes, change)
		}
	}
	if err := c.api.Store.CommitAll(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	foreign := binding
	foreign.ID = ""
	saved := accessCall(t, c, "POST", "/api/v1/projects/other/probes", foreign, "", 200)
	d.ID = "foreign-page"
	d.Slug = "CUSTOMER-HEALTH"
	d.Probes[0].Probe = saved["id"].(string)
	accessCall(t, c, "PUT", "/api/v1/projects/other/status-pages/foreign-page", d, "0", 409)
}

func TestStatusVanityURLConcurrentClaimsHaveOneWinner(t *testing.T) {
	c, _, binding := probeFixture(t)
	service := &statuspages.Service{Store: c.api.Store, Auth: c.api.Auth, Probes: &probes.Service{Store: c.api.Store, Auth: c.api.Auth, Catalog: c.api.Catalog, Runs: c.api.Runs, Schedules: c.api.Schedules}}
	gate := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-gate
			_, err := service.Save(t.Context(), "admin", "example", statuspages.Definition{ID: fmt.Sprintf("claim-%d", index), Slug: "shared-health", Title: "Health", Background: "#ffffff", Window: "24h", MaxAge: 900, Probes: []statuspages.Entry{{Probe: binding.ID, Title: "Website"}}}, 0)
			results <- err
		}(i)
	}
	close(gate)
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if errors.Is(err, statuspages.ErrSlugTaken) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatal(wins, conflicts)
	}
	rows, err := c.api.Store.All(t.Context(), statuspages.Bucket("example"), "example")
	if err != nil || len(rows) != 1 {
		t.Fatal("losing draft was committed", rows, err)
	}
}
