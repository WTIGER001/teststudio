package azureworker

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestAzureSDKControlUsesScopedSingleMutationRequests(t *testing.T) {
	c, _, plan, now := fixture(t, "go", "")
	wire := &wireFixture{t: t, plan: plan}
	client, err := NewSDKClient(c, fixtureCredential{}, wire)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Create(t.Context(), plan.Name(), plan.Definition()); err != nil {
		t.Fatal(err)
	}
	start, _ := plan.Start(now)
	receipt, err := client.Start(t.Context(), plan.Name(), start)
	if err != nil || receipt == nil || plan.VerifyReceipt(*receipt) != nil {
		t.Fatal("SDK control lost its execution receipt", err)
	}
	if err := client.Stop(t.Context(), plan.Name(), value(receipt.Name)); err != nil || wire.stage != 3 {
		t.Fatal("SDK control repeated a mutation", err)
	}
}

func TestAzureSDKControlAcceptsEscapedOwnedResourceGroup(t *testing.T) {
	c, job, _, now := fixture(t, "go", "")
	c.ResourceGroup = "studio(group)"
	plan, err := NewPlan(c, job, now)
	if err != nil {
		t.Fatal(err)
	}
	wire := &wireFixture{t: t, plan: plan}
	client, err := NewSDKClient(c, fixtureCredential{}, wire)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Create(t.Context(), plan.Name(), plan.Definition()); err != nil {
		t.Fatal("SDK encoding rejected an owned resource group", err)
	}
}

func TestAzureSDKControlRejectsPaginationScopeAndDoesNotRetryUnknownStart(t *testing.T) {
	for _, mode := range []string{"foreign-job", "foreign-page", "unknown-start", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			c, _, plan, now := fixture(t, "playwright", "")
			calls := 0
			transport := transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				status := 200
				body := `{"value":[],"nextLink":"https://management.azure.com` + c.resourceID("Microsoft.App", "jobs", c.NamePrefix+"-"+strings.Repeat("f", 24)) + `/executions?api-version=2025-07-01"}`
				if mode == "unknown-start" {
					status = 503
					body = `{"error":{"code":"ServiceUnavailable","message":"private-provider-canary"}}`
				}
				if mode == "redirect" {
					status = 302
					body = ""
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}, "Location": []string{"https://outside.invalid/private"}}, Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: r}, nil
			})
			client, err := NewSDKClient(c, fixtureCredential{}, transport)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			switch mode {
			case "foreign-job":
				_, err = client.Get(t.Context(), "another-job")
			case "foreign-page":
				_, err = client.Executions(t.Context(), plan.Name())
			case "unknown-start":
				start, _ := plan.Start(now)
				_, err = client.Start(t.Context(), plan.Name(), start)
			case "redirect":
				_, err = client.Get(t.Context(), plan.Name())
			}
			if err == nil || strings.Contains(err.Error(), "canary") {
				t.Fatal("SDK scope or private diagnostics escaped", err)
			}
			if mode == "foreign-job" {
				if calls != 0 || !errors.Is(err, hostedexecution.ErrOwnership) {
					t.Fatal("foreign request reached transport", calls, err)
				}
			} else if calls != 1 {
				t.Fatal("SDK retried a mutation or followed an unowned location", calls)
			}
		})
	}
}
