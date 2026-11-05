package azuresandbox

import (
	"context"
	"slices"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKNetwork struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func networkVersions(s Settings) map[string]string {
	return map[string]string{s.Placement.SecurityGroup: "2025-03-01", s.Placement.RouteTable: "2025-03-01"}
}

func NewSDKNetwork(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKNetwork, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	s.Placement.Network.PrivateEndpointIPs = slices.Clone(s.Placement.Network.PrivateEndpointIPs)
	reader, err := newMetadataReader(networkVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	return &SDKNetwork{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (n *SDKNetwork) Close() { n.http.Close() }

func (n *SDKNetwork) Check(ctx context.Context, config Config) error {
	if hash(config) != hash(n.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	if ctx.Err() != nil || n.http.closed.Load() {
		return hostedexecution.ErrUnavailable
	}
	epoch, started := n.cache.begin(n.now)
	err := n.read(ctx)
	n.cache.remember(epoch, started, err, n.http.closed.Load)
	return err
}

func (n *SDKNetwork) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(n.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return n.cache.check(ctx, n.now, n.http.closed.Load, n.read)
}

func (n *SDKNetwork) read(ctx context.Context) error {
	values, err := n.observe(ctx, []string{n.settings.Placement.SecurityGroup, n.settings.Placement.RouteTable})
	if err != nil {
		return err
	}
	if err := n.settings.verifySecurityGroup(values[0]); err != nil {
		return err
	}
	return n.settings.verifyRouteTable(values[1])
}
