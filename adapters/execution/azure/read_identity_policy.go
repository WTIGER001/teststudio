package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

func (s ReadIdentitySettings) RoleDefinition(c Config) (a.RoleDefinition, error) {
	if _, err := s.canonical(c); err != nil {
		return a.RoleDefinition{}, err
	}
	return a.RoleDefinition{Properties: &a.RoleDefinitionProperties{RoleName: to.Ptr("TestStudio worker observer"), RoleType: to.Ptr("CustomRole"), AssignableScopes: []*string{to.Ptr(observerScope(c))}, Permissions: []*a.Permission{{Actions: []*string{to.Ptr("*/read"), to.Ptr(networkReadAssociationAction)}}}}}, nil
}

func (s ReadIdentitySettings) AssignmentDefinition(c Config) (a.RoleAssignment, error) {
	if _, err := s.canonical(c); err != nil {
		return a.RoleAssignment{}, err
	}
	return a.RoleAssignment{Properties: &a.RoleAssignmentProperties{PrincipalID: to.Ptr(s.PrincipalID), PrincipalType: to.Ptr(a.PrincipalTypeServicePrincipal), RoleDefinitionID: to.Ptr(s.Role)}}, nil
}
