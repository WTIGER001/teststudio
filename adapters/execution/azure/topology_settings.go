package azureworker

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type TopologySettings struct {
	WorkerVNet          string `json:"workerVnet"`
	WorkerSubnet        string `json:"workerSubnet"`
	WorkerVNetPrefix    string `json:"workerVnetPrefix"`
	WorkerSubnetPrefix  string `json:"workerSubnetPrefix"`
	ControllerVNet      string `json:"controllerVnet"`
	ControllerSubnet    string `json:"controllerSubnet"`
	SecurityGroup       string `json:"securityGroup"`
	RouteTable          string `json:"routeTable"`
	InfrastructureGroup string `json:"infrastructureGroup"`
}

func privatePrefix(raw string) (netip.Prefix, bool) {
	p, err := netip.ParsePrefix(raw)
	if err != nil || !p.Addr().Is4() || p != p.Masked() || p.String() != raw {
		return p, false
	}
	for _, block := range []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("172.16.0.0/12"), netip.MustParsePrefix("192.168.0.0/16")} {
		if p.Bits() >= block.Bits() && block.Contains(p.Addr()) {
			return p, true
		}
	}
	return p, false
}

func subnetResource(c Config, id, parent string) bool {
	parts := strings.Split(id, "/")
	return len(parts) == 11 && parts[9] == "subnets" && resourceName.MatchString(parts[10]) && id == parent+"/subnets/"+parts[10] && c.ownsResource(parent, "Microsoft.Network", "virtualNetworks")
}

func (s TopologySettings) validate(c Config) error {
	vnet, vok := privatePrefix(s.WorkerVNetPrefix)
	subnet, sok := privatePrefix(s.WorkerSubnetPrefix)
	if Validate(c) != nil || !c.ownsResource(s.WorkerVNet, "Microsoft.Network", "virtualNetworks") || !c.ownsResource(s.ControllerVNet, "Microsoft.Network", "virtualNetworks") || strings.EqualFold(s.WorkerVNet, s.ControllerVNet) || !subnetResource(c, s.WorkerSubnet, s.WorkerVNet) || !subnetResource(c, s.ControllerSubnet, s.ControllerVNet) || !c.ownsResource(s.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") || !c.ownsResource(s.RouteTable, "Microsoft.Network", "routeTables") || !groupName.MatchString(s.InfrastructureGroup) || strings.HasSuffix(s.InfrastructureGroup, ".") || strings.EqualFold(s.InfrastructureGroup, c.ResourceGroup) || !vok || !sok || subnet.Bits() > 27 || !vnet.Contains(subnet.Addr()) || vnet.Bits() > subnet.Bits() {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func resourceParts(id string) (group, name string) {
	parts := strings.Split(id, "/")
	return parts[4], parts[8]
}
