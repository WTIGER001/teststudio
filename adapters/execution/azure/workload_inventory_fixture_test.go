package azureworker

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

type workloadInventoryFixture struct {
	t        *testing.T
	mu       sync.Mutex
	worker   Config
	settings ExchangeSettings
	apps     []*a.ContainerApp
	jobs     []*a.Job
	reads    int
	pages    map[string]int
	mode     string
}

func workloadInventoryTest(t *testing.T, relayName ...string) (Config, ExchangeSettings, *workloadInventoryFixture, *WorkloadInventoryGuard) {
	t.Helper()
	c, s, selected, guard := exchangeServiceTest(t, relayName...)
	f := &workloadInventoryFixture{t: t, worker: c, settings: s, pages: map[string]int{}}
	other := c.resourceID("Microsoft.App", "containerApps", "unrelated-app")
	f.apps = []*a.ContainerApp{to.Ptr(clone(selected.application)), {ID: to.Ptr(other), Name: to.Ptr("unrelated-app"), Type: to.Ptr("Microsoft.App/containerApps"), Properties: &a.ContainerAppProperties{ManagedEnvironmentID: to.Ptr(c.ControllerEnvironment), Configuration: &a.Configuration{Secrets: []*a.Secret{{Name: to.Ptr("unrelated"), Value: to.Ptr("private-list-config-canary")}}}}}}
	f.jobs = []*a.Job{{ID: to.Ptr(c.resourceID("Microsoft.App", "jobs", "unrelated-job")), Name: to.Ptr("unrelated-job"), Type: to.Ptr("Microsoft.App/jobs"), Properties: &a.JobProperties{EnvironmentID: to.Ptr(c.WorkerEnvironment), Configuration: &a.JobConfiguration{Secrets: []*a.Secret{{Name: to.Ptr("unrelated"), Value: to.Ptr("private-job-config-canary")}}}}}}
	reader, err := NewSDKWorkloadInventoryReader(c, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	result, err := NewWorkloadInventoryGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, result
}

func (f *workloadInventoryFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2025-07-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("workload inventory SDK widened authority")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	switch f.mode {
	case "missing":
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound", "message": "private-inventory-canary"}}), nil
	case "forbidden":
		return deadlineResponse(r, 403, map[string]any{"error": map[string]any{"code": "AuthorizationFailed", "message": "private-inventory-canary"}}), nil
	case "unavailable":
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-inventory-canary"}}), nil
	case "redirect":
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	var page map[string]any
	path := r.URL.Path
	switch path {
	case workloadInventoryPath(f.worker, "containerApps"):
		f.pages[path]++
		page = map[string]any{"value": f.apps}
		switch f.mode {
		case "paged", "cycle":
			if r.URL.Query().Get("$skipToken") == "" {
				page["value"] = f.apps[:1]
				page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-07-01&%24skipToken=one"
			} else {
				page["value"] = f.apps[1:]
				if f.mode == "cycle" {
					page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-07-01&%24skipToken=one"
				}
			}
		case "foreign-page":
			page["nextLink"] = "https://foreign.invalid/private?api-version=2025-07-01&%24skipToken=one"
		case "other-collection":
			page["nextLink"] = "https://management.azure.com" + workloadInventoryPath(f.worker, "jobs") + "?api-version=2025-07-01&%24skipToken=one"
		case "foreign-subscription":
			page["nextLink"] = "https://management.azure.com/subscriptions/99999999-2222-3333-4444-555555555555/providers/Microsoft.App/containerApps?api-version=2025-07-01&%24skipToken=one"
		case "endless":
			page["value"] = []any{}
			page["nextLink"] = "https://management.azure.com" + path + "?api-version=2025-07-01&%24skipToken=" + strconv.Itoa(f.pages[path])
		}
	case workloadInventoryPath(f.worker, "jobs"):
		f.pages[path]++
		page = map[string]any{"value": f.jobs}
		if f.mode == "missing-value" {
			delete(page, "value")
		}
	default:
		f.t.Error("unowned inventory resource requested")
		return deadlineResponse(r, 404, nil), nil
	}
	response := deadlineResponse(r, 200, page)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}
