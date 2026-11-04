package azureworker

import (
	"context"
	"errors"
	"sync"
	"time"

	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PrivateDNSReader interface {
	Zone(context.Context, string) (p.PrivateZone, error)
	Records(context.Context, string) ([]p.RecordSet, error)
	Links(context.Context, string) ([]p.VirtualNetworkLink, error)
}

type PrivateDNSGuard struct {
	worker   Config
	settings PrivateDNSSettings
	dns      *DNSGuard
	reader   PrivateDNSReader
}

type privateZoneObservation struct {
	zone    p.PrivateZone
	records []p.RecordSet
	links   []p.VirtualNetworkLink
	errors  [3]error
}

func NewPrivateDNSGuard(c Config, s PrivateDNSSettings, dns *DNSGuard, reader PrivateDNSReader) (*PrivateDNSGuard, error) {
	s, err := s.canonical(c)
	if err != nil || dns == nil || reader == nil || fingerprint(c) != fingerprint(dns.worker) || fingerprint(s.DNS) != fingerprint(dns.settings) {
		return nil, hostedexecution.ErrConfiguration
	}
	return &PrivateDNSGuard{worker: c, settings: s, dns: dns, reader: reader}, nil
}

func (g *PrivateDNSGuard) Check(ctx context.Context, c Config) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if fingerprint(c) != fingerprint(g.worker) {
		return hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := g.dns.Check(ctx, c); err != nil {
		return err
	}
	zones := []string{g.settings.ExchangeZone, g.settings.RegistryZone}
	observed := make([]privateZoneObservation, len(zones))
	var group sync.WaitGroup
	for i, id := range zones {
		group.Go(func() { observed[i].zone, observed[i].errors[0] = g.reader.Zone(ctx, id) })
		group.Go(func() { observed[i].records, observed[i].errors[1] = g.reader.Records(ctx, id) })
		group.Go(func() { observed[i].links, observed[i].errors[2] = g.reader.Links(ctx, id) })
	}
	group.Wait()
	for _, entry := range observed {
		for _, err := range entry.errors {
			if errors.Is(err, hostedexecution.ErrOwnership) {
				return hostedexecution.ErrOwnership
			}
		}
	}
	for _, entry := range observed {
		for _, err := range entry.errors {
			if err != nil {
				return hostedexecution.ErrUnavailable
			}
		}
	}
	for i, id := range zones {
		if err := g.verifyZone(id, observed[i]); err != nil {
			return err
		}
	}
	return nil
}
