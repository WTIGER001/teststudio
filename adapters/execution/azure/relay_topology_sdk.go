package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKRelayTopologyReader struct {
	networks *n.VirtualNetworksClient
	id       string
	http     *armReadHTTP
}

func NewSDKRelayTopologyReader(c Config, s RelayTopologySettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKRelayTopologyReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	id := s.BackendEndpoint.Backend.Exchange.VirtualNetwork
	h := newARMReadHTTP(c, map[string]string{id: "2025-03-01"}, transport)
	networks, err := n.NewVirtualNetworksClient(c.Subscription, credential, publicARMOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKRelayTopologyReader{networks: networks, id: id, http: h}, nil
}

func (c *SDKRelayTopologyReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKRelayTopologyReader) VirtualNetwork(ctx context.Context, id string) (n.VirtualNetwork, error) {
	if id != c.id {
		return n.VirtualNetwork{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	result, err := c.networks.Get(ctx, group, name, nil)
	return result.VirtualNetwork, sdkError(err, false)
}

var _ RelayTopologyReader = (*SDKRelayTopologyReader)(nil)
