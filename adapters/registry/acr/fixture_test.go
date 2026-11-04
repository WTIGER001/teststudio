package acr

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type fixtureClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *fixtureClock) now() time.Time          { c.mu.Lock(); defer c.mu.Unlock(); return c.at }
func (c *fixtureClock) advance(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.at = c.at.Add(d) }

type fixtureCredential struct {
	mu      sync.Mutex
	current azcore.AccessToken
	tenant  string
	calls   int
	fail    bool
}

func (c *fixtureCredential) GetToken(_ context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if len(options.Scopes) != 1 || options.Scopes[0] != "https://containerregistry.azure.net/.default" || options.TenantID != c.tenant || options.EnableCAE {
		return azcore.AccessToken{}, errors.New("unexpected registry credential scope")
	}
	if c.fail {
		return azcore.AccessToken{}, errors.New("credential-private-canary")
	}
	return c.current, nil
}
func (c *fixtureCredential) raw() string { c.mu.Lock(); defer c.mu.Unlock(); return c.current.Token }

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func jwt(claims any) string {
	body, _ := json.Marshal(claims)
	return "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(body) + ".c2lnbmF0dXJl"
}

type fixture struct {
	t                                                          *testing.T
	mu                                                         sync.Mutex
	clock                                                      *fixtureClock
	credential                                                 *fixtureCredential
	registry, tenant, repo, reference, configHash              string
	client                                                     *Client
	objects                                                    map[string][]byte
	types                                                      map[string]string
	tokens                                                     map[string]string
	exchanges, grants, metadata, writes, blobReads, unexpected int
	mode                                                       string
}

func fresh(t *testing.T, runner string, publish, base bool) *fixture {
	t.Helper()
	f := &fixture{t: t, clock: &fixtureClock{at: time.Now().UTC()}, registry: "studioregistry.azurecr.io", tenant: "11111111-2222-3333-4444-555555555555", repo: "teststudio/" + runner, objects: map[string][]byte{}, types: map[string]string{}, tokens: map[string]string{}}
	f.credential = &fixtureCredential{tenant: f.tenant, current: azcore.AccessToken{Token: "entra-private-canary", ExpiresOn: f.clock.now().Add(time.Hour)}}
	labels := map[string]string{"teststudio.runtime": runner + "-fixture-v1", "teststudio.runtime-kind": "source-base/v1", "teststudio.job-supervisor": "worker-exchange/v1", "teststudio.discovery-supervisor": "worker-exchange/v1", "teststudio.work-item-bytes": "4194304", "teststudio.job-runtime-base": hash([]byte("trusted-base"))}
	if !base {
		labels["teststudio.project"] = "parcel"
		labels["teststudio.installation"] = "studio"
		labels["teststudio.preparation"] = "source-preparation"
		labels["teststudio.source"] = hash([]byte("new-source"))
	}
	entry := []string{"/runner/teststudio-go"}
	if runner == "playwright" {
		entry = []string{"node", "/runner/run.mjs"}
		labels["teststudio.playwright-version"] = "1.64.1"
	}
	configuration, _ := json.Marshal(map[string]any{"os": "linux", "architecture": "amd64", "config": map[string]any{"User": "1000:1000", "WorkingDir": "/suite", "Entrypoint": entry, "Labels": labels}, "rootfs": map[string]any{"type": "layers", "diff_ids": []string{hash([]byte("base-layer"))}}})
	f.configHash = hash(configuration)
	f.objects[f.configHash] = configuration
	f.types[f.configHash] = "application/octet-stream"
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": map[string]any{"mediaType": "application/vnd.oci.image.config.v1+json", "digest": f.configHash, "size": len(configuration)}, "layers": []any{map[string]any{"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": hash([]byte("base-layer")), "size": len("base-layer")}}})
	id := hash(manifest)
	f.objects[id] = manifest
	f.types[id] = "application/vnd.oci.image.manifest.v1+json"
	f.reference = f.registry + "/" + f.repo + "@" + id
	var err error
	f.client, err = New(Options{Registry: f.registry, Tenant: f.tenant, RepositoryPrefix: "teststudio", Publish: publish, Credential: f.credential, Transport: f, Now: f.clock.now, BlobRedirectOrigins: []string{"https://studioblobs.blob.core.windows.net"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.client.Close)
	return f
}

func (f *fixture) response(r *http.Request, status int, kind string, body []byte, headers map[string]string) *http.Response {
	h := http.Header{"Content-Type": []string{kind}}
	for key, v := range headers {
		h.Set(key, v)
	}
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(string(body))), ContentLength: int64(len(body)), Request: r}
}

