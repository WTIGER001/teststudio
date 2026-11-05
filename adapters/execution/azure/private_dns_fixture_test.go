package azureworker

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
)

type privateDNSFixture struct {
	t                         *testing.T
	mu                        sync.Mutex
	settings                  PrivateDNSSettings
	pagingZone, alternateZone string
	zones                     map[string]p.PrivateZone
	records                   map[string][]p.RecordSet
	links                     map[string][]p.VirtualNetworkLink
	pages                     map[string]int
	reads                     int
	mode                      string
}

func privateDNSTest(t *testing.T) (Config, PrivateDNSSettings, *privateDNSFixture, *PrivateDNSGuard) {
	t.Helper()
	c, dns, _, guard := dnsTest(t)
	origin, _ := url.Parse(c.ExchangeOrigin)
	s := PrivateDNSSettings{DNS: dns, ExchangeZone: c.resourceID("Microsoft.Network", "privateDnsZones", origin.Hostname()), RegistryZone: c.resourceID("Microsoft.Network", "privateDnsZones", "privatelink.azurecr.io"), ExchangeIP: "10.20.2.4", RegistryIP: "10.20.2.5", DataIP: "10.20.2.6"}
	f := privateZoneFixture(t, []privateZoneBinding{s.binding(c, s.ExchangeZone), s.binding(c, s.RegistryZone)})
	f.settings = s
	reader, err := NewSDKPrivateDNSReader(c, s, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	private, err := NewPrivateDNSGuard(c, s, guard, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, private
}

func privateZoneFixture(t *testing.T, bindings []privateZoneBinding) *privateDNSFixture {
	t.Helper()
	f := &privateDNSFixture{t: t, zones: map[string]p.PrivateZone{}, records: map[string][]p.RecordSet{}, links: map[string][]p.VirtualNetworkLink{}, pages: map[string]int{}}
	f.pagingZone = bindings[0].ID
	f.alternateZone = bindings[0].ID + "-foreign"
	if len(bindings) > 1 {
		f.alternateZone = bindings[1].ID
	}
	for _, binding := range bindings {
		id := binding.ID
		_, name := resourceParts(id)
		zone := p.PrivateZone{Location: to.Ptr("global")}
		zone.ID, zone.Name, zone.Type = to.Ptr(id), to.Ptr(name), to.Ptr("Microsoft.Network/privateDnsZones")
		count := int64(len(binding.Records) + 1)
		zone.Properties = &p.PrivateZoneProperties{ProvisioningState: to.Ptr(p.ProvisioningStateSucceeded), NumberOfRecordSets: to.Ptr(count), NumberOfVirtualNetworkLinks: to.Ptr[int64](1), NumberOfVirtualNetworkLinksWithRegistration: to.Ptr[int64](0)}
		f.zones[id] = zone
		soa := p.RecordSet{ID: to.Ptr(id + "/SOA/@"), Name: to.Ptr("@"), Type: to.Ptr("Microsoft.Network/privateDnsZones/SOA"), Properties: &p.RecordSetProperties{TTL: to.Ptr[int64](3600), SoaRecord: &p.SoaRecord{Host: to.Ptr("azureprivatedns.net"), Email: to.Ptr("azureprivatedns-host.microsoft.com"), SerialNumber: to.Ptr[int64](1), RefreshTime: to.Ptr[int64](3600), RetryTime: to.Ptr[int64](300), ExpireTime: to.Ptr[int64](2419200), MinimumTTL: to.Ptr[int64](10)}, Fqdn: to.Ptr(name + ".")}}
		f.records[id] = []p.RecordSet{soa}
		definitions := binding.recordDefinitions()
		names := make([]string, 0, len(definitions))
		for relative := range definitions {
			names = append(names, relative)
		}
		slices.Sort(names)
		for _, relative := range names {
			record := definitions[relative]
			record.ID, record.Name, record.Type = to.Ptr(id+"/A/"+relative), to.Ptr(relative), to.Ptr("Microsoft.Network/privateDnsZones/A")
			fqdn := name
			if relative != "@" {
				fqdn = relative + "." + name
			}
			record.Properties.Fqdn = to.Ptr(fqdn + ".")
			f.records[id] = append(f.records[id], record)
		}
		link := binding.linkDefinition()
		link.ID, link.Name, link.Type = to.Ptr(id+"/virtualNetworkLinks/"+binding.LinkName), to.Ptr(binding.LinkName), to.Ptr("Microsoft.Network/privateDnsZones/virtualNetworkLinks")
		link.Properties.ProvisioningState = to.Ptr(p.ProvisioningStateSucceeded)
		link.Properties.VirtualNetworkLinkState = to.Ptr(p.VirtualNetworkLinkStateCompleted)
		f.links[id] = []p.VirtualNetworkLink{link}
	}
	return f
}

func (f *privateDNSFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Query().Get("api-version") != "2024-06-01" || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("private DNS SDK escaped the resource boundary")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-provider-canary"}}), nil
	}
	if f.mode == "missing" {
		return deadlineResponse(r, 404, map[string]any{"error": map[string]any{"code": "NotFound"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	var object any
	path := r.URL.Path
	if zone, ok := f.zones[path]; ok {
		object = zone
	} else if strings.HasSuffix(path, "/ALL") {
		id := strings.TrimSuffix(path, "/ALL")
		records, ok := f.records[id]
		if !ok {
			f.t.Error("foreign records requested")
		}
		f.pages[path]++
		page := map[string]any{"value": records}
		if id == f.pagingZone {
			switch f.mode {
			case "paged", "cycle":
				if r.URL.Query().Get("$skipToken") == "" {
					page["value"] = records[:1]
					page["nextLink"] = "https://management.azure.com" + path + "?api-version=2024-06-01&%24skipToken=one"
				} else {
					page["value"] = records[1:]
					if f.mode == "cycle" {
						page["nextLink"] = "https://management.azure.com" + path + "?api-version=2024-06-01&%24skipToken=one"
					}
				}
			case "foreign-page":
				page["nextLink"] = "https://foreign.invalid/private?api-version=2024-06-01&%24skipToken=one"
			case "foreign-zone-page":
				page["nextLink"] = "https://management.azure.com" + f.alternateZone + "/ALL?api-version=2024-06-01&%24skipToken=one"
			case "endless":
				page["value"] = []any{}
				page["nextLink"] = "https://management.azure.com" + path + "?api-version=2024-06-01&%24skipToken=" + strconv.Itoa(f.pages[path])
			}
		}
		object = page
	} else if strings.HasSuffix(path, "/virtualNetworkLinks") {
		id := strings.TrimSuffix(path, "/virtualNetworkLinks")
		links, ok := f.links[id]
		if !ok {
			f.t.Error("foreign links requested")
		}
		f.pages[path]++
		object = map[string]any{"value": links}
	} else {
		f.t.Error("unowned private DNS resource requested")
		return deadlineResponse(r, 404, nil), nil
	}
	response := deadlineResponse(r, 200, object)
	if f.mode == "oversized" {
		response.ContentLength = 5 << 20
	}
	return response, nil
}
