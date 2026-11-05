package azuresandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxClientUsesPinnedAPIForImageAndExecutionLifecycle(t *testing.T) {
	p := fixturePlan(t, "go", "")
	c := p.config
	ctx := context.Background()
	identity := "/subscriptions/" + c.Subscription + "/resourceGroups/studio-identities/providers/Microsoft.ManagedIdentity/userAssignedIdentities/studio-pull"
	image := map[string]any{"id": p.proof.DiskImage, "image": map[string]any{"base": p.proof.Image, "entrypoint": []string{"/runner/teststudio-job"}, "cmd": []string{}}, "status": map[string]string{"state": "Ready"}}
	instance := instanceBody(t, p, "sandbox-123")
	requests := []string{}
	cred := &credential{t: t}
	client, err := NewClient(c, cred, roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Scheme != "https" || r.URL.Host != "management.westus2.azuredevcompute.io" || r.URL.Query().Get("api-version") != "2026-02-01-preview" || r.Header.Get("Authorization") != "Bearer "+identityToken || !strings.HasPrefix(r.URL.EscapedPath(), c.path()+"/") {
			t.Fatal("request escaped selected sandbox scope")
		}
		path := strings.TrimPrefix(r.URL.EscapedPath(), c.path())
		requests = append(requests, r.Method+" "+path)
		switch r.Method + " " + path {
		case "PUT /diskimages":
			body := readBody(t, r)
			if body["managedIdentityResourceId"] != identity || body["registryCredentials"] != nil || body["image"].(map[string]any)["base"] != p.proof.Image {
				t.Fatal("image import used a different source or credentials")
			}
			return jsonResponse(image), nil
		case "GET /diskimages/image-123":
			return jsonResponse(image), nil
		case "GET /diskimages":
			return jsonResponse([]any{image}), nil
		case "PUT /sandboxes":
			body := readBody(t, r)
			if body["environment"].(map[string]any)["TESTSTUDIO_JOB_TOKEN"] != secretTicket || body["managedIdentityResourceId"] != nil || body["registryCredentials"] != nil {
				t.Fatal("execution did not use only its bounded ticket")
			}
			body["id"] = "sandbox-123"
			body["state"] = "Running"
			return jsonResponse(body), nil
		case "GET /sandboxes/sandbox-123":
			return jsonResponse(instance), nil
		case "GET /sandboxes":
			return jsonResponse(map[string]any{"value": []any{instance}}), nil
		case "POST /sandboxes/sandbox-123/stop", "DELETE /sandboxes/sandbox-123":
			return response(202, ""), nil
		default:
			t.Fatal("unexpected operation", r.Method, path)
			return nil, hostedexecution.ErrUnavailable
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.ImportImage(ctx, p.proof.Image, identity); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReadImage(ctx, p.proof.DiskImage); err != nil {
		t.Fatal(err)
	}
	if images, err := client.Images(ctx); err != nil || len(images) != 1 || images[0].Base != p.proof.Image {
		t.Fatal("image reconciliation inventory failed", err)
	}
	created, err := client.Create(ctx, p)
	if err != nil || created.ID != "sandbox-123" {
		t.Fatal("sandbox creation failed", err)
	}
	read, err := client.Read(ctx, created.ID)
	if err != nil || p.Verify(read) != nil {
		t.Fatal("sandbox observation failed", err)
	}
	items, err := client.Inventory(ctx)
	if err != nil || len(items) != 1 || items[0].ID != created.ID {
		t.Fatal("inventory failed", err)
	}
	if err := client.Stop(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := client.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 8 || cred.calls != 8 {
		t.Fatal("unexpected implicit requests or retries")
	}
}

func TestSandboxClientPreservesAmbiguousMutationsAndScrubsErrors(t *testing.T) {
	for _, status := range []int{302, 400, 401, 403, 404, 409, 429, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			p := fixturePlan(t, "go", "")
			calls := 0
			client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				resp := response(status, secretTicket+identityToken)
				resp.Header.Set("Location", "https://unrelated.example/steal")
				return resp, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			_, err = client.Create(context.Background(), p)
			want := hostedexecution.ErrUnavailable
			if status == 400 || status == 401 || status == 403 || status == 404 {
				want = hostedexecution.ErrRejected
			}
			if !errors.Is(err, want) || calls != 1 || strings.Contains(err.Error(), secretTicket) || strings.Contains(err.Error(), identityToken) {
				t.Fatal("uncertain creation was retried, reclassified, or exposed diagnostics")
			}
		})
	}
	t.Run("lost acknowledgment", func(t *testing.T) {
		p := fixturePlan(t, "go", "")
		calls := 0
		client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("lost response " + secretTicket + identityToken)
		}))
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()
		if _, err := client.Create(context.Background(), p); err != hostedexecution.ErrUnavailable || calls != 1 {
			t.Fatal("lost acknowledgment did not remain uncertain")
		}
	})
}

