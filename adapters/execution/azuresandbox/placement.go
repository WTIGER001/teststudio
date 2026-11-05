package azuresandbox

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type placementResource struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	Location         string          `json:"location"`
	Kind             string          `json:"kind,omitempty"`
	ManagedBy        string          `json:"managedBy,omitempty"`
	Identity         json.RawMessage `json:"identity"`
	ExtendedLocation json.RawMessage `json:"extendedLocation,omitempty"`
	Properties       json.RawMessage `json:"properties"`
	SKU              struct {
		Name string  `json:"name"`
		Tier *string `json:"tier,omitempty"`
	} `json:"sku"`
}

type SDKPlacement struct {
	*metadataReader
	settings Settings
	now      func() time.Time
	cache    metadataCache
}

func NewSDKPlacement(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPlacement, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := newMetadataReader(placementVersions(s), credential, transport)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKPlacement{settings: s, metadataReader: reader, now: time.Now}, nil
}

func (c *SDKPlacement) Close() { c.http.Close() }

func (c *SDKPlacement) Check(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	if ctx.Err() != nil || c.http.closed.Load() {
		return hostedexecution.ErrUnavailable
	}
	epoch, started := c.beginPlacement()
	ids := []string{config.groupID(), config.connectionID(), c.settings.Placement.Subnet, c.settings.PullIdentity, c.settings.Placement.Registry, c.settings.Placement.PullAssignment, "/subscriptions/" + config.Subscription + pullRole}
	values, err := c.observe(ctx, config, ids)
	if err == nil {
		err = c.settings.verifyPlacement(values)
	}
	c.rememberPlacement(epoch, started, err)
	return err
}

func (c *SDKPlacement) CheckRunning(ctx context.Context, config Config) error {
	return c.runningPlacement(ctx, config)
}

func (c *SDKPlacement) observe(ctx context.Context, config Config, ids []string) ([]placementResource, error) {
	if hash(config) != hash(c.settings.Worker) {
		return nil, hostedexecution.ErrOwnership
	}
	if ctx.Err() != nil || c.http.closed.Load() {
		return nil, hostedexecution.ErrUnavailable
	}
	return c.metadataReader.observe(ctx, ids)
}

func (c *metadataReader) observe(ctx context.Context, ids []string) ([]placementResource, error) {
	if ctx.Err() != nil || c.http.closed.Load() {
		return nil, hostedexecution.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	values := make([]placementResource, len(ids))
	errors := make([]error, len(ids))
	var group sync.WaitGroup
	for i, id := range ids {
		group.Go(func() { values[i], errors[i] = c.get(ctx, id) })
	}
	group.Wait()
	if ctx.Err() != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

type placementAdmission struct {
	placement        *SDKPlacement
	authority        *SDKAuthority
	network          *SDKNetwork
	private          *SDKPrivate
	topology         *SDKTopology
	resolver         *SDKResolver
	services         *SDKServices
	backendEndpoint  *SDKBackendEndpoint
	relayNetwork     *SDKRelayNetwork
	registryService  *SDKRegistryService
	protection       Admission
	importProtection ImageAdmission
}

func (a *placementAdmission) BeforeCreate(ctx context.Context, c Config) error {
	if err := a.placement.Check(ctx, c); err != nil {
		return err
	}
	if err := a.network.Check(ctx, c); err != nil {
		return err
	}
	if err := a.private.Check(ctx, c); err != nil {
		return err
	}
	if err := a.topology.Check(ctx, c); err != nil {
		return err
	}
	if err := a.resolver.Check(ctx, c); err != nil {
		return err
	}
	if err := a.services.Check(ctx, c); err != nil {
		return err
	}
	if err := a.backendEndpoint.Check(ctx, c); err != nil {
		return err
	}
	if err := a.relayNetwork.Check(ctx, c); err != nil {
		return err
	}
	if err := a.registryService.Check(ctx, c); err != nil {
		return err
	}
	if err := a.authority.Check(ctx, c); err != nil {
		return err
	}
	return a.protection.BeforeCreate(ctx, c)
}

func (a *placementAdmission) VerifyInstance(ctx context.Context, c Config, actual Snapshot) error {
	if !identifier.MatchString(actual.ID) || !checksum.MatchString(actual.DefinitionHash) || actual.State != "Running" {
		return hostedexecution.ErrOwnership
	}
	if err := a.placement.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.network.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.private.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.topology.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.resolver.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.services.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.backendEndpoint.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.relayNetwork.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.registryService.CheckRunning(ctx, c); err != nil {
		return err
	}
	if err := a.authority.Check(ctx, c); err != nil {
		return err
	}
	return a.protection.VerifyInstance(ctx, c, actual)
}

func (a *placementAdmission) BeforeImport(ctx context.Context, c Config, identity string) error {
	if identity != a.placement.settings.PullIdentity || a.importProtection == nil {
		return hostedexecution.ErrOwnership
	}
	if err := a.placement.Check(ctx, c); err != nil {
		return err
	}
	if err := a.network.Check(ctx, c); err != nil {
		return err
	}
	if err := a.private.Check(ctx, c); err != nil {
		return err
	}
	if err := a.topology.Check(ctx, c); err != nil {
		return err
	}
	if err := a.resolver.Check(ctx, c); err != nil {
		return err
	}
	if err := a.services.Check(ctx, c); err != nil {
		return err
	}
	if err := a.backendEndpoint.Check(ctx, c); err != nil {
		return err
	}
	if err := a.relayNetwork.Check(ctx, c); err != nil {
		return err
	}
	if err := a.registryService.Check(ctx, c); err != nil {
		return err
	}
	if err := a.authority.Check(ctx, c); err != nil {
		return err
	}
	return a.importProtection.BeforeImport(ctx, c, identity)
}
