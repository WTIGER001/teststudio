package azuresandbox

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s Settings) verifyRelaySubnet(r placementResource) error {
	if !namedResource(r, s.Placement.Services.Relay.Subnet, "Microsoft.Network/virtualNetworks/subnets") {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		State       string   `json:"provisioningState"`
		Prefix      string   `json:"addressPrefix"`
		Prefixes    []string `json:"addressPrefixes"`
		Delegations []struct {
			Properties struct {
				Service string `json:"serviceName"`
				State   string `json:"provisioningState"`
			} `json:"properties"`
		} `json:"delegations"`
	}
	if err := properties(r, &props); err != nil {
		return err
	}
	if err := provisioned(props.State); err != nil {
		return err
	}
	if props.Prefix == "" && len(props.Prefixes) == 1 {
		props.Prefix, props.Prefixes = props.Prefixes[0], nil
	}
	if props.Prefix != s.Placement.Services.BackendEndpoint.RelaySubnetPrefix || len(props.Prefixes) != 0 || len(props.Delegations) != 1 || !strings.EqualFold(props.Delegations[0].Properties.Service, "Microsoft.App/environments") {
		return hostedexecution.ErrOwnership
	}
	return provisioned(props.Delegations[0].Properties.State)
}
