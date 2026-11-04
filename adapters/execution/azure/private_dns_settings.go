package azureworker

import (
	"net/url"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type PrivateDNSSettings struct {
	DNS          DNSSettings `json:"dns"`
	ExchangeZone string      `json:"exchangeZone"`
	RegistryZone string      `json:"registryZone"`
	ExchangeIP   string      `json:"exchangeIp"`
	RegistryIP   string      `json:"registryIp"`
	DataIP       string      `json:"dataIp"`
}

func zoneResource(c Config, id, name string) bool {
	parts := strings.Split(id, "/")
	return len(parts) == 9 && parts[0] == "" && parts[1] == "subscriptions" && parts[2] == c.Subscription && parts[3] == "resourceGroups" && groupName.MatchString(parts[4]) && !strings.HasSuffix(parts[4], ".") && parts[5] == "providers" && parts[6] == "Microsoft.Network" && parts[7] == "privateDnsZones" && parts[8] == name
}

func (s PrivateDNSSettings) canonical(c Config) (PrivateDNSSettings, error) {
	var err error
	s.DNS, err = s.DNS.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	origin, _ := url.Parse(c.ExchangeOrigin)
	if strings.HasSuffix(origin.Hostname(), ".azurecr.io") || !zoneResource(c, s.ExchangeZone, origin.Hostname()) || !zoneResource(c, s.RegistryZone, "privatelink.azurecr.io") {
		return s, hostedexecution.ErrConfiguration
	}
	ips := []string{s.ExchangeIP, s.RegistryIP, s.DataIP}
	for _, ip := range ips {
		if _, valid := privateHost(ip); !valid {
			return s, hostedexecution.ErrConfiguration
		}
	}
	slices.Sort(ips)
	expected := s.DNS.Firewall.Perimeter.PrivateEndpointIPs
	if len(expected) != 3 || ips[0] == ips[1] || ips[1] == ips[2] || !slices.Equal(ips, expected) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func (s PrivateDNSSettings) records(c Config, zone string) map[string]string {
	if zone == s.ExchangeZone {
		return map[string]string{"@": s.ExchangeIP}
	}
	name := strings.TrimSuffix(c.Registry, ".azurecr.io")
	return map[string]string{name: s.RegistryIP, name + "." + s.DNS.RegistryDataRegion + ".data": s.DataIP}
}
