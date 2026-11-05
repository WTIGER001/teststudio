package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ObserverHostSettings struct {
	Identity        ReadIdentitySettings `json:"identity"`
	Application     string               `json:"application"`
	Image           string               `json:"image"`
	PullIdentity    string               `json:"pullIdentity"`
	StartupIdentity string               `json:"startupIdentity"`
	RuntimeIdentity string               `json:"runtimeIdentity"`
}

func (s ObserverHostSettings) canonical(c Config) (ObserverHostSettings, error) {
	var err error
	s.Identity, err = s.Identity.canonical(c)
	if err != nil || !c.ownsResource(s.Application, "Microsoft.App", "containerApps") || !trustedServiceImage(c, s.Image) {
		return s, hostedexecution.ErrConfiguration
	}
	backend := s.Identity.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend
	if strings.EqualFold(s.Application, backend.Application) || strings.EqualFold(s.Application, backend.Exchange.Application) {
		return s, hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	for _, id := range []string{c.PullIdentity, backend.PullIdentity, backend.StartupIdentity, backend.RuntimeIdentity, s.Identity.Identity} {
		seen[inventoryIdentityKey(id)] = true
	}
	for _, id := range []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity} {
		key := inventoryIdentityKey(id)
		if !c.ownsResource(id, "Microsoft.ManagedIdentity", "userAssignedIdentities") || seen[key] {
			return s, hostedexecution.ErrConfiguration
		}
		seen[key] = true
	}
	return s, nil
}

func (s ObserverHostSettings) lifecycles() map[string]string {
	return map[string]string{
		inventoryIdentityKey(s.PullIdentity): "None", inventoryIdentityKey(s.StartupIdentity): "None",
		inventoryIdentityKey(s.RuntimeIdentity): "Main", inventoryIdentityKey(s.Identity.Identity): "Main",
	}
}
