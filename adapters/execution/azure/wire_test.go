package azureworker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

type fixtureCredential struct{}

func (fixtureCredential) GetToken(_ context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if len(options.Scopes) != 1 || options.Scopes[0] != "https://management.core.windows.net//.default" {
		return azcore.AccessToken{}, errors.New("unexpected token scope")
	}
	return azcore.AccessToken{Token: "local-credential-canary", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type wireFixture struct {
	t     *testing.T
	plan  *Plan
	stage int
}

func (f *wireFixture) Do(request *http.Request) (*http.Response, error) {
	f.t.Helper()
	if request.URL.Scheme != "https" || request.URL.Host != "management.azure.com" || request.URL.Query().Get("api-version") != "2025-07-01" || request.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Fatal("SDK request escaped the fixed management endpoint, version, or credential")
	}
	f.stage++
	status, result := http.StatusOK, any(nil)
	switch f.stage {
	case 1:
		if request.Method != http.MethodPut || request.URL.Path != f.plan.ResourceID() {
			f.t.Fatal("job creation widened its resource scope")
		}
		var body a.Job
		if json.NewDecoder(request.Body).Decode(&body) != nil || fingerprint(document(body)) != fingerprint(document(f.plan.Definition())) {
			f.t.Fatal("SDK creation changed the immutable job")
		}
		status, result = http.StatusCreated, definition(f.plan)
	case 2:
		if request.Method != http.MethodPost || request.URL.Path != f.plan.ResourceID()+"/start" {
			f.t.Fatal("execution start widened its resource scope")
		}
		var body a.JobExecutionTemplate
		if json.NewDecoder(request.Body).Decode(&body) != nil {
			f.t.Fatal("SDK start template is invalid")
		}
		receipt := executionReceipt(f.plan)
		actual := a.JobExecution{ID: receipt.ID, Name: receipt.Name, Properties: &a.JobExecutionProperties{Template: &body, Status: to.Ptr(a.JobExecutionRunningStateProcessing)}}
		if _, err := f.plan.VerifyExecution(receipt, actual); err != nil {
			f.t.Fatal("SDK start changed its execution contract", err)
		}
		result = receipt
	case 3:
		receipt := executionReceipt(f.plan)
		if request.Method != http.MethodPost || request.URL.Path != value(receipt.ID)+"/stop" {
			f.t.Fatal("SDK cancellation stopped a different execution")
		}
		result = a.JobExecutionNamesCollection{Value: []*a.JobExecutionBase{&receipt}}
	default:
		f.t.Fatal("SDK repeated a mutation or made an unexpected call")
	}
	body, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), ContentLength: int64(len(body)), Request: request}, nil
}

func TestAzureJobSDKSerializesScopedCreateStartAndExactCancellationLocally(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"/"+operation, func(t *testing.T) {
				c, _, p, now := fixture(t, runner, operation)
				wire := &wireFixture{t: t, plan: p}
				client, err := a.NewJobsClient(c.Subscription, fixtureCredential{}, &arm.ClientOptions{ClientOptions: policy.ClientOptions{Transport: wire, Retry: policy.RetryOptions{MaxRetries: -1}}})
				if err != nil {
					t.Fatal(err)
				}
				creation, err := client.BeginCreateOrUpdate(t.Context(), c.ResourceGroup, p.Name(), p.Definition(), nil)
				if err != nil || !creation.Done() {
					t.Fatal("local creation did not complete", err)
				}
				created, err := creation.Result(t.Context())
				if err != nil || p.VerifyDefinition(created.Job) != nil {
					t.Fatal("SDK could not read its owned definition", err)
				}
				options, err := p.Start(now)
				if err != nil {
					t.Fatal(err)
				}
				start, err := client.BeginStart(t.Context(), c.ResourceGroup, p.Name(), options)
				if err != nil || !start.Done() {
					t.Fatal("local start did not complete", err)
				}
				started, err := start.Result(t.Context())
				if err != nil || p.VerifyReceipt(started.JobExecutionBase) != nil {
					t.Fatal("SDK could not read its execution receipt", err)
				}
				stop, err := client.BeginStopExecution(t.Context(), c.ResourceGroup, p.Name(), value(started.Name), nil)
				if err != nil || !stop.Done() {
					t.Fatal("local stop did not complete", err)
				}
				if _, err := stop.Result(t.Context()); err != nil || wire.stage != 3 {
					t.Fatal("SDK duplicated lifecycle mutations", err)
				}
			})
		}
	}
}
