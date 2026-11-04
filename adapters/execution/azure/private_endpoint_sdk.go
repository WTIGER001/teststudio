package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type sdkEndpointReader struct {
	endpoints         *n.PrivateEndpointsClient
	interfaces        *n.InterfacesClient
	subnets           *n.SubnetsClient
	groups            *n.SecurityGroupsClient
	allowedEndpoints  map[string]bool
	allowedInterfaces map[string]bool
	allowedSubnets    map[string]bool
	allowedGroups     map[string]bool
	http              *armReadHTTP
}

func newSDKEndpointReader(c Config, endpoints, interfaces, subnets, groups []string, credential azcore.TokenCredential, transport policy.Transporter) (*sdkEndpointReader, error) {
	if credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	allowed := map[string]string{}
	sets := []map[string]bool{}
	for _, ids := range [][]string{endpoints, interfaces, subnets, groups} {
		set := map[string]bool{}
		for _, id := range ids {
			set[id] = true
			allowed[id] = "2025-03-01"
		}
		sets = append(sets, set)
	}
	h := newARMReadHTTP(c, allowed, transport)
	options := publicARMOptions(h)
	endpointClient, err := n.NewPrivateEndpointsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	interfaceClient, err := n.NewInterfacesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	subnetClient, err := n.NewSubnetsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	groupClient, err := n.NewSecurityGroupsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}

	return &sdkEndpointReader{endpoints: endpointClient, interfaces: interfaceClient, subnets: subnetClient, groups: groupClient, allowedEndpoints: sets[0], allowedInterfaces: sets[1], allowedSubnets: sets[2], allowedGroups: sets[3], http: h}, nil
}

func (c *sdkEndpointReader) Close() { c.http.CloseIdleConnections() }

func (c *sdkEndpointReader) Endpoint(ctx context.Context, id string) (n.PrivateEndpoint, error) {
	if !c.allowedEndpoints[id] {
		return n.PrivateEndpoint{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.endpoints.Get(ctx, group, name, nil)
	return response.PrivateEndpoint, sdkError(err, false)
}

func (c *sdkEndpointReader) Interface(ctx context.Context, id string) (n.Interface, error) {
	if !c.allowedInterfaces[id] {
		return n.Interface{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.interfaces.Get(ctx, group, name, nil)
	return response.Interface, sdkError(err, false)
}

func (c *sdkEndpointReader) Subnet(ctx context.Context, id string) (n.Subnet, error) {
	if !c.allowedSubnets[id] {
		return n.Subnet{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	parts := strings.Split(id, "/")
	response, err := c.subnets.Get(ctx, group, name, parts[10], nil)
	return response.Subnet, sdkError(err, false)
}

func (c *sdkEndpointReader) SecurityGroup(ctx context.Context, id string) (n.SecurityGroup, error) {
	if !c.allowedGroups[id] {
		return n.SecurityGroup{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.groups.Get(ctx, group, name, nil)
	return response.SecurityGroup, sdkError(err, false)
}
