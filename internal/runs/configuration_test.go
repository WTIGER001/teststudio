package runs_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/storage"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type configurationSecrets struct {
	origin    string
	available atomic.Bool
}

func (p *configurationSecrets) References(context.Context, string, string) ([]secrets.Reference, error) {
	return []secrets.Reference{{ID: "reader", Origins: []string{p.origin}}}, nil
}
func (p *configurationSecrets) Resolve(_ context.Context, project, id, purpose string) (secrets.Value, error) {
	if project != "project" || id != "reader" || purpose != secrets.APICredential || !p.available.Load() {
		return secrets.Value{}, errors.New("private provider detail")
	}
	return secrets.Value{Bytes: []byte("runtime-only-value")}, nil
}

func TestEnvironmentConfigurationGatesSelectedDependenciesAndFrozenParameters(t *testing.T) {
	s, p, _ := setup(t, nil)
	fallback := "3"
	requirements := []projects.Requirement{{Name: "tenantId", Kind: "parameter", ValueType: "integer"}, {Name: "retryCount", Kind: "parameter", ValueType: "integer", Default: &fallback}}
	if err := p.SaveRequirements(t.Context(), "admin", "project", "app", requirements, 1); err != nil {
		t.Fatal(err)
	}
	original, _ := currentTest(t, s, "HEALTH-1")
	dependent := original
	dependent.ID = "CONFIG-1"
	dependent.RequiredParameters = []string{"tenantId"}
	if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", dependent, 0); err != nil {
		t.Fatal(err)
	}
	root := original
	root.ID = "CONFIG-ROOT"
	root.Dependencies = []string{"CONFIG-1"}
	if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", root, 0); err != nil {
		t.Fatal(err)
	}
	publishAdmission(t, s, "configuration-selection")
	if _, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local", IDs: []string{"HEALTH-1"}}, "unrelated", "manual"); err != nil {
		t.Fatal("unrelated optional requirements block run", err)
	}
	request := runs.Request{Suite: "health", Environment: "local", IDs: []string{"CONFIG-ROOT"}}
	if _, err := s.Create(t.Context(), "admin", "project", request, "missing", "manual"); err == nil || !strings.Contains(err.Error(), "tenantId") {
		t.Fatal("missing dependency parameter admitted", err)
	}
	environment, revision := currentEnvironment(t, s)
	environment.Parameters = map[string]string{"tenantId": "bad"}
	if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(t.Context(), "admin", "project", request, "invalid", "manual"); err == nil {
		t.Fatal("invalid parameter admitted")
	}
	environment.Parameters["tenantId"] = "7"
	if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision+1); err != nil {
		t.Fatal(err)
	}
	queued, err := s.Create(t.Context(), "admin", "project", request, "configured", "manual")
	if err != nil || queued.Environment.Parameters["retryCount"] != "3" {
		t.Fatal("configured run or default failed", err)
	}
	environment.Parameters["tenantId"] = "8"
	if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision+2); err != nil {
		t.Fatal(err)
	}
	stop := start(t, s)
	defer stop()
	complete := finished(t, s, queued.ID)
	if complete.State != "blocked" || complete.Environment.Parameters["tenantId"] != "7" {
		t.Fatal("changed parameters execute or rewrite frozen values", complete.State)
	}
}

func TestEnvironmentConfigurationRechecksSecretAvailabilityWithoutPersistingValues(t *testing.T) {
	s, p, _ := setup(t, nil)
	environment, revision := currentEnvironment(t, s)
	provider := &configurationSecrets{origin: environment.Endpoints["api"]}
	provider.available.Store(true)
	p.Secrets, s.Secrets = provider, provider
	requirements := []projects.Requirement{{Name: "apiToken", Kind: "secret", Required: true}}
	if err := p.SaveRequirements(t.Context(), "admin", "project", "app", requirements, 1); err != nil {
		t.Fatal(err)
	}
	request := runs.Request{Suite: "health", Environment: "local", IDs: []string{"HEALTH-1"}}
	if _, err := s.Create(t.Context(), "admin", "project", request, "missing-reference", "manual"); err == nil {
		t.Fatal("missing secret admitted")
	}
	environment.Credentials = map[string]string{"apiToken": "reader"}
	if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision); err != nil {
		t.Fatal(err)
	}
	queued, err := s.Create(t.Context(), "admin", "project", request, "ready-reference", "manual")
	if err != nil {
		t.Fatal(err)
	}
	row, _ := s.Store.Get(t.Context(), "runs", queued.ID)
	if strings.Contains(string(row.Data), "runtime-only-value") {
		t.Fatal("secret persisted in run")
	}
	provider.available.Store(false)
	if _, err := s.Create(t.Context(), "admin", "project", request, "unavailable-reference", "manual"); err == nil || strings.Contains(err.Error(), "private provider detail") {
		t.Fatal("unavailable secret admitted or private error exposed")
	}
	stop := start(t, s)
	defer stop()
	complete := finished(t, s, queued.ID)
	if complete.State != "blocked" || !strings.Contains(complete.Error, "apiToken") || strings.Contains(complete.Error, "private provider detail") {
		t.Fatal("queued secret availability was not rechecked", complete.State, complete.Error)
	}
}

func TestHTTPEnvironmentParametersResolveWithoutChangingDefinition(t *testing.T) {
	requests := make(chan [2]string, 1)
	s, p, _ := setup(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- [2]string{r.URL.EscapedPath(), r.URL.RawQuery}
		json.NewEncoder(w).Encode(map[string]string{"status": "up"})
	}))
	if err := p.SaveRequirements(t.Context(), "admin", "project", "app", []projects.Requirement{{Name: "tenantId", Kind: "parameter"}}, 1); err != nil {
		t.Fatal(err)
	}
	test, revision := currentTest(t, s, "HEALTH-1")
	test.RequiredParameters = []string{"tenantId"}
	test.Check.Path = "/tenants/{{tenantId}}?tenant={{tenantId}}"
	if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", test, revision); err != nil {
		t.Fatal(err)
	}
	publishAdmission(t, s, "parameterized-http")
	environment, rev := currentEnvironment(t, s)
	environment.Parameters = map[string]string{"tenantId": "shop / north"}
	if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, rev); err != nil {
		t.Fatal(err)
	}
	run, err := s.Create(t.Context(), "admin", "project", runs.Request{Suite: "health", Environment: "local", IDs: []string{"HEALTH-1"}}, "parameterized", "manual")
	if err != nil {
		t.Fatal(err)
	}
	stop := start(t, s)
	defer stop()
	complete := finished(t, s, run.ID)
	var observed [2]string
	select {
	case observed = <-requests:
	case <-time.After(time.Second):
		t.Fatal("HTTP attempt did not reach its target")
	}
	if observed != [2]string{"/tenants/shop%20%2F%20north", "tenant=shop+%2F+north"} {
		t.Fatal("parameter escaping changed the request", observed)
	}
	if complete.Outcome != "passed" || complete.Tests[0].Check.Path != test.Check.Path {
		t.Fatal("parameterized execution changed definition or failed", complete.State, complete.Error)
	}
	row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "project", "health"), test.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := storage.Decode[catalog.Test](row)
	if current.Check.Path != test.Check.Path {
		t.Fatal("execution mutated working definition")
	}
}
