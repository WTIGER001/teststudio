package azureworker

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

const preparationPushRole = "/providers/Microsoft.Authorization/roleDefinitions/8311e382-0749-4cb8-b61a-304f252e45ec"

type PreparationWriterSettings struct {
	Registry    PreparationRegistrySettings `json:"registry"`
	Identity    string                      `json:"identity"`
	ClientID    string                      `json:"clientId"`
	PrincipalID string                      `json:"principalId"`
	TenantID    string                      `json:"tenantId"`
	Assignment  string                      `json:"assignment"`
}

type PreparationSettings struct {
	Worker         Config                    `json:"worker"`
	Writer         PreparationWriterSettings `json:"writer"`
	ReaderClientID string                    `json:"readerClientId,omitempty"`
	Host           *PreparationHostSettings  `json:"host,omitempty"`
}

func (s PreparationSettings) ValidateNative() error {
	if !observerGUID(s.ReaderClientID) || s.ReaderClientID == s.Writer.ClientID || s.Host == nil {
		return hostedexecution.ErrConfiguration
	}
	writer, err := s.Writer.canonical(s.Worker)
	if err == nil {
		_, err = s.Host.canonical(s.Worker, writer, s.ReaderClientID)
	}
	return err
}

func (s PreparationWriterSettings) canonical(c Config) (PreparationWriterSettings, error) {
	var err error
	s.Registry, err = s.Registry.canonical(c)
	if err != nil || Validate(c) != nil || !c.ownsResource(s.Identity, "Microsoft.ManagedIdentity", "userAssignedIdentities") || strings.EqualFold(s.Identity, c.PullIdentity) || !observerGUID(s.ClientID) || !observerGUID(s.PrincipalID) || !observerGUID(s.TenantID) || s.ClientID == s.PrincipalID {
		return s, hostedexecution.ErrConfiguration
	}
	prefix := s.Registry.Registry.registry() + "/providers/Microsoft.Authorization/roleAssignments/"
	if !strings.HasPrefix(s.Assignment, prefix) || !observerGUID(strings.TrimPrefix(s.Assignment, prefix)) {
		return s, hostedexecution.ErrConfiguration
	}
	return s, nil
}

func LoadPreparationSettings(file, variable string, lookup func(string) (string, bool)) (*PreparationSettings, error) {
	var s PreparationSettings
	found, err := runtimeconfig.LoadJSON(file, variable, "Azure preparation", lookup, &s)
	if err != nil || !found {
		return nil, err
	}
	s.Writer, err = s.Writer.canonical(s.Worker)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (s PreparationWriterSettings) AssignmentDefinition(c Config) (a.RoleAssignment, error) {
	if _, err := s.canonical(c); err != nil {
		return a.RoleAssignment{}, err
	}
	return a.RoleAssignment{Properties: &a.RoleAssignmentProperties{PrincipalID: to.Ptr(s.PrincipalID), PrincipalType: to.Ptr(a.PrincipalTypeServicePrincipal), RoleDefinitionID: to.Ptr(observerScope(c) + preparationPushRole)}}, nil
}
