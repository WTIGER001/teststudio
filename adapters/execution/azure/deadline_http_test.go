package azureworker

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureDeadlineHTTPRejectsScopeRedirectsAndUnknownMutationRetries(t *testing.T) {
	worker, _, plan, _ := fixture(t, "go", "")
	c := DeadlineConfig{Worker: worker, ResourceGroup: "deadline(group)", StopIdentity: worker.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "stop-deadline")}
	s := plan.deadlineSpec()
	for _, mode := range []string{"foreign-host", "foreign-job", "wrong-version", "extra-query", "child-operation", "wrong-method", "escaped-path", "redirect", "oversized", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			wire := transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				response := deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-provider-canary"}})
				if mode == "redirect" {
					response.StatusCode = 302
					response.Header.Set("Location", "https://foreign.invalid")
				}
				if mode == "oversized" {
					response.ContentLength = 2 << 20
				}
				return response, nil
			})
			h := newDeadlineHTTP(c, wire)
			path := strings.Replace(c.resourceID(s), "/resourceGroups/"+c.ResourceGroup+"/", "/resourceGroups/"+url.PathEscape(c.ResourceGroup)+"/", 1)
			r, _ := http.NewRequestWithContext(context.Background(), "GET", "https://management.azure.com"+path+"?api-version=2019-05-01", nil)
			switch mode {
			case "foreign-host":
				r.URL.Host = "foreign.invalid"
			case "foreign-job":
				r.URL.Path += "foreign"
			case "wrong-version":
				r.URL.RawQuery = "api-version=2025-07-01"
			case "extra-query":
				r.URL.RawQuery += "&extra=true"
			case "child-operation":
				r.URL.Path += "/triggers/Deadline/run"
			case "wrong-method":
				r.Method = "POST"
			case "escaped-path":
				r.URL.RawPath = strings.Replace(r.URL.EscapedPath(), "/workflows/", "/workflows%2f", 1)
			}
			_, err := h.Do(r)
			if mode == "redirect" || mode == "oversized" {
				requireDeadlineError(t, err, hostedexecution.ErrUnavailable)
				if calls != 1 {
					t.Fatal("redirect or oversized response repeated")
				}
			} else if mode != "unavailable" {
				requireDeadlineError(t, err, hostedexecution.ErrOwnership)
				if calls != 0 {
					t.Fatal("foreign request reached transport")
				}
			}
			if mode == "unavailable" {
				f := providerTest(t, "go", "")
				wireFixture := &deadlineWire{}
				client := attachDeadlines(t, f, wireFixture)
				spec := prepareDeadline(t, f, client)
				client.http.base = wire
				requireDeadlineError(t, client.Ensure(t.Context(), spec), hostedexecution.ErrUnavailable)
				if calls != 2 {
					t.Fatal("SDK retried unavailable GET", calls)
				}
				calls = 0
				client.http.base = transportFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method == "GET" {
						return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
					}
					return wire.Do(r)
				})
				requireDeadlineError(t, client.Ensure(t.Context(), spec), hostedexecution.ErrUnavailable)
				if calls != 3 {
					t.Fatal("SDK retried an unknown PUT", calls)
				}
			}
		})
	}
}

func TestAzureDeadlineStopActionMatchesContainerAppsSDKRequest(t *testing.T) {
	worker, _, plan, _ := fixture(t, "playwright", "")
	c := DeadlineConfig{Worker: worker, ResourceGroup: "deadlines", StopIdentity: worker.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "stop-deadline")}
	spec := plan.deadlineSpec()
	def := c.definition(spec).Properties.Definition.(map[string]any)
	inputs := def["actions"].(map[string]any)["Stop"].(map[string]any)["inputs"].(map[string]any)
	expected, _ := url.Parse(inputs["uri"].(string))
	calls := 0
	wire := transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != inputs["method"] || r.URL.String() != expected.String() || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
			t.Fatal("timer stop action differs from the pinned SDK wire contract")
		}
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			if len(b) != 0 {
				t.Fatal("stop request gained executable input")
			}
		}
		return deadlineResponse(r, 200, map[string]any{"value": []any{}}), nil
	})
	options := deadlineClientOptions(newDeadlineHTTP(c, wire))
	options.Transport = wire
	options.Retry = policy.RetryOptions{MaxRetries: -1}
	client, err := a.NewJobsClient(worker.Subscription, fixtureCredential{}, options)
	if err != nil {
		t.Fatal(err)
	}
	poller, err := client.BeginStopMultipleExecutions(t.Context(), worker.ResourceGroup, plan.Name(), nil)
	if err != nil || !poller.Done() || calls != 1 {
		t.Fatal("SDK stop request failed", err)
	}
}
