package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKPrivateDNSReader struct {
	settings PrivateDNSSettings
	zones    *p.PrivateZonesClient
	records  *p.RecordSetsClient
	links    *p.VirtualNetworkLinksClient
	http     *armReadHTTP
}

func NewSDKPrivateDNSReader(c Config, s PrivateDNSSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPrivateDNSReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	resources := map[string]string{}
	pages := map[string]bool{}
	for _, id := range []string{s.ExchangeZone, s.RegistryZone} {
		resources[id] = "2024-06-01"
		for _, suffix := range []string{"/ALL", "/virtualNetworkLinks"} {
			resources[id+suffix] = "2024-06-01"
			pages[id+suffix] = true
		}
	}
	h := newARMReadHTTP(c, resources, transport)
	h.pages = pages
	options := publicARMOptions(h)
	zones, err := p.NewPrivateZonesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	records, err := p.NewRecordSetsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	links, err := p.NewVirtualNetworkLinksClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKPrivateDNSReader{settings: s, zones: zones, records: records, links: links, http: h}, nil
}

func (c *SDKPrivateDNSReader) Close() { c.http.CloseIdleConnections() }
func (c *SDKPrivateDNSReader) owns(id string) bool {
	return id == c.settings.ExchangeZone || id == c.settings.RegistryZone
}

func (c *SDKPrivateDNSReader) Zone(ctx context.Context, id string) (p.PrivateZone, error) {
	if !c.owns(id) {
		return p.PrivateZone{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.zones.Get(ctx, group, name, nil)
	return result.PrivateZone, sdkError(err, false)
}

func (c *SDKPrivateDNSReader) Records(ctx context.Context, id string) ([]p.RecordSet, error) {
	if !c.owns(id) {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.records.NewListPager(group, name, nil)
	max := 2
	if id == c.settings.RegistryZone {
		max = 3
	}
	return armInventory(ctx, c.http, id+"/ALL", "2024-06-01", max, 3, pager.More, func(ctx context.Context) ([]*p.RecordSet, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

func (c *SDKPrivateDNSReader) Links(ctx context.Context, id string) ([]p.VirtualNetworkLink, error) {
	if !c.owns(id) {
		return nil, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	pager := c.links.NewListPager(group, name, nil)
	return armInventory(ctx, c.http, id+"/virtualNetworkLinks", "2024-06-01", 1, 2, pager.More, func(ctx context.Context) ([]*p.VirtualNetworkLink, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	})
}

var _ PrivateDNSReader = (*SDKPrivateDNSReader)(nil)
