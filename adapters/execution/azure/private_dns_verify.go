package azureworker

import (
	"strings"

	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s privateZoneBinding) verify(observed privateZoneObservation) error {
	id := s.ID
	zone := observed.zone
	if !dnsIdentity(value(zone.ID), value(zone.Name), value(zone.Type), id, "Microsoft.Network/privateDnsZones", value(zone.Location), "global") || zone.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	props := zone.Properties
	if value(props.ProvisioningState) != p.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if len(observed.links) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if len(observed.links) != 1 {
		return hostedexecution.ErrOwnership
	}
	link := observed.links[0]
	if !dnsIdentity(value(link.ID), value(link.Name), value(link.Type), id+"/virtualNetworkLinks/"+s.LinkName, "Microsoft.Network/privateDnsZones/virtualNetworkLinks", value(link.Location), "global") || link.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	l := link.Properties
	if value(l.ProvisioningState) != p.ProvisioningStateSucceeded || value(l.VirtualNetworkLinkState) != p.VirtualNetworkLinkStateCompleted {
		return hostedexecution.ErrUnavailable
	}
	if l.RegistrationEnabled == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(l.RegistrationEnabled) || l.VirtualNetwork == nil || !strings.EqualFold(value(l.VirtualNetwork.ID), s.VirtualNetwork) || l.ResolutionPolicy != nil && value(l.ResolutionPolicy) != p.ResolutionPolicyDefault {
		return hostedexecution.ErrOwnership
	}
	expected := s.linkDefinition()
	fields := document(l)
	for _, key := range []string{"provisioningState", "virtualNetworkLinkState", "resolutionPolicy"} {
		delete(fields, key)
	}
	expectedFields := document(expected.Properties)
	delete(expectedFields, "resolutionPolicy")
	if fingerprint(fields) != fingerprint(expectedFields) {
		return hostedexecution.ErrOwnership
	}
	if err := s.verifyRecords(observed.records); err != nil {
		return err
	}
	if props.NumberOfRecordSets == nil || props.NumberOfVirtualNetworkLinks == nil || props.NumberOfVirtualNetworkLinksWithRegistration == nil || value(props.NumberOfRecordSets) != int64(len(observed.records)) || value(props.NumberOfVirtualNetworkLinks) != 1 || value(props.NumberOfVirtualNetworkLinksWithRegistration) != 0 {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (s privateZoneBinding) verifyRecords(records []p.RecordSet) error {
	zone := s.ID
	expected := s.recordDefinitions()
	count := len(expected) + 1
	if len(records) < count {
		return hostedexecution.ErrUnavailable
	}
	if len(records) != count {
		return hostedexecution.ErrOwnership
	}
	_, zoneName := resourceParts(zone)
	seen := map[string]bool{}
	soa := false
	for _, record := range records {
		name, kind := value(record.Name), value(record.Type)
		key := strings.ToLower(kind + "/" + name)
		if seen[key] || record.Properties == nil {
			return hostedexecution.ErrOwnership
		}
		seen[key] = true
		props := record.Properties
		fqdn := zoneName
		if name != "@" {
			fqdn = name + "." + fqdn
		}
		if value(props.IsAutoRegistered) || props.Fqdn != nil && !strings.EqualFold(strings.TrimSuffix(value(props.Fqdn), "."), fqdn) || props.TTL == nil || value(props.TTL) < 1 {
			return hostedexecution.ErrOwnership
		}
		fields := document(props)
		for _, key := range []string{"metadata", "fqdn", "isAutoRegistered"} {
			delete(fields, key)
		}
		if strings.EqualFold(kind, "Microsoft.Network/privateDnsZones/SOA") && name == "@" {
			if soa || !strings.EqualFold(value(record.ID), zone+"/SOA/@") || props.SoaRecord == nil {
				return hostedexecution.ErrOwnership
			}
			s := props.SoaRecord
			if value(s.Host) == "" || value(s.Email) == "" || s.SerialNumber == nil || value(s.SerialNumber) < 0 || value(s.RefreshTime) < 1 || value(s.RetryTime) < 1 || value(s.ExpireTime) < 1 || value(s.MinimumTTL) < 1 || min(value(props.TTL), value(s.MinimumTTL)) > 300 {
				return hostedexecution.ErrOwnership
			}
			baseline := p.RecordSetProperties{TTL: props.TTL, SoaRecord: s}
			if fingerprint(fields) != fingerprint(document(baseline)) {
				return hostedexecution.ErrOwnership
			}
			soa = true
			continue
		}
		wanted, ok := expected[name]
		if !ok || !strings.EqualFold(kind, "Microsoft.Network/privateDnsZones/A") || !strings.EqualFold(value(record.ID), zone+"/A/"+name) || value(props.TTL) > 300 {
			return hostedexecution.ErrOwnership
		}
		wanted.Properties.TTL = props.TTL
		if fingerprint(fields) != fingerprint(document(wanted.Properties)) {
			return hostedexecution.ErrOwnership
		}
		delete(expected, name)
	}
	if !soa || len(expected) != 0 {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
