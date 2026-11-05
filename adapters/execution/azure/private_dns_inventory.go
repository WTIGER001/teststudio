package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const privateDNSInventoryZones = 1024
const privateDNSInventoryLinks = 4096
const privateDNSInventoryPages = 16
const privateDNSInventoryRequests = 2048

type PrivateDNSZoneLink struct {
	ZoneID, ResourceID, VirtualNetworkID string
	RegistrationEnabled                  bool
}

type PrivateDNSZoneInventory struct {
	Zones []string
	Links []PrivateDNSZoneLink
}

type PrivateDNSInventoryReader interface {
	Observe(context.Context) (PrivateDNSZoneInventory, error)
}

type PrivateDNSInventoryGuard struct {
	worker   Config
	settings RelayPrivateDNSSettings
	selected *RelayPrivateDNSGuard
	reader   PrivateDNSInventoryReader
}

func NewPrivateDNSInventoryGuard(c Config, s RelayPrivateDNSSettings, selected *RelayPrivateDNSGuard, reader PrivateDNSInventoryReader) (*PrivateDNSInventoryGuard, error) {
	s, err := s.canonical(c)
	if err != nil || selected == nil || reader == nil || fingerprint(c) != fingerprint(selected.worker) || fingerprint(s) != fingerprint(selected.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &PrivateDNSInventoryGuard{worker: c, settings: s, selected: selected, reader: reader}, nil
}

func (g *PrivateDNSInventoryGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 115*time.Second)
	defer cancel()
	if err := g.selected.Check(ctx, c); err != nil {
		return err
	}
	observed, err := g.reader.Observe(ctx)
	if err != nil {
		return err
	}
	return verifyPrivateDNSInventory(c, g.settings, observed)
}

func verifyPrivateDNSInventory(c Config, s RelayPrivateDNSSettings, observed PrivateDNSZoneInventory) error {
	if observed.Zones == nil || observed.Links == nil || len(observed.Zones) > privateDNSInventoryZones || len(observed.Links) > privateDNSInventoryLinks {
		return hostedexecution.ErrUnavailable
	}
	worker := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.Registry.Endpoints.PrivateDNS
	vnet := worker.DNS.Firewall.Perimeter.Topology.WorkerVNet
	relay := s.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange.VirtualNetwork
	wanted := map[string]string{}
	for _, zone := range []string{worker.ExchangeZone, worker.RegistryZone} {
		wanted[inventoryIdentityKey(zone+"/virtualNetworkLinks/worker-network")] = inventoryIdentityKey(vnet)
	}
	wanted[inventoryIdentityKey(s.RegistryZone+"/virtualNetworkLinks/relay-network")] = inventoryIdentityKey(relay)
	zones, links := map[string]bool{}, map[string]bool{}
	for _, id := range observed.Zones {
		if !privateDNSInventoryID(c, id) || zones[inventoryIdentityKey(id)] {
			return hostedexecution.ErrOwnership
		}
		zones[inventoryIdentityKey(id)] = true
	}
	for _, zone := range []string{worker.ExchangeZone, worker.RegistryZone, s.RegistryZone} {
		if !zones[inventoryIdentityKey(zone)] {
			return hostedexecution.ErrUnavailable
		}
	}
	for _, link := range observed.Links {
		id := inventoryIdentityKey(link.ResourceID)
		if !zones[inventoryIdentityKey(link.ZoneID)] || links[id] || !privateDNSInventoryLinkID(link.ZoneID, link.ResourceID) || !privateDNSVNetReference(link.VirtualNetworkID) {
			return hostedexecution.ErrOwnership
		}
		links[id] = true
		target := inventoryIdentityKey(link.VirtualNetworkID)
		if expected, ok := wanted[id]; ok {
			if expected != target || link.RegistrationEnabled {
				return hostedexecution.ErrOwnership
			}
			delete(wanted, id)
		} else if strings.EqualFold(link.VirtualNetworkID, vnet) || strings.EqualFold(link.VirtualNetworkID, relay) {
			return hostedexecution.ErrOwnership
		}
	}
	if len(wanted) != 0 {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
