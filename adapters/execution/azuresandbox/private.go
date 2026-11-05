package azuresandbox

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	p "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKPrivate struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func privateVersions(s Settings) map[string]string {
	p := s.Placement.Private
	versions := map[string]string{p.RegistryEndpoint: "2025-03-01", p.RegistryInterface: "2025-03-01", p.ExchangeEndpoint: "2025-03-01", p.ExchangeInterface: "2025-03-01", p.Subnet: "2025-03-01", p.SecurityGroup: "2025-03-01"}
	for _, zone := range p.zoneBindings(s) {
		for _, suffix := range []string{"", "/ALL", "/virtualNetworkLinks"} {
			versions[zone.ID+suffix] = "2024-06-01"
		}
	}
	return versions
}

func NewSDKPrivate(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPrivate, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	s.Placement.Network.PrivateEndpointIPs = append([]string(nil), s.Placement.Network.PrivateEndpointIPs...)
	reader, err := newMetadataReader(privateVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	reader.http.pages = map[string]bool{}
	for _, zone := range s.Placement.Private.zoneBindings(s) {
		reader.http.pages[zone.ID+"/ALL"] = true
		reader.http.pages[zone.ID+"/virtualNetworkLinks"] = true
	}
	return &SDKPrivate{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKPrivate) Close() { c.http.Close() }

func (c *SDKPrivate) Check(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	if ctx.Err() != nil || c.http.closed.Load() {
		return hostedexecution.ErrUnavailable
	}
	epoch, started := c.cache.begin(c.now)
	err := c.read(ctx)
	c.cache.remember(epoch, started, err, c.http.closed.Load)
	return err
}

func (c *SDKPrivate) CheckRunning(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func resourceModel[T any](value placementResource) (T, error) {
	var result T
	data, err := json.Marshal(value)
	if err != nil || json.Unmarshal(data, &result) != nil {
		return result, hostedexecution.ErrUnavailable
	}
	return result, nil
}

func resourceModels[T any](values []placementResource) ([]T, error) {
	result := make([]T, 0, len(values))
	for _, value := range values {
		model, err := resourceModel[T](value)
		if err != nil {
			return nil, err
		}
		result = append(result, model)
	}
	return result, nil
}

func (c *SDKPrivate) read(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setting := c.settings.Placement.Private
	var resources []placementResource
	var errors [3]error
	var group sync.WaitGroup
	group.Go(func() {
		resources, errors[0] = c.observe(ctx, []string{setting.RegistryEndpoint, setting.RegistryInterface, setting.ExchangeEndpoint, setting.ExchangeInterface, setting.Subnet, setting.SecurityGroup})
	})
	for i, zone := range setting.zoneBindings(c.settings) {
		group.Go(func() { errors[i+1] = c.readPrivateZone(ctx, zone) })
	}
	group.Wait()
	if ctx.Err() != nil {
		return hostedexecution.ErrUnavailable
	}
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	interfaces := []n.Interface{}
	for i, binding := range setting.endpointBindings(c.settings) {
		endpoint, err := resourceModel[n.PrivateEndpoint](resources[i*2])
		if err != nil {
			return err
		}
		nic, err := resourceModel[n.Interface](resources[i*2+1])
		if err != nil {
			return err
		}
		if err := binding.Verify(endpoint, nic); err != nil {
			return err
		}
		interfaces = append(interfaces, nic)
	}
	subnet, err := resourceModel[n.Subnet](resources[4])
	if err != nil {
		return err
	}
	if err := azureworker.VerifyPrivateEndpointSubnet(subnet, interfaces, setting.Subnet, setting.SubnetPrefix, setting.SecurityGroup, []string{setting.RegistryEndpoint, setting.ExchangeEndpoint}); err != nil {
		return err
	}
	return verifyNetworkGroup(resources[5], setting.SecurityGroup, setting.Subnet, c.settings.Worker.Region, setting.rules(c.settings))
}

func (c *metadataReader) readPrivateZone(ctx context.Context, binding azureworker.PrivateZoneBinding) error {
	var zone placementResource
	var records, links []placementResource
	var errors [3]error
	var group sync.WaitGroup
	group.Go(func() { zone, errors[0] = c.get(ctx, binding.ID) })
	group.Go(func() { records, errors[1] = c.inventory(ctx, binding.ID+"/ALL", len(binding.Records)+1) })
	group.Go(func() { links, errors[2] = c.inventory(ctx, binding.ID+"/virtualNetworkLinks", 1) })
	group.Wait()
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	z, err := resourceModel[p.PrivateZone](zone)
	if err != nil {
		return err
	}
	r, err := resourceModels[p.RecordSet](records)
	if err != nil {
		return err
	}
	l, err := resourceModels[p.VirtualNetworkLink](links)
	if err != nil {
		return err
	}
	return binding.Verify(z, r, l)
}
