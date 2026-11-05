package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type NetworkDenySettings struct {
	PrivateDNS                   RelayPrivateDNSSettings `json:"privateDns"`
	WorkerAssignment             string                  `json:"workerAssignment"`
	RelayAssignment              string                  `json:"relayAssignment"`
	WorkerExchangeZoneAssignment string                  `json:"workerExchangeZoneAssignment"`
	WorkerRegistryZoneAssignment string                  `json:"workerRegistryZoneAssignment"`
	RelayRegistryZoneAssignment  string                  `json:"relayRegistryZoneAssignment"`
}

type networkDenyBinding struct{ assignment, scope string }

func (s NetworkDenySettings) canonical(c Config) (NetworkDenySettings, error) {
	var err error
	s.PrivateDNS, err = s.PrivateDNS.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	for _, binding := range s.bindings() {
		key := inventoryIdentityKey(binding.assignment)
		if !networkDenyAssignmentID(binding.assignment, binding.scope) || seen[key] {
			return s, hostedexecution.ErrConfiguration
		}
		seen[key] = true
	}
	return s, nil
}

func networkDenyAssignmentID(id, scope string) bool {
	parts := strings.Split(id, "/")
	n := len(parts)
	if n < 5 {
		return false
	}
	return strings.EqualFold(strings.Join(parts[:n-4], "/"), scope) && strings.EqualFold(parts[n-4], "providers") && strings.EqualFold(parts[n-3], "Microsoft.Authorization") && strings.EqualFold(parts[n-2], "denyAssignments") && guid.MatchString(strings.ToLower(parts[n-1]))
}

func (s NetworkDenySettings) bindings() []networkDenyBinding {
	exchange := s.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend.Exchange
	zones := exchange.Registry.Endpoints.PrivateDNS
	worker := zones.DNS.Firewall.Perimeter.Topology.WorkerVNet
	return []networkDenyBinding{
		{s.WorkerAssignment, worker}, {s.RelayAssignment, exchange.VirtualNetwork},
		{s.WorkerExchangeZoneAssignment, zones.ExchangeZone}, {s.WorkerRegistryZoneAssignment, zones.RegistryZone},
		{s.RelayRegistryZoneAssignment, s.PrivateDNS.RegistryZone},
	}
}
