package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s PrivateDNSSettings) ZoneDefinition(c Config, zone string) (p.PrivateZone, error) {
	if _, err := s.canonical(c); err != nil {
		return p.PrivateZone{}, err
	}
	if zone != s.ExchangeZone && zone != s.RegistryZone {
		return p.PrivateZone{}, hostedexecution.ErrOwnership
	}
	return p.PrivateZone{Location: to.Ptr("global")}, nil
}

func (s PrivateDNSSettings) LinkDefinition(c Config, zone string) (p.VirtualNetworkLink, error) {
	if _, err := s.ZoneDefinition(c, zone); err != nil {
		return p.VirtualNetworkLink{}, err
	}
	return s.binding(c, zone).linkDefinition(), nil
}

func (s PrivateDNSSettings) RecordDefinitions(c Config, zone string) (map[string]p.RecordSet, error) {
	if _, err := s.ZoneDefinition(c, zone); err != nil {
		return nil, err
	}
	return s.binding(c, zone).recordDefinitions(), nil
}

func (s privateZoneBinding) linkDefinition() p.VirtualNetworkLink {
	props := &p.VirtualNetworkLinkProperties{VirtualNetwork: &p.SubResource{ID: to.Ptr(s.VirtualNetwork)}, RegistrationEnabled: to.Ptr(false)}
	if s.ResolutionPolicy {
		props.ResolutionPolicy = to.Ptr(p.ResolutionPolicyDefault)
	}
	return p.VirtualNetworkLink{Location: to.Ptr("global"), Properties: props}
}

func (s privateZoneBinding) recordDefinitions() map[string]p.RecordSet {
	records := map[string]p.RecordSet{}
	for name, ip := range s.Records {
		records[name] = p.RecordSet{Properties: &p.RecordSetProperties{TTL: to.Ptr[int64](60), ARecords: []*p.ARecord{{IPv4Address: to.Ptr(ip)}}}}
	}
	return records
}
