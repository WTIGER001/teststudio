package azureworker

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureWorkloadInventorySDKPermitsOnlySelectedSubscriptionCollections(t *testing.T) {
	c, s, f, g := workloadInventoryTest(t)
	reader := g.reader.(*SDKWorkloadInventoryReader)
	path := workloadInventoryPath(c, "containerApps")
	for name, mutate := range map[string]func(*http.Request){
		"write":                   func(r *http.Request) { r.Method = "PUT" },
		"delete":                  func(r *http.Request) { r.Method = "DELETE" },
		"secrets":                 func(r *http.Request) { r.Method = "POST"; r.URL.Path = s.Application + "/listSecrets" },
		"single-app":              func(r *http.Request) { r.URL.Path = s.Application },
		"resource-group-fallback": func(r *http.Request) { r.URL.Path = strings.TrimSuffix(s.Application, "/worker-exchange-relay") },
		"foreign-subscription": func(r *http.Request) {
			r.URL.Path = strings.Replace(path, c.Subscription, "99999999-2222-3333-4444-555555555555", 1)
		},
		"different-provider": func(r *http.Request) {
			r.URL.Path = "/subscriptions/" + c.Subscription + "/providers/Microsoft.Authorization/permissions"
		},
		"filter":         func(r *http.Request) { r.URL.RawQuery += "&%24filter=environmentId" },
		"select":         func(r *http.Request) { r.URL.RawQuery += "&%24select=id" },
		"extra-version":  func(r *http.Request) { r.URL.RawQuery += "&api-version=2025-07-01" },
		"foreign-origin": func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"encoded-route-alias": func(r *http.Request) {
			r.URL.RawPath = strings.Replace(path, "/subscriptions/", "/%73ubscriptions/", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+path+"?api-version=2025-07-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if f.reads != 0 {
		t.Fatal("invalid inventory scope reached transport", f.reads)
	}
	if err := g.Check(t.Context(), c); err != nil || f.reads != 2 {
		t.Fatal("subscription collection SDK route failed", err, f.reads)
	}
}

func TestAzureWorkloadInventoryBindsPriorSettingsAndExplicitCredentials(t *testing.T) {
	c, s, _, g := workloadInventoryTest(t)
	changed := s
	changed.BackendOrigin = "https://different.internal.example.test"
	if _, err := NewWorkloadInventoryGuard(c, changed, g.exchange, g.reader); err != hostedexecution.ErrConfiguration {
		t.Fatal("changed selected exchange reused its guard", err)
	}
	if _, err := NewWorkloadInventoryGuard(c, s, nil, g.reader); err != hostedexecution.ErrConfiguration {
		t.Fatal("missing selected exchange guard accepted", err)
	}
	if _, err := NewWorkloadInventoryGuard(c, s, g.exchange, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("missing inventory reader accepted", err)
	}
	if _, err := NewSDKWorkloadInventoryReader(c, nil, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("implicit inventory credential discovery accepted", err)
	}
	c.Subscription = "invalid"
	if _, err := NewSDKWorkloadInventoryReader(c, fixtureCredential{}, nil); err != hostedexecution.ErrConfiguration {
		t.Fatal("invalid subscription accepted", err)
	}
}
