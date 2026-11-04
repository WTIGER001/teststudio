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

type SDKEndpointReader struct {
	settings   EndpointSettings
	endpoints  *n.PrivateEndpointsClient
	interfaces *n.InterfacesClient
	subnets    *n.SubnetsClient
	groups     *n.SecurityGroupsClient
	http       *armReadHTTP
}

func NewSDKEndpointReader(c Config, s EndpointSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKEndpointReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	allowed := map[string]string{}
	for _, id := range []string{s.RegistryEndpoint, s.ExchangeEndpoint, s.RegistryNIC, s.ExchangeNIC, s.Subnet, s.SecurityGroup} {
		allowed[id] = "2025-03-01"
	}
	h := newARMReadHTTP(c, allowed, transport)
	options := publicARMOptions(h)
	endpoints, err := n.NewPrivateEndpointsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	interfaces, err := n.NewInterfacesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	subnets, err := n.NewSubnetsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	groups, err := n.NewSecurityGroupsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKEndpointReader{settings: s, endpoints: endpoints, interfaces: interfaces, subnets: subnets, groups: groups, http: h}, nil
}

func (c *SDKEndpointReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKEndpointReader) Endpoint(ctx context.Context, id string) (n.PrivateEndpoint, error) {
	if id != c.settings.RegistryEndpoint && id != c.settings.ExchangeEndpoint {
		return n.PrivateEndpoint{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.endpoints.Get(ctx, group, name, nil)
	return response.PrivateEndpoint, sdkError(err, false)
}

func (c *SDKEndpointReader) Interface(ctx context.Context, id string) (n.Interface, error) {
	if id != c.settings.RegistryNIC && id != c.settings.ExchangeNIC {
		return n.Interface{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.interfaces.Get(ctx, group, name, nil)
	return response.Interface, sdkError(err, false)
}

func (c *SDKEndpointReader) Subnet(ctx context.Context, id string) (n.Subnet, error) {
	if id != c.settings.Subnet {
		return n.Subnet{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	parts := strings.Split(id, "/")
	response, err := c.subnets.Get(ctx, group, name, parts[10], nil)
	return response.Subnet, sdkError(err, false)
}

func (c *SDKEndpointReader) SecurityGroup(ctx context.Context, id string) (n.SecurityGroup, error) {
	if id != c.settings.SecurityGroup {
		return n.SecurityGroup{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.groups.Get(ctx, group, name, nil)
	return response.SecurityGroup, sdkError(err, false)
}

var _ EndpointReader = (*SDKEndpointReader)(nil)