func TestSandboxInventoryRejectsForeignIncompleteAndRepeatedPages(t *testing.T) {
	p := fixturePlan(t, "go", "")
	for name, next := range map[string]string{
		"foreign-host":  "https://other.example" + p.config.path() + "/sandboxes?api-version=" + APIVersion,
		"foreign-group": p.config.endpoint() + "/subscriptions/other/resourceGroups/other/sandboxGroups/other/sandboxes?api-version=" + APIVersion,
		"changed-api":   p.config.endpoint() + p.config.path() + "/sandboxes?api-version=other",
		"filtered":      p.config.endpoint() + p.config.path() + "/sandboxes?api-version=" + APIVersion + "&labels=owner%3Dother",
		"cycle":         p.config.endpoint() + p.config.path() + "/sandboxes?api-version=" + APIVersion,
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			cred := &credential{t: t}
			client, err := NewClient(p.config, cred, roundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(map[string]any{"value": []any{instanceBody(t, p, "sandbox-123")}, "nextLink": next}), nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			items, err := client.Inventory(context.Background())
			if err != hostedexecution.ErrUnavailable || items != nil || calls != 1 || cred.calls != 1 {
				t.Fatal("partial or escaped inventory accepted")
			}
		})
	}
	t.Run("complete pagination", func(t *testing.T) {
		calls := 0
		client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return jsonResponse(map[string]any{"value": []any{instanceBody(t, p, "sandbox-first")}, "nextLink": p.config.endpoint() + p.config.path() + "/sandboxes?api-version=" + APIVersion + "&$skiptoken=next"}), nil
			}
			if r.URL.Query().Get("$skiptoken") != "next" {
				t.Fatal("continuation token lost")
			}
			return jsonResponse([]any{instanceBody(t, p, "sandbox-second")}), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()
		items, err := client.Inventory(context.Background())
		if err != nil || len(items) != 2 || calls != 2 {
			t.Fatal("complete inventory failed", err)
		}
	})
}

func TestSandboxClientRejectsInvalidReferencesBeforeAcquiringCredentials(t *testing.T) {
	p := fixturePlan(t, "go", "")
	cred := &credential{t: t}
	client, err := NewClient(p.config, cred, roundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid request reached transport")
		return nil, io.EOF
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()
	for _, id := range []string{"../other", "x/stop", "x?api-version=other", "%2fother", ""} {
		if _, err := client.Read(ctx, id); err != hostedexecution.ErrOwnership {
			t.Fatal("invalid read accepted")
		}
		if err := client.Stop(ctx, id); err != hostedexecution.ErrOwnership {
			t.Fatal("invalid stop accepted")
		}
		if err := client.Delete(ctx, id); err != hostedexecution.ErrOwnership {
			t.Fatal("invalid delete accepted")
		}
	}
	restored, err := Restore(p.config, p.Proof())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Create(ctx, restored); err != hostedexecution.ErrRejected {
		t.Fatal("credential-free recovery proof can dispatch")
	}
	if _, err := client.ImportImage(ctx, "public.example/image:latest", "identity"); err != hostedexecution.ErrConfiguration {
		t.Fatal("unapproved image import accepted")
	}
	if cred.calls != 0 {
		t.Fatal("invalid operations acquired installation credentials")
	}
	data, _ := json.Marshal(restored)
	if strings.Contains(string(data), secretTicket) {
		t.Fatal("restored plan exposes a ticket")
	}
}
