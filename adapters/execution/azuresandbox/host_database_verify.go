package azuresandbox

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *SDKHostDatabase) verifyNetwork(values []placementResource) error {
	if len(values) != 5 {
		return hostedexecution.ErrUnavailable
	}
	s, h, d := c.settings, c.settings.Host, c.settings.Host.Database
	endpoint, err := resourceModel[n.PrivateEndpoint](values[1])
	if err != nil {
		return err
	}
	nic, err := resourceModel[n.Interface](values[2])
	if err != nil {
		return err
	}
	binding := azureworker.PrivateEndpointBinding{Endpoint: d.Endpoint, Interface: d.Interface, Service: d.Server, Group: "postgresqlServer", Subnet: d.Subnet, Region: s.Worker.Region, Members: map[string]string{"postgresqlServer": d.Address}}
	if err := binding.Verify(endpoint, nic); err != nil {
		return err
	}
	subnet, err := resourceModel[n.Subnet](values[3])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateEndpointSubnet(subnet, []n.Interface{nic}, d.Subnet, d.SubnetPrefix, d.SecurityGroup, []string{d.Endpoint}); err != nil {
		return err
	}
	return verifyNetworkGroup(values[4], d.SecurityGroup, d.Subnet, s.Worker.Region, h.databaseRules())
}

func (s Settings) verifyDatabaseService(server placementResource, connections, links []placementResource) error {
	d := s.Host.Database
	if !namedResource(server, d.Server, "Microsoft.DBforPostgreSQL/flexibleServers") || !strings.EqualFold(server.Location, s.Worker.Region) {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		State   string `json:"state"`
		Host    string `json:"fullyQualifiedDomainName"`
		Network *struct {
			Public string `json:"publicNetworkAccess"`
			Subnet string `json:"delegatedSubnetResourceId"`
			Zone   string `json:"privateDnsZoneArmResourceId"`
		} `json:"network"`
		Auth *struct {
			Password string `json:"passwordAuth"`
		} `json:"authConfig"`
		Connections *[]placementResource `json:"privateEndpointConnections"`
	}
	if err := properties(server, &props); err != nil {
		return err
	}
	if props.State != "Ready" || props.Network == nil || props.Auth == nil || props.Connections == nil {
		return hostedexecution.ErrUnavailable
	}
	if props.Host != s.Host.DatabaseHost || props.Network.Public != "Disabled" || props.Network.Subnet != "" || props.Network.Zone != "" || props.Auth.Password != "Enabled" {
		return hostedexecution.ErrOwnership
	}
	actual, err := d.connectionMap(connections)
	if err != nil {
		return err
	}
	embedded, err := d.connectionMap(*props.Connections)
	if err != nil {
		return err
	}
	if hash(actual) != hash(embedded) {
		return hostedexecution.ErrUnavailable
	}
	if len(links) != 1 {
		return hostedexecution.ErrUnavailable
	}
	link := links[0]
	if link.Name == "" || strings.ContainsAny(link.Name, "/\\?#%") || !namedResource(link, d.Server+"/privateLinkResources/"+link.Name, "Microsoft.DBforPostgreSQL/flexibleServers/privateLinkResources") {
		return hostedexecution.ErrOwnership
	}
	var pl struct {
		Group   string   `json:"groupId"`
		Members []string `json:"requiredMembers"`
		Zones   []string `json:"requiredZoneNames"`
	}
	if err := properties(link, &pl); err != nil {
		return err
	}
	if pl.Group != "postgresqlServer" || hash(pl.Members) != hash([]string{"postgresqlServer"}) || hash(pl.Zones) != hash([]string{postgresPrivateZone}) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (d HostDatabaseSettings) connectionMap(values []placementResource) (map[string]string, error) {
	expected := map[string]bool{}
	for _, id := range d.consumers() {
		expected[strings.ToLower(id)] = true
	}
	if len(values) != len(expected) {
		return nil, hostedexecution.ErrUnavailable
	}
	result, seen := map[string]string{}, map[string]bool{}
	for _, value := range values {
		if value.Name == "" || len(value.Name) > 128 || strings.ContainsAny(value.Name, "/\\?#%") || !namedResource(value, d.Server+"/privateEndpointConnections/"+value.Name, "Microsoft.DBforPostgreSQL/flexibleServers/privateEndpointConnections") {
			return nil, hostedexecution.ErrOwnership
		}
		var props struct {
			State    string `json:"provisioningState"`
			Endpoint *struct {
				ID string `json:"id"`
			} `json:"privateEndpoint"`
			Groups     []string `json:"groupIds"`
			Connection *struct {
				Status  string `json:"status"`
				Actions string `json:"actionsRequired"`
			} `json:"privateLinkServiceConnectionState"`
		}
		if err := properties(value, &props); err != nil {
			return nil, err
		}
		if (props.State != "Succeeded" && props.State != "Ready") || props.Endpoint == nil || props.Connection == nil || props.Connection.Status == "" || props.Connection.Status == "Pending" {
			return nil, hostedexecution.ErrUnavailable
		}
		endpoint, id := strings.ToLower(props.Endpoint.ID), strings.ToLower(value.ID)
		if !expected[endpoint] || result[endpoint] != "" || seen[id] || props.Connection.Status != "Approved" || hash(props.Groups) != hash([]string{"postgresqlServer"}) {
			return nil, hostedexecution.ErrOwnership
		}
		if props.Connection.Actions != "" && props.Connection.Actions != "None" {
			return nil, hostedexecution.ErrUnavailable
		}
		result[endpoint], seen[id] = id, true
	}
	return result, nil
}
