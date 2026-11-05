package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type networkFixture struct {
	blockWorkerObservation atomic.Bool
	base                   *fixture
	mu                     sync.Mutex
	objects                map[string]networkObject
	posts                  map[string]int
	lostGatewayAck         bool
	badGateway             bool
	job                    hostedexecution.Job
}

func newNetworkFixture(t *testing.T, prepare ...bool) *networkFixture {
	t.Helper()
	n := &networkFixture{objects: map[string]networkObject{}, posts: map[string]int{}}
	f := fresh(t, "playwright", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n.serve(w, r, next) })
	})
	n.base = f
	f.mode = "running"
	config := f.provider.config
	config.Gateway = &GatewayConfig{Image: "registry.example.test/teststudio/gateway@sha256:" + strings.Repeat("b", 64), ServiceAccount: "gateway", TLSSecret: "gateway-tls", ExchangeTrustSecret: "exchange-ca"}
	f.provider.Close()
	var err error
	f.provider, err = New(Options{Config: config, Store: f.store, Inspector: networkImageInspector{imageInspector{"playwright"}}, Transport: f.server.Client().Transport, Token: f.provider.token, GatewayAuthority: f.exchange})
	if err != nil {
		t.Fatal(err)
	}
	ref := *f.input.Reference
	ref.Profile = "playwright-kubernetes-environment-v1"
	ref.Gateway = "teststudio-gateway-" + strings.TrimPrefix(ref.Container, "teststudio-job-")
	*f.input.Reference = ref
	f.input.Test.Type = "browser"
	f.input.Browser = &browserpolicy.Target{Origin: "https://application.test"}
	f.input.Artifact.Browser = &browserpolicy.Spec{Endpoint: "api"}
	f.input.AwaitDispatch = true
	entry, row, err := (workerstate.Ledger{Store: f.store}).Get(t.Context(), ref.Run, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	entry.Reference = ref
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	var ticket workerexchange.Ticket
	if len(prepare) == 0 || prepare[0] {
		ticket, err = f.exchange.Prepare(t.Context(), ref, f.input, time.Now().Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
	}
	n.job = hostedexecution.Job{Reference: ref, Image: f.input.Artifact.Image, Runner: "playwright", ExchangeOrigin: config.ExchangeOrigin, Ticket: ticket}
	return n
}

func (n *networkFixture) serve(w http.ResponseWriter, r *http.Request, next http.Handler) {
	f := n.base
	if n.blockWorkerObservation.Load() && r.Method == "GET" && r.URL.Path == f.provider.podPath(n.job.Reference.Container) {
		<-r.Context().Done()
		return
	}
	if strings.HasSuffix(r.URL.Path, "/serviceaccounts/gateway") {
		json.NewEncoder(w).Encode(map[string]any{"metadata": metadata{Name: "gateway"}, "automountServiceAccountToken": false})
		return
	}
	if r.URL.Path == f.provider.networkPath("NetworkPolicy", "") && r.Method == "GET" {
		n.mu.Lock()
		defer n.mu.Unlock()
		items := []networkObject{{Metadata: metadata{Name: f.provider.config.PolicyName}, Spec: f.provider.policy()}}
		if r.URL.Query().Get("labelSelector") != "" {
			items = nil
		}
		for _, object := range n.objects {
			if object.Kind == "NetworkPolicy" {
				items = append(items, object)
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"items": items})
		return
	}
	if r.Method == "GET" && (r.URL.Path == f.provider.podsPath() || r.URL.Path == f.provider.networkPath("Service", "")) {
		n.mu.Lock()
		defer n.mu.Unlock()
		items := []networkObject{}
		for _, object := range n.objects {
			if r.URL.Path == f.provider.podsPath() && object.Kind == "Pod" || r.URL.Path == f.provider.networkPath("Service", "") && object.Kind == "Service" {
				items = append(items, object)
			}
		}
		if r.URL.Path == f.provider.podsPath() {
			f.mu.Lock()
			if f.value != nil {
				v := f.value
				items = append(items, networkObject{APIVersion: v.APIVersion, Kind: v.Kind, Metadata: v.Metadata, Spec: v.Spec})
			}
			f.mu.Unlock()
		}
		json.NewEncoder(w).Encode(map[string]any{"items": items})
		return
	}
	isGateway := r.URL.Path == f.provider.podPath(n.job.Reference.Gateway)
	isAuxiliary := strings.Contains(r.URL.Path, "/services") || strings.Contains(r.URL.Path, "/networkpolicies")
	var posted networkObject
	if r.Method == "POST" && r.URL.Path == f.provider.podsPath() {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &posted)
		if posted.Metadata.Name == n.job.Reference.Gateway {
			isGateway = true
		} else {
			r.Body = io.NopCloser(bytes.NewReader(data))
		}
	}
	if !isGateway && !isAuxiliary {
		next.ServeHTTP(w, r)
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer "+f.token || r.URL.Query().Get("timeout") != "4s" {
		w.WriteHeader(403)
		return
	}
	if r.Method == "POST" {
		if posted.Kind == "" && json.NewDecoder(r.Body).Decode(&posted) != nil {
			w.WriteHeader(400)
			return
		}
		key := f.provider.networkPath(posted.Kind, posted.Metadata.Name)
		n.posts[key]++
		if _, exists := n.objects[key]; exists {
			w.WriteHeader(409)
			return
		}
		posted.Metadata.UID = "owned-" + short(key)
		posted.Metadata.Version = "1"
		if posted.Kind == "Pod" {
			posted.Status = json.RawMessage(`{"phase":"Running","containerStatuses":[{"name":"gateway","ready":false,"restartCount":0,"state":{"running":{}}}]}`)
			if n.badGateway {
				containers := posted.Spec["containers"].([]any)
				containers[0].(map[string]any)["image"] = "foreign:mutable"
			}
		}
		if posted.Kind == "Service" {
			posted.Spec["clusterIPs"] = []any{"None"}
			posted.Spec["ipFamilies"] = []any{"IPv4"}
			posted.Spec["ipFamilyPolicy"] = "SingleStack"
			posted.Spec["sessionAffinity"] = "None"
			posted.Spec["internalTrafficPolicy"] = "Cluster"
		}
		n.objects[key] = posted
		if posted.Kind == "Pod" && n.lostGatewayAck {
			n.lostGatewayAck = false
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(posted)
		return
	}
	value, exists := n.objects[r.URL.Path]
	if !exists {
		w.WriteHeader(404)
		return
	}
	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(value)
	case "PATCH":
		var patch struct {
			Metadata metadata `json:"metadata"`
			Spec     struct {
				Seconds int `json:"activeDeadlineSeconds"`
			} `json:"spec"`
		}
		if json.NewDecoder(r.Body).Decode(&patch) != nil || patch.Metadata.UID != value.Metadata.UID || patch.Metadata.Version != value.Metadata.Version || patch.Spec.Seconds != 1 {
			f.t.Error("gateway stop lacks UID, revision, or deadline guard")
			w.WriteHeader(409)
			return
		}
		value.Spec["activeDeadlineSeconds"] = 1
		value.Metadata.Version = "2"
		value.Status = json.RawMessage(`{"phase":"Failed","containerStatuses":[{"name":"gateway","ready":false,"restartCount":0,"state":{"terminated":{"exitCode":137}}}]}`)
		n.objects[r.URL.Path] = value
		json.NewEncoder(w).Encode(value)
	case "DELETE":
		var options struct {
			Preconditions struct{ UID, ResourceVersion string } `json:"preconditions"`
		}
		if json.NewDecoder(r.Body).Decode(&options) != nil || options.Preconditions.UID != value.Metadata.UID || options.Preconditions.ResourceVersion != value.Metadata.Version {
			f.t.Error("resource deletion lacks UID and revision guard")
			w.WriteHeader(409)
			return
		}
		if value.Kind == "Pod" && !bytes.Contains(value.Status, []byte(`"phase":"Failed"`)) {
			f.t.Error("active gateway deletion")
			w.WriteHeader(409)
			return
		}
		delete(n.objects, r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{"status": "Success"})
	default:
		w.WriteHeader(405)
	}
}

