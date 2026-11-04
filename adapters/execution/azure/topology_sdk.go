package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKTopologyReader struct {
	environments *a.ManagedEnvironmentsClient
	networks     *n.VirtualNetworksClient
	subnets      *n.SubnetsClient
	http         *armReadHTTP
}

func NewSDKTopologyReader(c Config, s TopologySettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKTopologyReader, error) {
	if s.validate(c) != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	h := newTopologyHTTP(c, s, transport)
	options := publicARMOptions(h)
	environments, err := a.NewManagedEnvironmentsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	networks, err := n.NewVirtualNetworksClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	subnets, err := n.NewSubnetsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKTopologyReader{environments: environments, networks: networks, subnets: subnets, http: h}, nil
}

func (c *SDKTopologyReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKTopologyReader) Environment(ctx context.Context, id string) (a.ManagedEnvironment, error) {
	if !c.http.allows(id, "2025-07-01") {
		return a.ManagedEnvironment{}, hostedexecution.ErrOwnership
	}
	group, name := resourceParts(id)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.environments.Get(ctx, group, name, nil)
	return result.ManagedEnvironment, sdkError(err, false)
}

func (c *SDKTopologyReader) VirtualNetwork(ctx context.Context, id string) (n.VirtualNetwork, error) {
	if !c.http.allows(id, "2025-03-01") || len(strings.Split(id, "/")) != 9 {
		return n.VirtualNetwork{}, hostedexecution.ErrOwnership
	}
	group, name := resourceParts(id)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.networks.Get(ctx, group, name, nil)
	return result.VirtualNetwork, sdkError(err, false)
}

func (c *SDKTopologyReader) Subnet(ctx context.Context, id string) (n.Subnet, error) {
	if !c.http.allows(id, "2025-03-01") || len(strings.Split(id, "/")) != 11 {
		return n.Subnet{}, hostedexecution.ErrOwnership
	}
	group, name := resourceParts(id)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.subnets.Get(ctx, group, name, strings.Split(id, "/")[10], nil)
	return result.Subnet, sdkError(err, false)
}

var _ TopologyReader = (*SDKTopologyReader)(nil)
