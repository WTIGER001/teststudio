package registryimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
)

type registry struct {
	mu               sync.Mutex
	server           *httptest.Server
	client           *oci.Client
	base             string
	objects          map[string][]byte
	types            map[string]string
	sessions         map[string]bool
	scopes           []string
	posts, manifests int
	mode, password   string
	cancel           context.CancelFunc
	baseConfig       map[string]any
}

const repositoryPath = "/v2/teststudio/source/"

func checksum(data []byte) string {
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func fixture(t *testing.T, runner string, alter func(map[string]any)) *registry {
	t.Helper()
	f := &registry{objects: map[string][]byte{}, types: map[string]string{}, sessions: map[string]bool{}, password: "private-password-canary"}
	entrypoint := []string{"/runner/teststudio-go"}
	if runner == "playwright" {
		entrypoint = []string{"node", "/runner/run.mjs"}
	}
	layer := []byte("owned protocol fixture base layer")
	base := map[string]any{"architecture": "arm64", "os": "linux", "rootfs": map[string]any{"type": "layers", "diff_ids": []string{checksum(layer)}}, "config": map[string]any{"User": "1000:1000", "WorkingDir": "/suite", "Entrypoint": entrypoint, "Env": []string{"GOPROXY=off", "TRUSTED_RUNTIME=preserved"}, "StopSignal": "SIGTERM", "Labels": map[string]string{"teststudio.runtime": runner + "-registry-fixture-v1", "teststudio.runtime-kind": "source-base/v1", "teststudio.work-item-bytes": "4194304", "teststudio.job-supervisor": "worker-exchange/v1", "teststudio.discovery-supervisor": "worker-exchange/v1", "teststudio.job-runtime-base": "sha256:" + strings.Repeat("b", 64), "teststudio.playwright-version": "1.64.1"}}}
	if alter != nil {
		alter(base)
	}
	f.baseConfig = base
	configuration, _ := json.Marshal(base)
	configHash, layerHash := checksum(configuration), checksum(layer)
	f.objects[configHash], f.types[configHash] = configuration, "application/octet-stream"
	f.objects[layerHash], f.types[layerHash] = layer, "application/octet-stream"
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": map[string]any{"mediaType": "application/vnd.oci.image.config.v1+json", "digest": configHash, "size": len(configuration)}, "layers": []any{map[string]any{"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": layerHash, "size": len(layer)}}})
	manifestHash := checksum(manifest)
	f.objects[manifestHash], f.types[manifestHash] = manifest, "application/vnd.oci.image.manifest.v1+json"
	f.server = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	var err error
	f.client, err = oci.New(oci.Options{Origin: f.server.URL, RepositoryPrefix: "teststudio", Architecture: "arm64", TokenURL: f.server.URL + "/token", TokenService: "source-registry", Credentials: func(context.Context) (oci.Credential, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		return oci.Credential{Username: "source-worker", Password: f.password}, nil
	}, Transport: f.server.Client().Transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.client.Close)
	f.base = strings.TrimPrefix(f.server.URL, "https://") + "/teststudio/source@" + manifestHash
	return f
}

func (f *registry) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/token" {
		username, password, valid := r.BasicAuth()
		scope := r.URL.Query().Get("scope")
		if !valid || username != "source-worker" || password != f.password || r.URL.Query().Get("service") != "source-registry" || scope != "repository:teststudio/source:pull" && scope != "repository:teststudio/source:pull,push" {
			w.WriteHeader(403)
			return
		}
		f.scopes = append(f.scopes, scope)
		json.NewEncoder(w).Encode(map[string]string{"token": scope})
		return
	}
	if !strings.HasPrefix(r.URL.Path, repositoryPath) {
		w.WriteHeader(404)
		return
	}
	write := r.Method != "GET"
	scope := "repository:teststudio/source:pull"
	if write {
		scope += ",push"
	}
	if r.Header.Get("Authorization") != "Bearer "+scope {
		challengeScope := scope
		if r.Method == "HEAD" {
			challengeScope = "repository:teststudio/source:pull"
		}
		if r.Method == "POST" {
			challengeScope = "repository:teststudio/source:push"
		}
		if write && f.mode == "extra-scope" {
			challengeScope += ",delete"
		}
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm=%q,service="source-registry",scope=%q`, f.server.URL+"/token", challengeScope))
		w.WriteHeader(401)
		return
	}
	kind := strings.TrimPrefix(r.URL.Path, repositoryPath)
	if kind == "blobs/uploads/" && r.Method == "POST" {
		f.posts++
		name := fmt.Sprintf("upload-%d", f.posts)
		f.sessions[name] = true
		if f.mode == "lost-opening" {
			connection, _, _ := w.(http.Hijacker).Hijack()
			connection.Close()
			return
		}
		location := repositoryPath + "blobs/uploads/" + name + "?_state=private-upload-canary"
		if f.mode == "foreign-upload" {
			location = "https://foreign.invalid/credential-canary"
		}
		w.Header().Set("Location", location)
		w.WriteHeader(202)
		return
	}
	if strings.HasPrefix(kind, "blobs/uploads/") {
		name := strings.TrimPrefix(kind, "blobs/uploads/")
		if !f.sessions[name] {
			w.WriteHeader(404)
			return
		}
		if r.URL.Query().Get("_state") != "private-upload-canary" {
			w.WriteHeader(400)
			return
		}
		if r.Method == "DELETE" {
			if f.mode == "cleanup-failure" {
				w.WriteHeader(503)
				return
			}
			delete(f.sessions, name)
			w.WriteHeader(204)
			return
		}
		if r.Method == "PUT" {
			if f.cancel != nil {
				f.cancel()
				w.WriteHeader(503)
				return
			}
			if f.mode == "cleanup-failure" {
				w.WriteHeader(503)
				return
			}
			data, err := io.ReadAll(io.LimitReader(r.Body, 145<<20))
			hash := checksum(data)
			if err != nil || r.URL.Query().Get("digest") != hash {
				w.WriteHeader(400)
				return
			}
			f.objects[hash], f.types[hash] = data, "application/octet-stream"
			if f.mode == "concurrent-blob" {
				w.WriteHeader(503)
				return
			}
			delete(f.sessions, name)
			if f.mode == "lost-blob-ack" {
				connection, _, _ := w.(http.Hijacker).Hijack()
				connection.Close()
				return
			}
			w.Header().Set("Docker-Content-Digest", hash)
			w.WriteHeader(201)
			return
		}
	}
	if strings.HasPrefix(kind, "manifests/") && r.Method == "PUT" {
		data, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		hash := checksum(data)
		if err != nil || strings.TrimPrefix(kind, "manifests/") != hash {
			w.WriteHeader(400)
			return
		}
		f.manifests++
		f.objects[hash], f.types[hash] = data, r.Header.Get("Content-Type")
		if f.mode == "lost-manifest-ack" {
			connection, _, _ := w.(http.Hijacker).Hijack()
			connection.Close()
			return
		}
		w.Header().Set("Docker-Content-Digest", hash)
		w.WriteHeader(201)
		return
	}
	if r.Method == "HEAD" || r.Method == "GET" {
		_, hash, _ := strings.Cut(kind, "/")
		data, found := f.objects[hash]
		if !found {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", f.types[hash])
		w.Header().Set("Docker-Content-Digest", hash)
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		if r.Method == "GET" {
			w.Write(data)
		}
		return
	}
	w.WriteHeader(405)
}
