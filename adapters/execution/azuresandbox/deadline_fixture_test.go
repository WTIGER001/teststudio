package azuresandbox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
)

type timerCredential struct{ t *testing.T }

func (c timerCredential) GetToken(_ context.Context, request policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if len(request.Scopes) != 1 || request.Scopes[0] != "https://management.core.windows.net//.default" {
		c.t.Fatal("unexpected ARM audience", request.Scopes)
	}
	return azcore.AccessToken{Token: "timer-identity-canary", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type timerTransport struct {
	t            *testing.T
	config       DeadlineConfig
	spec         DeadlineSpec
	workflow     *l.Workflow
	puts         int
	gets         int
	deletes      int
	lostCreate   bool
	emptyCreate  bool
	lostDelete   bool
	redirect     bool
	rejectCreate bool
}

func (f *timerTransport) decorate(value l.Workflow) l.Workflow {
	value.ID = to.Ptr(f.config.resourceID(f.spec))
	value.Name = to.Ptr(f.config.name(f.spec))
	value.Type = to.Ptr("Microsoft.Logic/workflows")
	value.Properties.ProvisioningState = to.Ptr(l.WorkflowProvisioningStateSucceeded)
	return value
}

func (f *timerTransport) Do(r *http.Request) (*http.Response, error) {
	f.t.Helper()
	if r.URL.Host != "management.azure.com" || r.URL.Path != f.config.resourceID(f.spec) || r.URL.RawQuery != "api-version=2019-05-01" || r.Header.Get("Authorization") != "Bearer timer-identity-canary" {
		f.t.Fatal("timer request escaped its selected ARM resource")
	}
	if f.redirect {
		result := response(302, "timer-identity-canary")
		result.Header.Set("Location", "https://unrelated.example/collect")
		return result, nil
	}
	switch r.Method {
	case "GET":
		f.gets++
		if f.workflow == nil {
			return response(404, `{"error":{"code":"ResourceNotFound","message":"timer not found"}}`), nil
		}
		return jsonResponse(f.workflow), nil
	case "PUT":
		f.puts++
		if f.rejectCreate {
			return response(403, `{"error":{"code":"AuthorizationFailed","message":"private-timer-canary"}}`), nil
		}
		data, err := io.ReadAll(r.Body)
		if err != nil || strings.Contains(string(data), "TESTSTUDIO_JOB_TOKEN") || strings.Contains(string(data), secretTicket) {
			f.t.Fatal("timer body contains runtime credentials", err)
		}
		var value l.Workflow
		if json.Unmarshal(data, &value) != nil {
			f.t.Fatal("invalid workflow request")
		}
		value = f.decorate(value)
		if !f.emptyCreate {
			f.workflow = &value
		}
		if f.lostCreate || f.emptyCreate {
			return response(503, `{"error":{"code":"Unavailable","message":"private-timer-canary"}}`), nil
		}
		return jsonResponse(value), nil
	case "DELETE":
		f.deletes++
		f.workflow = nil
		if f.lostDelete {
			f.lostDelete = false
			return response(503, `{"error":{"code":"Unavailable","message":"private-timer-canary"}}`), nil
		}
		return response(204, ""), nil
	}
	f.t.Fatal("unexpected timer mutation")
	return nil, nil
}

type timerFixture struct {
	*controllerFixture
	config    DeadlineConfig
	spec      DeadlineSpec
	transport *timerTransport
	timer     *SDKDeadlines
}

func newTimerFixture(t *testing.T, runner, operation string) *timerFixture {
	base := newControllerFixture(t, runner, operation)
	config := DeadlineConfig{Worker: base.plan.config, ResourceGroup: "studio-timers", NamePrefix: "studio", StopIdentity: "/subscriptions/" + base.plan.config.Subscription + "/resourceGroups/studio-identities/providers/Microsoft.ManagedIdentity/userAssignedIdentities/studio-stop"}
	spec := DeadlineSpec{Config: base.plan.config, Proof: base.plan.Proof()}
	f := &timerFixture{controllerFixture: base, config: config, spec: spec, transport: &timerTransport{t: t, config: config, spec: spec}}
	f.controls.ready = true
	f.bindTimer()
	t.Cleanup(func() { f.timer.Close() })
	return f
}

func (f *timerFixture) bindTimer() {
	f.t.Helper()
	var err error
	f.timer, err = NewSDKDeadlines(DeadlineOptions{Config: f.config, Exchange: f.exchange, Instances: f.api, Credential: timerCredential{t: f.t}, Transport: f.transport})
	if err != nil {
		f.t.Fatal(err)
	}
	f.controller.deadlines = f.timer
}

func (f *timerFixture) reopenTimer() {
	f.timer.Close()
	f.reopen()
	f.bindTimer()
}
