package azureworker

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureRegistryServiceSDKRejectsCredentialsMutationsAndForeignScope(t *testing.T) {
	c, s, f, g := registryServiceTest(t)
	reader := g.reader.(*SDKRegistryReader)
	if _, err := reader.Registry(t.Context(), s.Endpoints.ExchangeEnvironment); err != hostedexecution.ErrOwnership {
		t.Fatal("foreign service substituted for registry", err)
	}
	if _, err := reader.Connections(t.Context(), s.registry()+"-foreign"); err != hostedexecution.ErrOwnership {
		t.Fatal("foreign connections requested", err)
	}
	if _, err := reader.Links(t.Context(), s.links()); err != hostedexecution.ErrOwnership {
		t.Fatal("child substituted for registry", err)
	}
	if _, err := reader.Replicas(t.Context(), s.Endpoints.RegistryEndpoint); err != hostedexecution.ErrOwnership {
		t.Fatal("endpoint substituted for registry", err)
	}
	for name, mutate := range map[string]func(*http.Request){
		"credential-list":         func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.registry() + "/listCredentials" },
		"credential-regeneration": func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.registry() + "/regenerateCredential" },
		"approval-write":          func(r *http.Request) { r.Method = "PUT"; r.URL.Path = s.connections() + "/worker.approval" },
		"delete":                  func(r *http.Request) { r.Method = "DELETE" },
		"subscription-discovery": func(r *http.Request) {
			r.URL.Path = "/subscriptions/" + c.Subscription + "/providers/Microsoft.ContainerRegistry/registries"
		},
		"foreign-origin":     func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"different-registry": func(r *http.Request) { r.URL.Path = s.registry() + "-foreign/privateEndpointConnections" },
		"expand":             func(r *http.Request) { r.URL.RawQuery += "&%24expand=all" },
		"filter":             func(r *http.Request) { r.URL.RawQuery += "&%24filter=approved" },
		"wrong-version":      func(r *http.Request) { r.URL.RawQuery = "api-version=2023-07-01" },
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.registry()+"?api-version=2025-11-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if f.reads != 0 {
		t.Fatal("invalid registry request reached transport", f.reads)
	}
	if err := g.Check(t.Context(), c); err != nil || f.reads != 4 {
		t.Fatal("valid registry SDK reads failed", err, f.reads)
	}
}

func TestAzureRegistryServiceBindsPreparationEndpointAndPriorProfile(t *testing.T) {
	c, s, _, g := registryServiceTest(t)
	for name, mutate := range map[string]func(*RegistrySettings){
		"missing-preparation": func(s *RegistrySettings) { s.PreparationEndpoint = "" },
		"worker-path-reuse":   func(s *RegistrySettings) { s.PreparationEndpoint = s.Endpoints.RegistryEndpoint },
		"exchange-path-reuse": func(s *RegistrySettings) { s.PreparationEndpoint = s.Endpoints.ExchangeEndpoint },
		"foreign-subscription": func(s *RegistrySettings) {
			s.PreparationEndpoint = strings.Replace(s.PreparationEndpoint, c.Subscription, "99999999-2222-3333-4444-555555555555", 1)
		},
		"wrong-kind":       func(s *RegistrySettings) { s.PreparationEndpoint = s.Endpoints.RegistryNIC },
		"profile-mismatch": func(s *RegistrySettings) { s.Endpoints.RegistryEndpoint += "-different" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := s
			mutate(&changed)
			_, err := NewRegistryGuard(c, changed, g.endpoints, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	if _, err := NewRegistryGuard(c, s, nil, g.reader); err != hostedexecution.ErrConfiguration {
		t.Fatal("missing endpoint guard accepted", err)
	}
	if _, err := NewRegistryGuard(c, s, g.endpoints, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("missing registry reader accepted", err)
	}
	if _, err := NewSDKRegistryReader(c, s, nil, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("implicit credentials permitted", err)
	}
}
