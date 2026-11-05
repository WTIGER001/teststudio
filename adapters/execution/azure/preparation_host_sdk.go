package azureworker

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	r "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKPreparationHostReader struct {
	*sdkServiceReader
	settings    PreparationHostSettings
	storages    *a.ManagedEnvironmentsStoragesClient
	identities  *m.UserAssignedIdentitiesClient
	assignments *r.RoleAssignmentsClient
	roles       *r.RoleDefinitionsClient
}

func NewSDKPreparationHostReader(s PreparationSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKPreparationHostReader, error) {
	if s.validateHost() != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	c, h := s.Worker, *s.Host
	service, err := newSDKServiceReader(c, h.Environment, h.Application, nil, credential, transport)
	if err != nil {
		return nil, err
	}
	service.http.resources = map[string]string{h.Environment: exchangeVersion, h.Application: exchangeVersion, h.Application + "/revisions": exchangeVersion, h.storage(): exchangeVersion, h.ReaderIdentity: readIdentityVersion, h.ReaderAssignment: readAssignmentVersion, h.ReaderRole: readRoleVersion}
	service.http.pages = map[string]bool{h.Application + "/revisions": true}
	options := publicARMOptions(service.http)
	identities, err := m.NewUserAssignedIdentitiesClient(c.Subscription, credential, options)
	if err != nil {
		service.Close()
		return nil, hostedexecution.ErrConfiguration
	}
	factory, err := r.NewClientFactory(c.Subscription, credential, options)
	if err != nil {
		service.Close()
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKPreparationHostReader{sdkServiceReader: service, settings: h, storages: service.factory.NewManagedEnvironmentsStoragesClient(), identities: identities, assignments: factory.NewRoleAssignmentsClient(), roles: factory.NewRoleDefinitionsClient()}, nil
}

func (c *SDKPreparationHostReader) Storage(ctx context.Context, id string) (a.ManagedEnvironmentStorage, error) {
	if id != c.settings.storage() {
		return a.ManagedEnvironmentStorage{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, environment := resourceParts(c.settings.Environment)
	response, err := c.storages.Get(ctx, group, environment, c.settings.JournalStorageName, nil)
	if err != nil {
		return a.ManagedEnvironmentStorage{}, sdkError(err, false)
	}
	actual := response.ManagedEnvironmentStorage
	result := a.ManagedEnvironmentStorage{ID: actual.ID, Name: actual.Name, Type: actual.Type}
	if actual.Properties != nil {
		result.Properties = &a.ManagedEnvironmentStorageProperties{}
		if file := actual.Properties.AzureFile; file != nil {
			result.Properties.AzureFile = &a.AzureFileProperties{AccountName: file.AccountName, ShareName: file.ShareName, AccessMode: file.AccessMode}
		}
		if actual.Properties.NfsAzureFile != nil {
			result.Properties.NfsAzureFile = &a.NfsAzureFileProperties{}
		}
	}
	return result, nil
}

func (c *SDKPreparationHostReader) Identity(ctx context.Context, id string) (m.Identity, error) {
	if id != c.settings.ReaderIdentity {
		return m.Identity{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	group, name := resourceParts(id)
	response, err := c.identities.Get(ctx, group, name, nil)
	return response.Identity, sdkError(err, false)
}

func (c *SDKPreparationHostReader) Assignment(ctx context.Context, id string) (r.RoleAssignment, error) {
	if id != c.settings.ReaderAssignment {
		return r.RoleAssignment{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.assignments.GetByID(ctx, id, nil)
	return response.RoleAssignment, sdkError(err, false)
}

func (c *SDKPreparationHostReader) Role(ctx context.Context, id string) (r.RoleDefinition, error) {
	if id != c.settings.ReaderRole {
		return r.RoleDefinition{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.roles.Get(ctx, strings.Split(id, "/providers/")[0], id[strings.LastIndex(id, "/")+1:], nil)
	return response.RoleDefinition, sdkError(err, false)
}
