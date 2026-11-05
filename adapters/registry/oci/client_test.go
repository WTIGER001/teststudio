package oci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type registryFixture struct {
	t                     *testing.T
	server                *httptest.Server
	client                *Client
	objects               map[string][]byte
	types                 map[string]string
	reference, configHash string
	mode                  string
	requests, tokens      int
	credential            Credential
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fresh(t *testing.T, runner string, index bool, mutate func(*configuration, *manifest)) *registryFixture {
	t.Helper()
	f := &registryFixture{t: t, objects: map[string][]byte{}, types: map[string]string{}, credential: Credential{Username: "registry-user", Password: "password-canary"}}
	config := configuration{OS: "linux", Architecture: "arm64"}
	config.Config.User = "1000:1000"
	config.Config.WorkingDir = "/suite"
	config.Config.Entrypoint = []string{"/runner/teststudio-go"}
	if runner == "playwright" {
		config.Config.Entrypoint = []string{"node", "/runner/run.mjs"}
	}
	config.Config.Labels = map[string]string{"teststudio.runtime": runner + "-fixture-v1", "teststudio.runtime-kind": "source-base/v1", "teststudio.job-supervisor": "worker-exchange/v1", "teststudio.work-item-bytes": "4194304", "teststudio.job-runtime-base": "sha256:" + strings.Repeat("b", 64), "teststudio.project": "project", "teststudio.installation": "installation", "teststudio.preparation": "preparation", "teststudio.source": "sha256:" + strings.Repeat("c", 64)}
	config.RootFS.Type = "layers"
	config.RootFS.DiffIDs = []string{"sha256:" + strings.Repeat("d", 64)}
	image := manifest{Schema: 2, MediaType: ociManifest, Config: descriptor{MediaType: "application/vnd.oci.image.config.v1+json"}, Layers: []descriptor{{MediaType: "application/vnd.oci.image.layer.v1.tar+gzip", Digest: "sha256:" + strings.Repeat("a", 64), Size: 100}}}
	if mutate != nil {
		mutate(&config, &image)
	}
	body, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	f.configHash = hash(body)
	image.Config.Digest = f.configHash
	image.Config.Size = int64(len(body))
	f.objects[f.configHash] = body
	f.types[f.configHash] = "application/octet-stream"
	body, err = json.Marshal(image)
	if err != nil {
		t.Fatal(err)
	}
	selected := hash(body)
	f.objects[selected] = body
	f.types[selected] = ociManifest
	if index {
		list := manifest{Schema: 2, MediaType: ociIndex, Manifests: []descriptor{{MediaType: ociManifest, Digest: selected, Size: int64(len(body)), Platform: &platform{OS: "linux", Architecture: "arm64", Variant: "v8"}}, {MediaType: ociManifest, Digest: "sha256:" + strings.Repeat("e", 64), Size: 100, Platform: &platform{OS: "linux", Architecture: "amd64"}}}}
		body, err = json.Marshal(list)
		if err != nil {
			t.Fatal(err)
		}
		selected = hash(body)
		f.objects[selected] = body
		f.types[selected] = ociIndex
	}
	f.server = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	var errClient error
	f.client, errClient = New(Options{Origin: f.server.URL, RepositoryPrefix: "teststudio", Architecture: "arm64", TokenURL: f.server.URL + "/token", TokenService: "registry-fixture", Credentials: func(context.Context) (Credential, error) { return f.credential, nil }, Transport: f.server.Client().Transport})
	if errClient != nil {
		t.Fatal(errClient)
	}
	t.Cleanup(f.client.Close)
	f.reference = strings.TrimPrefix(f.server.URL, "https://") + "/teststudio/project/suite@" + selected
	return f
}

func (f *registryFixture) serve(w http.ResponseWriter, r *http.Request) {
	f.requests++
	if r.Method != "GET" {
		f.t.Error("registry inspection mutated a resource", r.Method)
		w.WriteHeader(405)
		return
	}
	if r.URL.Path == "/token" {
		f.tokens++
		user, password, ok := r.BasicAuth()
		if !ok || user != f.credential.Username || password != f.credential.Password || r.URL.Query().Get("scope") != "repository:teststudio/project/suite:pull" || r.URL.Query().Get("service") != "registry-fixture" {
			f.t.Error("token exchange lost its exact repository scope")
			w.WriteHeader(403)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"token": "scoped-token-canary"})
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/v2/teststudio/project/suite/") {
		f.t.Error("unexpected repository request", r.URL.Path)
		w.WriteHeader(404)
		return
	}
	if f.mode == "manifest-redirect" {
		http.Redirect(w, r, "https://unapproved.example.test/leak", 307)
		return
	}
	if r.Header.Get("Authorization") != "Bearer scoped-token-canary" {
		realm := f.server.URL + "/token"
		if f.mode == "foreign-realm" {
			realm = "https://unapproved.example.test/token"
		}
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm=%q,service="registry-fixture",scope="repository:teststudio/project/suite:pull"`, realm))
		w.WriteHeader(401)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]
	body, found := f.objects[id]
	if !found {
		f.t.Error("inspector fetched layers or unknown objects", id)
		w.WriteHeader(404)
		return
	}
	if f.mode == "blob-redirect" && id == f.configHash {
		http.Redirect(w, r, f.server.URL+"/content?signature=signed-url-canary", 307)
		return
	}
	if f.mode == "wrong-bytes" {
		body = []byte(`{"error":"private response canary"}`)
	}
	if f.mode == "oversized" {
		w.Header().Set("Content-Length", fmt.Sprint(metadataLimit+1))
		w.WriteHeader(200)
		return
	}
	w.Header().Set("Content-Type", f.types[id])
	w.Header().Set("Docker-Content-Digest", id)
	w.Write(body)
}

func TestRegistryInspectsPinnedSingleImagesAndPlatformIndexes(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, index := range []bool{false, true} {
			t.Run(fmt.Sprint(runner, index), func(t *testing.T) {
				f := fresh(t, runner, index, func(c *configuration, _ *manifest) {
					if index {
						c.Config.Labels["teststudio.job-gateway"] = "tls-execution/v1"
					}
				})
				metadata, err := f.client.Inspect(t.Context(), f.reference)
				if err != nil || metadata.Image.Reference != f.reference || metadata.Image.Runner != runner || metadata.Image.Architecture != "arm64" || metadata.Project != "project" || metadata.Image.Project != metadata.Project || metadata.Image.SourceDigest != metadata.Source || metadata.Preparation != "preparation" || metadata.Configuration != f.configHash {
					t.Fatal(metadata, err)
				}
				wantGateway := ""
				if index {
					wantGateway = "tls-execution/v1"
				}
				if metadata.Image.GatewayTransport != wantGateway {
					t.Fatal("registry lost optional gateway transport capability")
				}
				if f.tokens != 1 {
					t.Fatal("repository token was fetched repeatedly", f.tokens)
				}
				want := 4
				if index {
					want = 5
				}
				if f.requests != want {
					t.Fatal("unnecessary registry requests", f.requests)
				}
				f.credential.Password = "rotated-password-canary"
				if _, err := f.client.Inspect(t.Context(), f.reference); err != nil || f.tokens != 2 {
					t.Fatal("credentials did not rotate", err)
				}
			})
		}
	}
}

func TestRegistryRejectsUnapprovedReferencesWithoutMakingRequests(t *testing.T) {
	f := fresh(t, "go", false, nil)
	for _, reference := range []string{"sha256:" + strings.Repeat("a", 64), strings.Split(f.reference, "@")[0] + ":latest", "other.example.test/teststudio/suite@sha256:" + strings.Repeat("a", 64), strings.Replace(f.reference, "/teststudio/", "/teststudio-other/", 1), strings.Replace(f.reference, "/project/", "/../", 1)} {
		if _, err := f.client.Inspect(t.Context(), reference); !errors.Is(err, ErrInvalid) {
			t.Fatal(reference, err)
		}
	}
	if f.requests != 0 {
		t.Fatal("unapproved reference reached the network")
	}
}

func TestRegistryRejectsDigestMismatchAndCredentialRedirects(t *testing.T) {
	for _, mode := range []string{"wrong-bytes", "foreign-realm", "manifest-redirect", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t, "go", false, nil)
			f.mode = mode
			_, err := f.client.Inspect(t.Context(), f.reference)
			if err == nil || strings.Contains(err.Error(), "canary") || strings.Contains(fmt.Sprintf("%v %#v", f.credential, f.credential), f.credential.Password) {
				t.Fatal("invalid registry response escaped", err)
			}
			if mode == "foreign-realm" && f.tokens != 0 {
				t.Fatal("unapproved token service was contacted")
			}
		})
	}
}

func TestRegistryRejectsAmbiguousPlatformIndexesAndWrongDescriptorSizes(t *testing.T) {
	for _, mode := range []string{"duplicate-platform", "wrong-size"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t, "go", true, nil)
			_, id, _ := strings.Cut(f.reference, "@")
			var value manifest
			if json.Unmarshal(f.objects[id], &value) != nil {
				t.Fatal("fixture manifest")
			}
			if mode == "duplicate-platform" {
				value.Manifests = append(value.Manifests, value.Manifests[0])
			} else {
				value.Manifests[0].Size++
			}
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			next := hash(data)
			f.objects[next] = data
			f.types[next] = ociIndex
			f.reference = strings.Split(f.reference, "@")[0] + "@" + next
			if _, err := f.client.Inspect(t.Context(), f.reference); !errors.Is(err, ErrInvalid) {
				t.Fatal("ambiguous or wrong-sized index was admitted", err)
			}
		})
	}
}

func TestRegistryRejectsUnsupportedOrUnsupervisedPreparedImages(t *testing.T) {
	mutations := []func(*configuration, *manifest){
		func(c *configuration, m *manifest) { c.Config.User = "0" },
		func(c *configuration, m *manifest) { c.Architecture = "amd64" },
		func(c *configuration, m *manifest) { delete(c.Config.Labels, "teststudio.job-supervisor") },
		func(c *configuration, m *manifest) { c.Config.OnBuild = []string{"RUN unwanted"} },
		func(c *configuration, m *manifest) { m.Layers[0].URLs = []string{"https://foreign.example.test/layer"} },
		func(c *configuration, m *manifest) { c.Config.Entrypoint = []string{"sh"} },
	}
	for i, mutation := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			f := fresh(t, "go", false, mutation)
			if _, err := f.client.Inspect(t.Context(), f.reference); !errors.Is(err, ErrInvalid) {
				t.Fatal("unsupported metadata was admitted", err)
			}
		})
	}
}

func TestRegistryBlobRedirectsRequireAnExplicitOriginAndStripCredentials(t *testing.T) {
	f := fresh(t, "go", false, nil)
	original := f.server.Config.Handler
	f.mode = "blob-redirect"
	contentRequests := 0
	f.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/content" {
			contentRequests++
			if r.Header.Get("Authorization") != "" {
				t.Error("registry credentials followed a blob redirect")
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(f.objects[f.configHash])
			return
		}
		original.ServeHTTP(w, r)
	})
	if _, err := f.client.Inspect(t.Context(), f.reference); !errors.Is(err, ErrDenied) || contentRequests != 0 {
		t.Fatal("unapproved blob destination was followed", err)
	}
	f.client.blobRedirectOrigins[f.server.URL] = true
	if _, err := f.client.Inspect(t.Context(), f.reference); err != nil || contentRequests != 1 {
		t.Fatal("approved checksummed blob redirect failed", err)
	}
}

func TestRegistryValidatesOptionalTaskVolumeCapability(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, mode := range []string{"valid", "missing-path", "extra-path", "null-volume", "unknown-capability", "legacy"} {
			t.Run(runner+"-"+mode, func(t *testing.T) {
				f := fresh(t, runner, false, func(c *configuration, _ *manifest) {
					if mode == "legacy" {
						return
					}
					c.Config.Labels["teststudio.task-storage"] = hostedexecution.NonRootTaskStorage
					c.Config.Volumes = map[string]json.RawMessage{"/tmp": json.RawMessage(`{ }`), "/output": json.RawMessage(`{}`)}
					switch mode {
					case "missing-path":
						delete(c.Config.Volumes, "/output")
					case "extra-path":
						c.Config.Volumes["/runner"] = json.RawMessage(`{}`)
					case "null-volume":
						c.Config.Volumes["/tmp"] = json.RawMessage(`null`)
					case "unknown-capability":
						c.Config.Labels["teststudio.task-storage"] = "unexpected"
					}
				})
				metadata, err := f.client.Inspect(t.Context(), f.reference)
				if mode == "valid" || mode == "legacy" {
					if err != nil {
						t.Fatal(err)
					}
					want := ""
					if mode == "valid" {
						want = hostedexecution.NonRootTaskStorage
					}
					if metadata.Image.TaskStorage != want {
						t.Fatal("capability lost")
					}
				} else if !errors.Is(err, ErrInvalid) {
					t.Fatal("unsupported task volume metadata admitted", err)
				}
			})
		}
	}
}
