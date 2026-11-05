package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func preparationRoleID(c Config, id string) bool {
	return strings.EqualFold(id, preparationPushRole) || strings.EqualFold(id, observerScope(c)+preparationPushRole)
}

func verifyPreparationWriter(c Config, s PreparationWriterSettings, identity m.Identity, assignment a.RoleAssignment, role a.RoleDefinition) error {
	if !perimeterIdentity(value(identity.ID), value(identity.Name), value(identity.Type), s.Identity, "Microsoft.ManagedIdentity/userAssignedIdentities", value(identity.Location), c.Region) || !observerResourceIdentity(value(assignment.ID), value(assignment.Name), value(assignment.Type), s.Assignment, "roleAssignments") {
		return hostedexecution.ErrOwnership
	}
	ip, ap, rp := identity.Properties, assignment.Properties, role.Properties
	if ip == nil || ip.ClientID == nil || ip.PrincipalID == nil || ip.TenantID == nil || ap == nil || ap.PrincipalID == nil || ap.PrincipalType == nil || ap.RoleDefinitionID == nil || ap.Scope == nil || rp == nil || rp.RoleType == nil || len(rp.Permissions) == 0 || len(rp.AssignableScopes) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(ip.ClientID), s.ClientID) || !strings.EqualFold(value(ip.PrincipalID), s.PrincipalID) || !strings.EqualFold(value(ip.TenantID), s.TenantID) || !strings.EqualFold(value(ap.PrincipalID), s.PrincipalID) || value(ap.PrincipalType) != a.PrincipalTypeServicePrincipal || !preparationRoleID(c, value(ap.RoleDefinitionID)) || !strings.EqualFold(value(ap.Scope), s.Registry.Registry.registry()) || value(ap.Condition) != "" || value(ap.ConditionVersion) != "" || value(ap.DelegatedManagedIdentityResourceID) != "" {
		return hostedexecution.ErrOwnership
	}
	if !preparationRoleID(c, value(role.ID)) || !observerResourceIdentity(value(role.ID), value(role.Name), value(role.Type), value(role.ID), "roleDefinitions") || value(rp.RoleType) != "BuiltInRole" || value(rp.RoleName) != "AcrPush" || !networkDenyActions(rp.AssignableScopes, "/") || len(rp.Permissions) != 1 || rp.Permissions[0] == nil {
		return hostedexecution.ErrOwnership
	}
	p := rp.Permissions[0]
	if !networkDenyActions(p.Actions, "Microsoft.ContainerRegistry/registries/pull/read", "Microsoft.ContainerRegistry/registries/push/write") || len(p.NotActions)+len(p.DataActions)+len(p.NotDataActions) != 0 || value(p.Condition) != "" || value(p.ConditionVersion) != "" {
		return hostedexecution.ErrOwnership
	}
	return nil
}
