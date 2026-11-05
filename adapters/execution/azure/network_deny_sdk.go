package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const networkDenyVersion = "2024-07-01-preview"

type SDKNetworkDenyReader struct {
	client *a.DenyAssignmentsClient
	http   *armReadHTTP
}

func NewSDKNetworkDenyReader(c Config, s NetworkDenySettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKNetworkDenyReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	resources := map[string]string{}
	for _, binding := range s.bindings() {
		resources[binding.assignment] = networkDenyVersion
	}
	h := newARMReadHTTP(c, resources, transport)
	client, err := a.NewDenyAssignmentsClient(c.Subscription, credential, publicARMOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKNetworkDenyReader{client: client, http: h}, nil
}

func (c *SDKNetworkDenyReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKNetworkDenyReader) Assignment(ctx context.Context, id string) (a.DenyAssignment, error) {
	if !c.http.allows(id, networkDenyVersion) {
		return a.DenyAssignment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.client.GetByID(ctx, id, nil)
	if err != nil {
		return a.DenyAssignment{}, sdkError(err, false)
	}
	return response.DenyAssignment, nil
}
