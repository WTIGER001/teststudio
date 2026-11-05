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
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

const secretTicket = "abcdefghijklmnopqrstuvwxy0123456789ABCDEFGH"
const identityToken = "installation-identity-canary"

func fixtureConfig() Config {
	return Config{Subscription: "12345678-1234-1234-1234-123456789abc", ResourceGroup: "studio-workers", Region: "westus2", Group: "studio-sandboxes", Registry: "studiotest.azurecr.io", RepositoryPrefix: "studio/sources", ExchangeOrigin: "https://exchange.studio.example", VNetConnection: "worker-exchange", VMMType: "fixture-vmm", NetworkProfile: strings.Repeat("a", 64)}
}

func fixturePlan(t *testing.T, runner, operation string) *Plan {
	t.Helper()
	c := fixtureConfig()
	ref := execution.Reference{ID: "execution-123", Installation: "studio-local", Project: "catalog", Run: "run-123", Test: "check-website", Attempt: 1, Worker: "worker-123"}
	if operation == "discover" {
		ref.Installation = "preparation-studio"
		ref.Test = "catalog"
		ref = (&hostedexecution.DiscoveryInspector{Name: Location, Runner: runner}).Reference(ref)
	} else {
		ref = (&hostedexecution.Executor{Name: Location, Runner: runner}).Reference(ref)
	}
	image := c.Registry + "/" + c.RepositoryPrefix + "/catalog/source@sha256:" + strings.Repeat("b", 64)
	disk := DiskImage{ID: "image-123", Base: image, State: "Ready", Entrypoint: []string{"/runner/teststudio-job"}}
	inspected := hostedexecution.Image{Reference: image, Runner: runner, Runtime: "runtime-1", OS: "linux", Architecture: "amd64", Supervisor: "worker-exchange/v1", Discovery: "worker-exchange/v1", Project: ref.Project, SourceDigest: "sha256:" + strings.Repeat("c", 64), TaskStorage: hostedexecution.NonRootTaskStorage}
	job := hostedexecution.Job{Reference: ref, Image: image, Runner: runner, Operation: operation, ExchangeOrigin: c.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: ref.ID, Token: secretTicket, Deadline: time.Now().UTC().Add(time.Minute)}}
	p, err := NewPlan(c, job, disk, inspected, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

type credential struct {
	t     *testing.T
	calls int
	fail  error
	after func()
}

func (c *credential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.t.Helper()
	c.calls++
	if len(options.Scopes) != 1 || options.Scopes[0] != "https://dynamicsessions.io/.default" {
		c.t.Fatal("unexpected sandbox token scope")
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
		c.t.Fatal("unbounded credential request")
	}
	if c.after != nil {
		c.after()
	}
	return azcore.AccessToken{Token: identityToken, ExpiresOn: time.Now().Add(time.Hour)}, c.fail
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body))}
}

func jsonResponse(value any) *http.Response {
	body, _ := json.Marshal(value)
	return response(200, string(body))
}

func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func instanceBody(t *testing.T, p *Plan, id string) map[string]any {
	t.Helper()
	body, err := p.launch(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	body["id"] = id
	body["state"] = "Running"
	return normalize(body).(map[string]any)
}
