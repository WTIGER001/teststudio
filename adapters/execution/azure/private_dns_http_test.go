package azureworker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePrivateDNSSDKScopesSelectedZoneInventoriesAndEscapedGroups(t *testing.T) {
	for _, group := range []string{"studio", "studio(group)"} {
		t.Run(group, func(t *testing.T) {
			c, s, _, _ := privateDNSTest(t)
			data, _ := json.Marshal(struct {
				Config   Config
				Settings PrivateDNSSettings
			}{c, s})
			data = bytes.ReplaceAll(data, []byte("/resourceGroups/"+c.ResourceGroup+"/"), []byte("/resourceGroups/"+group+"/"))
			var settings struct {
				Config   Config
				Settings PrivateDNSSettings
			}
			if json.Unmarshal(data, &settings) != nil {
				t.Fatal("cannot adjust fixture resources")
			}
			c, s = settings.Config, settings.Settings
			c.ResourceGroup = group
			calls := 0
			allowed := map[string]bool{s.ExchangeZone: true, s.RegistryZone: true, s.ExchangeZone + "/ALL": true, s.RegistryZone + "/ALL": true, s.ExchangeZone + "/virtualNetworkLinks": true, s.RegistryZone + "/virtualNetworkLinks": true}
			wire := transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.Host != "management.azure.com" || r.URL.RawQuery != "api-version=2024-06-01" || !allowed[r.URL.Path] || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
					t.Fatal("private DNS request widened read authority")
				}
				if r.URL.Path == s.ExchangeZone || r.URL.Path == s.RegistryZone {
					return deadlineResponse(r, 200, map[string]any{"id": r.URL.Path}), nil
				}
				return deadlineResponse(r, 200, map[string]any{"value": []any{}}), nil
			})
			reader, err := NewSDKPrivateDNSReader(c, s, fixtureCredential{}, wire)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			for _, zone := range []string{s.ExchangeZone, s.RegistryZone} {
				if _, err := reader.Zone(t.Context(), zone); err != nil {
					t.Fatal(err)
				}
				if _, err := reader.Records(t.Context(), zone); err != nil {
					t.Fatal(err)
				}
				if _, err := reader.Links(t.Context(), zone); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 6 {
				t.Fatal("SDK repeated reads", calls)
			}
			if _, err := reader.Zone(t.Context(), s.ExchangeZone+"/ALL"); err != hostedexecution.ErrOwnership {
				t.Fatal("child substituted for zone")
			}
			if _, err := reader.Records(t.Context(), s.RegistryZone+"-foreign"); err != hostedexecution.ErrOwnership {
				t.Fatal("foreign zone inventory was read")
			}
			if calls != 6 {
				t.Fatal("invalid scope reached transport")
			}
		})
	}
}

func TestAzurePrivateDNSHTTPRejectsMutationsFilteredListsAndForeignInventory(t *testing.T) {
	c, s, _, g := privateDNSTest(t)
	reader := g.reader.(*SDKPrivateDNSReader)
	calls := 0
	reader.http.base = transportFunc(func(r *http.Request) (*http.Response, error) { calls++; return deadlineResponse(r, 200, nil), nil })
	cases := map[string]func(*http.Request){
		"mutation":       func(r *http.Request) { r.Method = "PUT" },
		"delete":         func(r *http.Request) { r.Method = "DELETE" },
		"unowned-list":   func(r *http.Request) { r.URL.Path = strings.TrimSuffix(s.RegistryZone, "/privatelink.azurecr.io") },
		"record-filter":  func(r *http.Request) { r.URL.RawQuery += "&%24recordsetnamesuffix=approved" },
		"single-record":  func(r *http.Request) { r.URL.Path = s.RegistryZone + "/A/unowned" },
		"foreign-origin": func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		"foreign-zone":   func(r *http.Request) { r.URL.Path = s.RegistryZone + "-foreign/ALL" },
		"wrong-version":  func(r *http.Request) { r.URL.RawQuery = "api-version=2025-05-01" },
		"policy-read-action": func(r *http.Request) {
			r.Method = "POST"
			r.URL.Path = s.DNS.vnetPolicyPath()
			r.URL.RawQuery = "api-version=2025-05-01"
		},
		"controller-resource": func(r *http.Request) { r.URL.Path = c.ControllerEnvironment; r.URL.RawQuery = "api-version=2025-07-01" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.RegistryZone+"/ALL?api-version=2024-06-01", nil)
			mutate(r)
			_, err := reader.http.Do(r)
			requireDeadlineError(t, err, hostedexecution.ErrOwnership)
		})
	}
	if calls != 0 {
		t.Fatal("unsafe private-zone read reached transport")
	}
}
