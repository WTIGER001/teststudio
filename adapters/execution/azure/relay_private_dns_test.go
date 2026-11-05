package azureworker

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type relayPrivateDNSFixture struct {
	*privateDNSFixture
	endpoint *relayRegistryFixture
}

func relayPrivateDNSTest(t *testing.T) (Config, RelayPrivateDNSSettings, *relayPrivateDNSFixture, *RelayPrivateDNSGuard) {
	t.Helper()
	c, endpoint, previous, prior := relayRegistryTest(t)
	zone := strings.Replace(c.resourceID("Microsoft.Network", "privateDnsZones", "privatelink.azurecr.io"), "/resourceGroups/"+c.ResourceGroup+"/", "/resourceGroups/relay-private-dns/", 1)
	s := RelayPrivateDNSSettings{Endpoint: endpoint, RegistryZone: zone}
	f := &relayPrivateDNSFixture{privateDNSFixture: privateZoneFixture(t, []privateZoneBinding{s.binding(c)}), endpoint: previous}
	reader, err := NewSDKRelayPrivateDNSReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewRelayPrivateDNSGuard(c, s, prior, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func TestAzureRelayPrivateDNSBindsFreshEndpointAddressesWithoutParentRereads(t *testing.T) {
	c, s, f, g := relayPrivateDNSTest(t)
	s.Endpoint.DNS.Firewall.Perimeter.RegistryEndpointIPs[0] = "8.8.8.8"
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*3 || f.endpoint.reads != i*4 || f.endpoint.registry.reads != i*4 || f.endpoint.network.reads != i {
			t.Fatal("relay private zone rejected or parent read repeated", err, f.reads, f.endpoint.reads, f.endpoint.registry.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("selected private zone became complete network authority")
	}
	f.records[s.RegistryZone][1].Properties.ARecords[0].IPv4Address = to.Ptr("10.20.2.5")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 9 {
		t.Fatal("changed private address reused cached approval")
	}
	f.endpoint.nic.Properties.PrivateEndpoint.ID = to.Ptr("other-endpoint-canary")
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 9 {
		t.Fatal("unverified endpoint reached private zone reads")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 9 {
		t.Fatal("foreign profile reached private zone reads")
	}
}

func TestAzureRelayPrivateDNSRejectsAliasesRegistrationAndForeignLinks(t *testing.T) {
	for name, mutate := range map[string]func(*relayPrivateDNSFixture, string){
		"worker-address": func(f *relayPrivateDNSFixture, zone string) {
			f.records[zone][1].Properties.ARecords[0].IPv4Address = to.Ptr("10.20.2.5")
		},
		"public-address": func(f *relayPrivateDNSFixture, zone string) {
			f.records[zone][1].Properties.ARecords[0].IPv4Address = to.Ptr("8.8.8.8")
		},
		"wildcard": func(f *relayPrivateDNSFixture, zone string) { f.records[zone][1].Name = to.Ptr("*") },
		"cname": func(f *relayPrivateDNSFixture, zone string) {
			f.records[zone][1].Properties.CnameRecord = &p.CnameRecord{Cname: to.Ptr("foreign.invalid")}
		},
		"ipv6": func(f *relayPrivateDNSFixture, zone string) {
			f.records[zone][1].Properties.AaaaRecords = []*p.AaaaRecord{{IPv6Address: to.Ptr("fd00::4")}}
		},
		"extra-address": func(f *relayPrivateDNSFixture, zone string) {
			f.records[zone][1].Properties.ARecords = append(f.records[zone][1].Properties.ARecords, &p.ARecord{IPv4Address: to.Ptr("10.40.3.6")})
		},
		"auto-record": func(f *relayPrivateDNSFixture, zone string) {
			f.records[zone][1].Properties.IsAutoRegistered = to.Ptr(true)
		},
		"positive-cache": func(f *relayPrivateDNSFixture, zone string) { f.records[zone][1].Properties.TTL = to.Ptr[int64](3600) },
		"negative-cache": func(f *relayPrivateDNSFixture, zone string) {
			f.records[zone][0].Properties.SoaRecord.MinimumTTL = to.Ptr[int64](3600)
		},
		"registration": func(f *relayPrivateDNSFixture, zone string) {
			f.links[zone][0].Properties.RegistrationEnabled = to.Ptr(true)
		},
		"public-fallback": func(f *relayPrivateDNSFixture, zone string) {
			f.links[zone][0].Properties.ResolutionPolicy = to.Ptr(p.ResolutionPolicyNxDomainRedirect)
		},
		"worker-link": func(f *relayPrivateDNSFixture, zone string) {
			f.links[zone][0].Properties.VirtualNetwork.ID = to.Ptr(f.endpoint.settings.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology.WorkerVNet)
		},
		"extra-link": func(f *relayPrivateDNSFixture, zone string) {
			f.links[zone] = append(f.links[zone], clone(f.links[zone][0]))
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, s, f, g := relayPrivateDNSTest(t)
			mutate(f, s.RegistryZone)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureRelayPrivateDNSSDKPreservesIncompleteStateAndExactZoneScope(t *testing.T) {
	c, s, f, g := relayPrivateDNSTest(t)
	reader := g.reader.(*SDKPrivateDNSReader)
	workerZone := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.RegistryZone
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST"; r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Path = workerZone },
		func(r *http.Request) { r.URL.Path += "/A/unselected" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$filter=approved" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.RegistryZone+"?api-version=2024-06-01", nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.Zone(t.Context(), workerZone)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected private zone request reached transport")
	}
	f.mode = "paged"
	if err := g.Check(t.Context(), c); err != nil || f.reads != 4 || f.pages[s.RegistryZone+"/ALL"] != 2 {
		t.Fatal("bounded complete registry record paging rejected", err, f.reads)
	}
	for _, mode := range []string{"unavailable", "foreign-page", "foreign-zone-page", "endless"} {
		f.mode = mode
		want := hostedexecution.ErrUnavailable
		if mode == "foreign-page" || mode == "foreign-zone-page" {
			want = hostedexecution.ErrOwnership
		}
		requireDeadlineError(t, g.Check(t.Context(), c), want)
	}
	f.mode = ""
	f.zones[s.RegistryZone].Properties.NumberOfRecordSets = to.Ptr[int64](4)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.links[s.RegistryZone] = nil
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
}

func TestAzureRelayPrivateDNSSettingsDeriveRecordsAndRequireDedicatedZone(t *testing.T) {
	c, s, _, g := relayPrivateDNSTest(t)
	workerZone := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS.RegistryZone
	for _, zone := range []string{workerZone, strings.Replace(s.RegistryZone, "privatelink.azurecr.io", "azurecr.io", 1), strings.Replace(s.RegistryZone, c.Subscription, "99999999-2222-3333-4444-555555555555", 1), s.RegistryZone + "?secret=private", ""} {
		changed := s
		changed.RegistryZone = zone
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.Endpoint.SecurityGroup = c.resourceID("Microsoft.Network", "networkSecurityGroups", "other-relay-registry")
	_, err := NewRelayPrivateDNSGuard(c, changed, g.endpoint, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKRelayPrivateDNSReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	region := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.region()
	first, _ := s.RecordDefinitions(c)
	first[name].Properties.ARecords[0].IPv4Address = to.Ptr("8.8.8.8")
	second, err := s.RecordDefinitions(c)
	if err != nil || len(second) != 2 || value(second[name].Properties.ARecords[0].IPv4Address) != s.Endpoint.RegistryIP || value(second[name+"."+region+".data"].Properties.ARecords[0].IPv4Address) != s.Endpoint.DataIP {
		t.Fatal("private records differ from endpoint members or retain mutation", err)
	}
	link, _ := s.LinkDefinition(c)
	if value(link.Properties.VirtualNetwork.ID) != s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork || value(link.Properties.RegistrationEnabled) || value(link.Properties.ResolutionPolicy) != p.ResolutionPolicyDefault {
		t.Fatal("relay zone link changed network, registration, or fallback")
	}
}
