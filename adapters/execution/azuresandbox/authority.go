package azuresandbox

import (
	"context"
	"sort"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKAuthority struct {
	*metadataReader
	settings Settings
	cache    metadataCache
	now      func() time.Time
}

func authorityVersions(s Settings) map[string]string {
	a := s.Authority
	versions := map[string]string{a.Control.Resource: "2024-11-30", a.Registry.Resource: "2024-11-30", a.Metadata.Resource: "2024-11-30", s.Timer.StopIdentity: "2024-11-30", a.RegistryAssignment: "2022-04-01"}
	for _, role := range s.allAuthorityRoles() {
		versions[role.grant.Role] = "2022-05-01-preview"
		versions[role.grant.Assignment] = "2022-04-01"
	}
	for id, version := range s.hostAccessVersions() {
		versions[id] = version
	}
	return versions
}

func NewSDKAuthority(s Settings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKAuthority, error) {
	if s.Validate() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	if s.Host.Access.Blob != nil {
		copy := *s.Host.Access.Blob
		s.Host.Access.Blob = &copy
	}
	if s.Host.Runtime != nil {
		copy := *s.Host.Runtime
		s.Host.Runtime = &copy
	}
	reader, err := newMetadataReader(authorityVersions(s), credential, transport)
	if err != nil {
		return nil, err
	}
	return &SDKAuthority{metadataReader: reader, settings: s, now: time.Now}, nil
}

func (c *SDKAuthority) Close() { c.http.Close() }

func (c *SDKAuthority) Check(ctx context.Context, config Config) error {
	if hash(config) != hash(c.settings.Worker) {
		return hostedexecution.ErrOwnership
	}
	return c.cache.check(ctx, c.now, c.http.closed.Load, c.read)
}

func (c *SDKAuthority) read(ctx context.Context) error {
	ids := make([]string, 0, len(c.http.allowed))
	for id := range c.http.allowed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	values, err := c.observe(ctx, ids)
	if err != nil {
		return err
	}
	resources := map[string]placementResource{}
	for i, value := range values {
		resources[ids[i]] = value
	}
	return c.settings.verifyAuthority(resources)
}
