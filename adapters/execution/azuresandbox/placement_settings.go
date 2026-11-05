package azuresandbox

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const pullRole = "/providers/Microsoft.Authorization/roleDefinitions/7f951dda-4ed3-4680-a7ca-43fe172d538d"

type PlacementSettings struct {
	Subnet          string          `json:"subnet"`
	SubnetPrefix    string          `json:"subnetPrefix"`
	SecurityGroup   string          `json:"securityGroup"`
	RouteTable      string          `json:"routeTable"`
	Registry        string          `json:"registry"`
	PullAssignment  string          `json:"pullAssignment"`
	PullClientID    string          `json:"pullClientId"`
	PullPrincipalID string          `json:"pullPrincipalId"`
	PolicyProfile   string          `json:"policyProfile"`
	Network         NetworkSettings `json:"network"`
	Private         PrivateSettings `json:"private"`
}

func resourceID(c Config, id, provider, kind string) bool {
	p := strings.Split(id, "/")
	return len(p) == 9 && p[0] == "" && p[1] == "subscriptions" && p[2] == c.Subscription && p[3] == "resourceGroups" && groupName.MatchString(p[4]) && !strings.HasSuffix(p[4], ".") && !strings.Contains(p[4], "..") && p[5] == "providers" && p[6] == provider && p[7] == kind && identifier.MatchString(p[8])
}

func privateIPv4Prefix(raw string) (netip.Prefix, bool) {
	space, err := netip.ParsePrefix(raw)
	private := false
	if err == nil && space == space.Masked() && space.Addr().Is4() {
		for _, raw := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
			block := netip.MustParsePrefix(raw)
			private = private || space.Bits() >= block.Bits() && block.Contains(space.Addr())
		}
	}
	return space, private
}

func (p PlacementSettings) validate(c Config) error {
	_, private := privateIPv4Prefix(p.SubnetPrefix)
	parts := strings.Split(p.Subnet, "/")
	if !private || len(parts) != 11 || parts[9] != "subnets" || !identifier.MatchString(parts[10]) || !resourceID(c, strings.Join(parts[:9], "/"), "Microsoft.Network", "virtualNetworks") || !resourceID(c, p.SecurityGroup, "Microsoft.Network", "networkSecurityGroups") || !resourceID(c, p.RouteTable, "Microsoft.Network", "routeTables") || !resourceID(c, p.Registry, "Microsoft.ContainerRegistry", "registries") || p.Registry[strings.LastIndex(p.Registry, "/")+1:] != strings.TrimSuffix(c.Registry, ".azurecr.io") || !strings.HasPrefix(p.PullAssignment, p.Registry+"/providers/Microsoft.Authorization/roleAssignments/") || !nonzeroGUID(strings.TrimPrefix(p.PullAssignment, p.Registry+"/providers/Microsoft.Authorization/roleAssignments/")) || !nonzeroGUID(p.PullClientID) || !nonzeroGUID(p.PullPrincipalID) || p.PullClientID == p.PullPrincipalID || !checksum.MatchString(p.PolicyProfile) {
		return hostedexecution.ErrConfiguration
	}
	if err := p.Network.validate(p.SubnetPrefix); err != nil {
		return err
	}
	if err := p.Private.validate(c, p); err != nil {
		return err
	}
	if err := p.Network.Topology.validate(c, p); err != nil {
		return err
	}
	return p.Network.Resolver.validate(c)
}

func (p PlacementSettings) NetworkProfile(c Config) (string, error) {
	shape := c
	shape.NetworkProfile = hash("placement validation")
	if Validate(shape) != nil || p.validate(c) != nil {
		return "", hostedexecution.ErrConfiguration
	}
	c.NetworkProfile = ""
	return hash([]any{"sandbox-placement/v5", c, p}), nil
}

func (c Config) groupID() string {
	return "/subscriptions/" + c.Subscription + "/resourceGroups/" + c.ResourceGroup + "/providers/Microsoft.App/sandboxGroups/" + c.Group
}
func (c Config) connectionID() string { return c.groupID() + "/vnetConnections/" + c.VNetConnection }
