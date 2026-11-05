package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const readDenyItems = 1024
const readDenyPages = 16

type SDKReadDenyReader struct {
	assignments *a.DenyAssignmentsClient
	path        string
	http        *armReadHTTP
}

func NewSDKReadDenyReader(c Config, credential azcore.TokenCredential, transport policy.Transporter) (*SDKReadDenyReader, error) {
	if Validate(c) != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	path := observerScope(c) + "/providers/Microsoft.Authorization/denyAssignments"
	h := newARMReadHTTP(c, map[string]string{path: networkDenyVersion}, transport)
	h.pages = map[string]bool{path: true}
	client, err := a.NewDenyAssignmentsClient(c.Subscription, credential, publicARMOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKReadDenyReader{assignments: client, path: path, http: h}, nil
}

func (c *SDKReadDenyReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKReadDenyReader) Assignments(ctx context.Context) ([]a.DenyAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pager := c.assignments.NewListPager(nil)
	return armInventory(ctx, c.http, c.path, networkDenyVersion, readDenyItems, readDenyPages, pager.More, func(ctx context.Context) ([]*a.DenyAssignment, *string, error) {
		page, err := pager.NextPage(ctx)
		if err == nil && page.Value == nil {
			return nil, nil, hostedexecution.ErrUnavailable
		}
		return page.Value, page.NextLink, err
	})
}
