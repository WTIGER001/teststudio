package azureworker

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKPrivateDNSInventoryReader struct {
	subscription string
	credential   azcore.TokenCredential
	base         policy.Transporter
}

type privateDNSInventoryBudget struct {
	base      policy.Transporter
	remaining atomic.Int64
}

func (b *privateDNSInventoryBudget) Do(r *http.Request) (*http.Response, error) {
	if b.remaining.Add(-1) < 0 {
		return nil, hostedexecution.ErrUnavailable
	}
	return b.base.Do(r)
}

func NewSDKPrivateDNSInventoryReader(c Config, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPrivateDNSInventoryReader, error) {
	if Validate(c) != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return NewBoundPrivateDNSInventoryReader(c.Subscription, credential, transport)
}

func NewBoundPrivateDNSInventoryReader(subscription string, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPrivateDNSInventoryReader, error) {
	if !guid.MatchString(subscription) || subscription == "00000000-0000-0000-0000-000000000000" || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKPrivateDNSInventoryReader{subscription: subscription, credential: credential, base: newARMReadHTTP(Config{Subscription: subscription}, nil, transport).base}, nil
}

func (c *SDKPrivateDNSInventoryReader) Close() {
	if closer, ok := c.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (c *SDKPrivateDNSInventoryReader) Observe(ctx context.Context) (PrivateDNSZoneInventory, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	empty := PrivateDNSZoneInventory{}
	path := "/subscriptions/" + c.subscription + "/providers/Microsoft.Network/privateDnsZones"
	budget := &privateDNSInventoryBudget{base: c.base}
	budget.remaining.Store(privateDNSInventoryRequests)
	h := newARMReadHTTP(Config{Subscription: c.subscription}, map[string]string{path: "2024-06-01"}, budget)
	h.pages = map[string]bool{path: true}
	zonesClient, err := p.NewPrivateZonesClient(c.subscription, c.credential, publicARMOptions(h))
	if err != nil {
		return empty, hostedexecution.ErrConfiguration
	}
	rootCtx, rootCancel := context.WithTimeout(ctx, 5*time.Second)
	pager := zonesClient.NewListPager(nil)
	zones, err := armInventory(rootCtx, h, path, "2024-06-01", privateDNSInventoryZones, privateDNSInventoryPages, pager.More, func(ctx context.Context) ([]*p.PrivateZone, *string, error) {
		page, err := pager.NextPage(ctx)
		if err == nil && page.Value == nil {
			return nil, nil, hostedexecution.ErrUnavailable
		}
		return page.Value, page.NextLink, err
	})
	rootCancel()
	if err != nil {
		return empty, err
	}
	result := PrivateDNSZoneInventory{Zones: []string{}, Links: []PrivateDNSZoneLink{}}
	seen := map[string]bool{}
	var total int64
	for i, zone := range zones {
		if err := privateDNSInventoryZone(Config{Subscription: c.subscription}, zone); err != nil {
			return empty, err
		}
		parts := strings.Split(value(zone.ID), "/")
		id := "/subscriptions/" + c.subscription + "/resourceGroups/" + parts[4] + "/providers/Microsoft.Network/privateDnsZones/" + parts[8]
		zones[i].ID = &id
		if seen[inventoryIdentityKey(id)] {
			return empty, hostedexecution.ErrOwnership
		}
		seen[inventoryIdentityKey(id)] = true
		total += value(zone.Properties.NumberOfVirtualNetworkLinks)
		if total > privateDNSInventoryLinks {
			return empty, hostedexecution.ErrUnavailable
		}
		result.Zones = append(result.Zones, id)
		linkPath := id + "/virtualNetworkLinks"
		h.resources[linkPath], h.pages[linkPath] = "2024-06-01", true
	}
	linksClient, err := p.NewVirtualNetworkLinksClient(c.subscription, c.credential, publicARMOptions(h))
	if err != nil {
		return empty, hostedexecution.ErrConfiguration
	}
	for start := 0; start < len(zones); start += 8 {
		batch := zones[start:min(start+8, len(zones))]
		items := make([][]PrivateDNSZoneLink, len(batch))
		errs := make([]error, len(batch))
		var group sync.WaitGroup
		for i, zone := range batch {
			group.Go(func() { items[i], errs[i] = privateDNSZoneLinks(ctx, linksClient, h, zone) })
		}
		group.Wait()
		for _, err := range errs {
			if errors.Is(err, hostedexecution.ErrOwnership) {
				return empty, hostedexecution.ErrOwnership
			}
		}
		for i, err := range errs {
			if err != nil {
				return empty, hostedexecution.ErrUnavailable
			}
			result.Links = append(result.Links, items[i]...)
			if len(result.Links) > privateDNSInventoryLinks {
				return empty, hostedexecution.ErrUnavailable
			}
		}
	}
	return result, nil
}

func privateDNSZoneLinks(ctx context.Context, client *p.VirtualNetworkLinksClient, h *armReadHTTP, zone p.PrivateZone) ([]PrivateDNSZoneLink, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	id := value(zone.ID)
	resourceGroup, name := resourceParts(id)
	pager := client.NewListPager(resourceGroup, name, nil)
	links, err := armInventory(ctx, h, id+"/virtualNetworkLinks", "2024-06-01", 1024, privateDNSInventoryPages, pager.More, func(ctx context.Context) ([]*p.VirtualNetworkLink, *string, error) {
		page, err := pager.NextPage(ctx)
		if err == nil && page.Value == nil {
			return nil, nil, hostedexecution.ErrUnavailable
		}
		return page.Value, page.NextLink, err
	})
	if err != nil {
		return nil, err
	}
	if int64(len(links)) != value(zone.Properties.NumberOfVirtualNetworkLinks) {
		return nil, hostedexecution.ErrUnavailable
	}
	result := []PrivateDNSZoneLink{}
	seen := map[string]bool{}
	var registered int64
	for _, link := range links {
		item, err := privateDNSInventoryLink(id, link)
		if err != nil {
			return nil, err
		}
		key := inventoryIdentityKey(item.ResourceID)
		if seen[key] {
			return nil, hostedexecution.ErrOwnership
		}
		seen[key] = true
		if item.RegistrationEnabled {
			registered++
		}
		result = append(result, item)
	}
	if registered != value(zone.Properties.NumberOfVirtualNetworkLinksWithRegistration) {
		return nil, hostedexecution.ErrUnavailable
	}
	return result, nil
}
