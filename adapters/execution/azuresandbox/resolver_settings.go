package azuresandbox

import (
	"net/url"
	"slices"
	"strings"

	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ResolverSettings struct {
	Policy    string `json:"policy"`
	AllowList string `json:"allowList"`
	BlockList string `json:"blockList"`
}

func (r ResolverSettings) validate(c Config) error {
	if !resourceID(c, r.Policy, "Microsoft.Network", "dnsResolverPolicies") || !resourceID(c, r.AllowList, "Microsoft.Network", "dnsResolverDomainLists") || !resourceID(c, r.BlockList, "Microsoft.Network", "dnsResolverDomainLists") || r.AllowList == r.BlockList {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func (r ResolverSettings) binding(s Settings) azureworker.ResolverBinding {
	u, _ := url.Parse(s.Worker.ExchangeOrigin)
	name := strings.TrimSuffix(s.Worker.Registry, ".azurecr.io")
	domains := []string{u.Hostname(), s.Worker.Registry, name + "." + s.Worker.Region + ".data.azurecr.io", name + ".privatelink.azurecr.io", name + "." + s.Worker.Region + ".data.privatelink.azurecr.io"}
	slices.Sort(domains)
	return azureworker.ResolverBinding{Policy: r.Policy, AllowList: r.AllowList, BlockList: r.BlockList, VirtualNetwork: s.Placement.virtualNetwork(), Role: "sandbox", Domains: domains}
}

func (s Settings) ResolverDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s.Placement.Network.Resolver.binding(s).Definitions(s.Worker.Region), nil
}
