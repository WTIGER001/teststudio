package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKPreparationWriterReader struct {
	settings    PreparationWriterSettings
	role        string
	identities  *m.UserAssignedIdentitiesClient
	assignments *a.RoleAssignmentsClient
	roles       *a.RoleDefinitionsClient
	http        *armReadHTTP
}

func NewSDKPreparationWriterReader(c Config, s PreparationWriterSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPreparationWriterReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	role := observerScope(c) + preparationPushRole
	h := newARMReadHTTP(c, map[string]string{s.Identity: readIdentityVersion, s.Assignment: readAssignmentVersion, role: readRoleVersion}, transport)
	options := publicARMOptions(h)
	identities, err := m.NewUserAssignedIdentitiesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	factory, err := a.NewClientFactory(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKPreparationWriterReader{settings: s, role: role, identities: identities, assignments: factory.NewRoleAssignmentsClient(), roles: factory.NewRoleDefinitionsClient(), http: h}, nil
}

func (c *SDKPreparationWriterReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKPreparationWriterReader) Identity(ctx context.Context, id string) (m.Identity, error) {
	if id != c.settings.Identity {
		return m.Identity{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.identities.Get(ctx, group, name, nil)
	return response.Identity, sdkError(err, false)
}

func (c *SDKPreparationWriterReader) Assignment(ctx context.Context, id string) (a.RoleAssignment, error) {
	if id != c.settings.Assignment {
		return a.RoleAssignment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.assignments.GetByID(ctx, id, nil)
	return response.RoleAssignment, sdkError(err, false)
}

func (c *SDKPreparationWriterReader) Role(ctx context.Context, id string) (a.RoleDefinition, error) {
	if id != c.role {
		return a.RoleDefinition{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.roles.Get(ctx, strings.Split(id, "/providers/")[0], strings.TrimPrefix(preparationPushRole, "/providers/Microsoft.Authorization/roleDefinitions/"), nil)
	return response.RoleDefinition, sdkError(err, false)
}
