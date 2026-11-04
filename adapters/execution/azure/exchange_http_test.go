package azureworker

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureExchangeBindsSettingsAndReturnsCredentialFreeDefinitions(t *testing.T) {
	c, s, _, g := exchangeServiceTest(t)
	for name, mutate := range map[string]func(*ExchangeSettings){
		"untrusted-suite-image": func(s *ExchangeSettings) {
			s.Image = c.Registry + "/" + c.RepositoryPrefix + "/worker-relay@sha256:" + strings.Repeat("a", 64)
		},
		"mutable-image": func(s *ExchangeSettings) { s.Image = c.Registry + "/services/worker-relay:latest" },
		"foreign-registry": func(s *ExchangeSettings) {
			s.Image = "foreign.invalid/services/relay@sha256:" + strings.Repeat("a", 64)
		},
		"foreign-certificate-environment": func(s *ExchangeSettings) { s.Certificate = c.WorkerEnvironment + "/certificates/worker-https" },
		"shared-worker-network": func(s *ExchangeSettings) {
			s.VirtualNetwork = s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNet
			s.Subnet = s.VirtualNetwork + "/subnets/exchange"
		},
		"shared-controller-network": func(s *ExchangeSettings) {
			s.VirtualNetwork = s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.ControllerVNet
			s.Subnet = s.VirtualNetwork + "/subnets/exchange"
		},
		"shared-infrastructure": func(s *ExchangeSettings) {
			s.InfrastructureGroup = s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.InfrastructureGroup
		},
		"backend-credentials":  func(s *ExchangeSettings) { s.BackendOrigin = "https://user:private-canary@backend.invalid" },
		"recursive-backend":    func(s *ExchangeSettings) { s.BackendOrigin = c.ExchangeOrigin + ":443" },
		"prior-stage-mismatch": func(s *ExchangeSettings) { s.Registry.PreparationEndpoint += "-changed" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := s
			mutate(&changed)
			_, err := NewExchangeGuard(c, changed, g.registry, g.reader)
			requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	definition, err := s.ApplicationDefinition(c)
	if err != nil {
		t.Fatal(err)
	}
	config, template := definition.Properties.Configuration, definition.Properties.Template
	if len(config.Secrets) != 0 || len(template.Volumes) != 0 || len(template.InitContainers) != 0 || len(template.Containers) != 1 || len(template.Containers[0].Env) != 0 || value(template.Containers[0].Image) != s.Image || value(template.Containers[0].Command[0]) != "/app/teststudio-exchange" || value(config.IdentitySettings[0].Lifecycle) != "None" {
		t.Fatal("definition gained runtime credential or executable authority")
	}
	template.Containers[0].Args[1] = to.Ptr("--public-origin=https://foreign.invalid")
	config.Ingress.CustomDomains[0].CertificateID = to.Ptr("foreign")
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("caller definition mutation changed saved guard", err)
	}
	if _, err := NewExchangeGuard(c, s, nil, g.reader); err != hostedexecution.ErrConfiguration {
		t.Fatal("missing registry guard accepted")
	}
	if _, err := NewSDKExchangeReader(c, s, nil, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("implicit credential discovery accepted")
	}
}

func TestAzureExchangeSDKRestrictsResourceKindsSecretsAndInventoryScope(t *testing.T) {
	c, s, f, g := exchangeServiceTest(t)
	reader := g.reader.(*SDKExchangeReader)
	if _, err := reader.Environment(t.Context(), c.WorkerEnvironment); err != hostedexecution.ErrOwnership {
		t.Fatal("worker environment substituted for exchange")
	}
	if _, err := reader.Application(t.Context(), s.environment()); err != hostedexecution.ErrOwnership {
		t.Fatal("environment substituted for application")
	}
	if _, err := reader.Certificate(t.Context(), s.Certificate+"-foreign"); err != hostedexecution.ErrOwnership {
		t.Fatal("foreign certificate requested")
	}
	if _, err := reader.Auth(t.Context(), s.auth()); err != hostedexecution.ErrOwnership {
		t.Fatal("child substituted for application")
	}
	if _, err := reader.Connections(t.Context(), s.Application); err != hostedexecution.ErrOwnership {
		t.Fatal("application substituted for environment")
	}
	if _, err := reader.Revisions(t.Context(), s.environment()); err != hostedexecution.ErrOwnership {
		t.Fatal("foreign revisions requested")
	}
	for name, mutate := range map[string]func(*http.Request){
		"write":              func(r *http.Request) { r.Method = "PUT" },
		"delete":             func(r *http.Request) { r.Method = "DELETE" },
		"secrets":            func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.Application + "/listSecrets" },
		"auth-token":         func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.Application + "/getAuthToken" },
		"certificate-export": func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.Certificate + "/export" },
		"subscription-inventory": func(r *http.Request) {
			r.URL.Path = "/subscriptions/" + c.Subscription + "/providers/Microsoft.App/containerApps"
		},
		"resource-group-inventory": func(r *http.Request) { r.URL.Path = strings.TrimSuffix(s.Application, "/worker-exchange-relay") },
		"filtered-revisions":       func(r *http.Request) { r.URL.Path = s.revisions(); r.URL.RawQuery += "&active=true" },
		"foreign-origin":           func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"foreign-certificate":      func(r *http.Request) { r.URL.Path = c.WorkerEnvironment + "/certificates/worker-https" },
		"wrong-version":            func(r *http.Request) { r.URL.RawQuery = "api-version=2026-07-01" },
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Application+"?api-version=2025-07-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if f.reads != 0 {
		t.Fatal("invalid exchange scope reached transport", f.reads)
	}
	if err := g.Check(t.Context(), c); err != nil || f.reads != 8 {
		t.Fatal("valid selected exchange SDK reads failed", err, f.reads)
	}
}