func (n *networkFixture) initialize(t *testing.T) {
	t.Helper()
	n.mu.Lock()
	key := n.base.provider.podPath(n.job.Reference.Gateway)
	value := n.objects[key]
	data, _ := json.Marshal(value.Spec["containers"])
	var containers []struct {
		Env []struct{ Name, Value string }
	}
	json.Unmarshal(data, &containers)
	token := ""
	for _, env := range containers[0].Env {
		if env.Name == "TESTSTUDIO_GATEWAY_TOKEN" {
			token = env.Value
		}
	}
	n.mu.Unlock()
	if _, err := n.base.exchange.GatewayInput(t.Context(), n.job.Reference.ID, token, value.Metadata.UID); err != nil {
		t.Fatal("private gateway initialization failed", err)
	}
	n.mu.Lock()
	value.Status = json.RawMessage(`{"phase":"Running","containerStatuses":[{"name":"gateway","ready":true,"restartCount":0,"state":{"running":{}}}]}`)
	n.objects[key] = value
	n.mu.Unlock()
}

func (n *networkFixture) checkpoint(t *testing.T) {
	t.Helper()
	ref := n.job.Reference
	entry, row, err := (workerstate.Ledger{Store: n.base.store}).Get(t.Context(), ref.Run, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	entry.ResultSaved, entry.EvidenceComplete = true, true
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := n.base.store.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
}

func (n *networkFixture) assertNoRepeatedCreation(t *testing.T) {
	t.Helper()
	n.mu.Lock()
	defer n.mu.Unlock()
	for path, count := range n.posts {
		if count != 1 {
			t.Fatal("repeated resource creation", path, count)
		}
	}
	n.base.mu.Lock()
	defer n.base.mu.Unlock()
	if n.base.posts != 1 {
		t.Fatal(fmt.Sprintf("expected one runner, got %d", n.base.posts))
	}
}
