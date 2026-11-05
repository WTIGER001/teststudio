package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeploymentstacks/v2"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const readIdentityVersion = "2024-11-30"
const readRoleVersion = "2022-05-01-preview"
const readAssignmentVersion = "2022-04-01"

type SDKReadIdentityReader struct {
	settings    ReadIdentitySettings
	identities  *m.UserAssignedIdentitiesClient
	federation  *m.FederatedIdentityCredentialsClient
	assignments *a.RoleAssignmentsClient
	roles       *a.RoleDefinitionsClient
	denies      *a.DenyAssignmentsClient
	stacks      *d.Client
	http        *armReadHTTP
}

func NewSDKReadIdentityReader(c Config, s ReadIdentitySettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKReadIdentityReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	h := newARMReadHTTP(c, map[string]string{s.Identity: readIdentityVersion, s.Identity + "/federatedIdentityCredentials": readIdentityVersion, s.Assignment: readAssignmentVersion, s.Role: readRoleVersion, s.ProtectionAssignment: networkDenyVersion, s.NetworkProtectionStack: protectionStackVersion, s.IdentityProtectionStack: protectionStackVersion}, transport)
	options := publicARMOptions(h)
	identities, err := m.NewUserAssignedIdentitiesClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	stacks, err := d.NewClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	federation, err := m.NewFederatedIdentityCredentialsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	factory, err := a.NewClientFactory(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKReadIdentityReader{settings: s, identities: identities, federation: federation, assignments: factory.NewRoleAssignmentsClient(), roles: factory.NewRoleDefinitionsClient(), denies: factory.NewDenyAssignmentsClient(), stacks: stacks, http: h}, nil
}

func (c *SDKReadIdentityReader) Close() { c.http.CloseIdleConnections() }

func (c *SDKReadIdentityReader) Identity(ctx context.Context, id string) (m.Identity, error) {
	if id != c.settings.Identity {
		return m.Identity{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.identities.Get(ctx, group, name, nil)
	return response.Identity, sdkError(err, false)
}

func (c *SDKReadIdentityReader) Assignment(ctx context.Context, id string) (a.RoleAssignment, error) {
	if id != c.settings.Assignment {
		return a.RoleAssignment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.assignments.GetByID(ctx, id, nil)
	return response.RoleAssignment, sdkError(err, false)
}

func (c *SDKReadIdentityReader) Role(ctx context.Context, id string) (a.RoleDefinition, error) {
	if id != c.settings.Role {
		return a.RoleDefinition{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	name := id[strings.LastIndex(id, "/")+1:]
	response, err := c.roles.Get(ctx, strings.Split(id, "/providers/")[0], name, nil)
	return response.RoleDefinition, sdkError(err, false)
}
