package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKPerimeterReader struct {
	groups           *n.SecurityGroupsClient
	routes           *n.RouteTablesClient
	http             *armReadHTTP
	groupID, routeID string
}

func NewSDKPerimeterReader(c Config, s PerimeterSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPerimeterReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return newSDKPerimeterReader(c, s.Topology.SecurityGroup, s.Topology.RouteTable, credential, transport)
}

func newSDKPerimeterReader(c Config, groupID, routeID string, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPerimeterReader, error) {
	h := newARMReadHTTP(c, map[string]string{groupID: "2025-03-01", routeID: "2025-03-01"}, transport)
	options := publicARMOptions(h)
	groups, err := n.NewSecurityGroupsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	routes, err := n.NewRouteTablesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKPerimeterReader{groups: groups, routes: routes, http: h, groupID: groupID, routeID: routeID}, nil
}

func (c *SDKPerimeterReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKPerimeterReader) SecurityGroup(ctx context.Context, id string) (n.SecurityGroup, error) {
	if id != c.groupID {
		return n.SecurityGroup{}, hostedexecution.ErrOwnership
	}
	group, name := resourceParts(id)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.groups.Get(ctx, group, name, nil)
	return result.SecurityGroup, sdkError(err, false)
}

func (c *SDKPerimeterReader) RouteTable(ctx context.Context, id string) (n.RouteTable, error) {
	if id != c.routeID {
		return n.RouteTable{}, hostedexecution.ErrOwnership
	}
	group, name := resourceParts(id)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.routes.Get(ctx, group, name, nil)
	return result.RouteTable, sdkError(err, false)
}

var _ PerimeterReader = (*SDKPerimeterReader)(nil)
