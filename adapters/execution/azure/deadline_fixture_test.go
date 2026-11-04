package azureworker

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
)

type deadlineWire struct {
	t                       *testing.T
	mu                      sync.Mutex
	config                  DeadlineConfig
	spec                    DeadlineSpec
	resource                *l.Workflow
	creates, deletes, reads int
	mode                    string
	hidden                  bool
}

func deadlineResponse(r *http.Request, status int, data any) *http.Response {
	body, _ := json.Marshal(data)
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), ContentLength: int64(len(body)), Request: r}
}

func (f *deadlineWire) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Path != f.config.resourceID(f.spec) || r.URL.RawQuery != "api-version=2019-05-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("deadline SDK escaped its namespace or explicit identity")
	}
	switch r.Method {
	case "PUT":
		f.creates++
		var body l.Workflow
		if json.NewDecoder(r.Body).Decode(&body) != nil || fingerprint(document(body)) != fingerprint(document(f.config.definition(f.spec))) {
			f.t.Error("deadline mutation changed the fixed timer")
		}
		if f.mode == "create-rejected" {
			return deadlineResponse(r, 403, map[string]any{"error": map[string]any{"code": "Forbidden", "message": "private-provider-canary"}}), nil
		}
		body.ID, body.Name, body.Type = to.Ptr(f.config.resourceID(f.spec)), to.Ptr(f.config.name(f.spec)), to.Ptr("Microsoft.Logic/workflows")
		body.Properties.ProvisioningState = to.Ptr(l.WorkflowProvisioningStateSucceeded)
		f.resource = &body
		if f.mode == "provisioning" {
			body.Properties.ProvisioningState = to.Ptr(l.WorkflowProvisioningStateInProgress)
		}
		if f.mode == "lost-create" {
			return nil, errors.New("private-provider-canary")
		}
		return deadlineResponse(r, 201, body), nil
	case "GET":
		f.reads++
		if f.resource == nil || f.hidden {
			return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
		}
		return deadlineResponse(r, 200, f.resource), nil
	case "DELETE":
		f.deletes++
		if f.mode != "delete-pending" {
			f.resource = nil
		}
		if f.mode == "lost-delete" {
			return nil, errors.New("private-provider-canary")
		}
		return deadlineResponse(r, 200, nil), nil
	}
	f.t.Error("deadline SDK attempted an unsupported action")
	return deadlineResponse(r, 500, nil), nil
}

func attachDeadlines(t *testing.T, f *providerFixture, wire *deadlineWire) *SDKDeadlines {
	t.Helper()
	c := DeadlineConfig{Worker: f.config, ResourceGroup: "studio-deadlines(group)", StopIdentity: f.config.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "deadline-stop")}
	_, _, plan, err := f.provider.load(t.Context(), f.job.Reference)
	if err != nil {
		plan, err = NewPlan(f.config, f.job, f.now)
	}
	if err != nil {
		t.Fatal(err)
	}
	wire.t, wire.config, wire.spec = t, c, plan.deadlineSpec()
	client, err := NewSDKDeadlines(DeadlineOptions{Config: c, Store: f.store, Jobs: f.jobs, Credential: fixtureCredential{}, Transport: wire, Now: func() time.Time { return f.now }, DiscoveryCleanup: f.provider.discoveryCleanup})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	f.provider.deadlines = client
	return client
}

func prepareDeadline(t *testing.T, f *providerFixture, client *SDKDeadlines) DeadlineSpec {
	t.Helper()
	if err := f.provider.Ensure(t.Context(), f.job); err != nil {
		t.Fatal(err)
	}
	f.bindDefinition()
	if _, err := f.provider.update(t.Context(), f.job.Reference, true, func(d *dispatch) error { d.DefinitionSeen, d.DeadlineAttempted = true, true; return nil }); err != nil {
		t.Fatal(err)
	}
	_, _, plan, err := f.provider.load(t.Context(), f.job.Reference)
	if err != nil {
		t.Fatal(err)
	}
	return plan.deadlineSpec()
}

func requireDeadlineError(t *testing.T, err, errorType error) {
	t.Helper()
	if !errors.Is(err, errorType) || strings.Contains(err.Error(), "private-provider-canary") {
		t.Fatal("deadline failed without a bounded expected error", err)
	}
}