func (f *fixture) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Host == "studioblobs.blob.core.windows.net" {
		f.blobReads++
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			f.t.Fatal("credentials entered the approved blob request")
		}
		return f.response(r, 200, "application/octet-stream", f.objects[f.configHash], nil), nil
	}
	if r.URL.Scheme != "https" || r.URL.Host != f.registry {
		f.unexpected++
		return nil, errors.New("unexpected external endpoint")
	}
	if strings.HasPrefix(r.URL.Path, "/oauth2/") {
		if r.Method != "POST" || r.Header.Get("Authorization") != "" || r.ParseForm() != nil || r.Form.Get("service") != f.registry {
			f.t.Fatal("invalid OAuth request")
		}
		field := "access_token"
		scope := ""
		if r.URL.Path == "/oauth2/exchange" {
			f.exchanges++
			field = "refresh_token"
			if r.Form.Get("grant_type") != "access_token" || r.Form.Get("tenant") != f.tenant || r.Form.Get("access_token") != f.credential.raw() {
				f.t.Fatal("exchange did not bind the selected identity, tenant, and registry")
			}
		} else {
			f.grants++
			scope = r.Form.Get("scope")
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") == "" || !strings.HasPrefix(scope, "repository:teststudio/") {
				f.t.Fatal("repository grant widened scope")
			}
		}
		if f.mode == "exchange-redirect" && field == "refresh_token" || f.mode == "token-redirect" && field == "access_token" {
			return f.response(r, 302, "application/json", nil, map[string]string{"Location": "https://outside.invalid/private"}), nil
		}
		if f.mode == "duplicate-json" {
			return f.response(r, 200, "application/json", []byte(`{"refresh_token":"private-canary","refresh_token":"other"}`), nil), nil
		}
		if f.mode == "oversize" {
			return f.response(r, 200, "application/json", []byte(strings.Repeat("x", (128<<10)+1)), nil), nil
		}
		if f.mode == "denied" {
			return f.response(r, 403, "application/json", []byte(`{"error":"private-canary"}`), nil), nil
		}
		claims := map[string]any{"exp": f.clock.now().Add(time.Hour).Unix(), "nbf": f.clock.now().Add(-time.Minute).Unix(), "aud": f.registry, "scope": scope}
		if f.mode == "expired" {
			claims["exp"] = f.clock.now().Add(-time.Minute).Unix()
		}
		if f.mode == "missing-exp" {
			delete(claims, "exp")
		}
		if f.mode == "future-nbf" {
			claims["nbf"] = f.clock.now().Add(time.Hour).Unix()
		}
		token := jwt(claims)
		if f.mode == "invalid-token" {
			token += " bad"
		}
		if field == "access_token" {
			f.tokens[token] = scope
		}
		body, _ := json.Marshal(map[string]any{field: token, "expires_in": 3600})
		return f.response(r, 200, "application/json", body, nil), nil
	}
	f.metadata++
	marker := "/blobs/"
	kind := "blobs"
	at := strings.LastIndex(r.URL.Path, marker)
	if i := strings.LastIndex(r.URL.Path, "/manifests/"); i >= 0 {
		at = i
		marker = "/manifests/"
		kind = "manifests"
	}
	if at < 0 {
		return nil, errors.New("unexpected registry operation")
	}
	repo := strings.TrimPrefix(r.URL.Path[:at], "/v2/")
	tail := r.URL.Path[at+len(marker):]
	scope := "repository:" + repo + ":pull"
	write := r.Method != "GET" && r.Method != "HEAD"
	if write {
		scope += ",push"
	}
	authorized := f.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	if authorized != scope && authorized != "repository:"+repo+":pull,push" {
		realm := "https://" + f.registry + "/oauth2/token"
		if f.mode == "foreign-challenge" {
			realm = "https://outside.invalid/oauth2/token"
		}
		challenge := fmt.Sprintf(`Bearer realm=%q,service=%q,scope=%q`, realm, f.registry, scope)
		return f.response(r, 401, "application/json", nil, map[string]string{"WWW-Authenticate": challenge}), nil
	}
	if r.Method == "POST" && tail == "uploads/" {
		f.writes++
		return f.response(r, 202, "application/json", nil, map[string]string{"Location": "https://" + f.registry + "/v2/" + repo + "/blobs/uploads/session-1"}), nil
	}
	if r.Method == "DELETE" {
		f.writes++
		return f.response(r, 204, "application/json", nil, nil), nil
	}
	if r.Method == "PUT" {
		f.writes++
		body, _ := io.ReadAll(r.Body)
		id := hash(body)
		if kind == "blobs" && r.URL.Query().Get("digest") != id {
			f.t.Fatal("blob upload changed its digest")
		}
		if kind == "manifests" && tail != id {
			f.t.Fatal("manifest upload changed its digest")
		}
		f.objects[id] = body
		f.types[id] = "application/octet-stream"
		if kind == "manifests" {
			f.types[id] = "application/vnd.oci.image.manifest.v1+json"
		}
		return f.response(r, 201, "application/json", nil, map[string]string{"Docker-Content-Digest": id}), nil
	}
	body, ok := f.objects[tail]
	if !ok {
		return f.response(r, 404, "application/json", nil, nil), nil
	}
	if f.mode == "blob-redirect" && kind == "blobs" && r.Method == "GET" {
		return f.response(r, 307, "application/json", nil, map[string]string{"Location": "https://studioblobs.blob.core.windows.net/config?sv=private-signed-query"}), nil
	}
	response := f.response(r, 200, f.types[tail], body, map[string]string{"Docker-Content-Digest": tail})
	if r.Method == "HEAD" {
		response.Body = http.NoBody
	}
	return response, nil
}
