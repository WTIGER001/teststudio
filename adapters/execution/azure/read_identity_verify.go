package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	m "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyReadIdentity(c Config, s ReadIdentitySettings, identity m.Identity, assignment a.RoleAssignment, role a.RoleDefinition) error {
	if !perimeterIdentity(value(identity.ID), value(identity.Name), value(identity.Type), s.Identity, "Microsoft.ManagedIdentity/userAssignedIdentities", value(identity.Location), c.Region) {
		return hostedexecution.ErrOwnership
	}
	ip := identity.Properties
	if ip == nil || ip.ClientID == nil || ip.PrincipalID == nil || ip.TenantID == nil || assignment.Properties == nil || role.Properties == nil {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(ip.ClientID), s.ClientID) || !strings.EqualFold(value(ip.PrincipalID), s.PrincipalID) || !strings.EqualFold(value(ip.TenantID), s.TenantID) || !observerResourceIdentity(value(assignment.ID), value(assignment.Name), value(assignment.Type), s.Assignment, "roleAssignments") || !observerResourceIdentity(value(role.ID), value(role.Name), value(role.Type), s.Role, "roleDefinitions") {
		return hostedexecution.ErrOwnership
	}
	ap, rp := assignment.Properties, role.Properties
	if ap.PrincipalID == nil || ap.PrincipalType == nil || ap.RoleDefinitionID == nil || ap.Scope == nil || len(rp.Permissions) == 0 || len(rp.AssignableScopes) == 0 || rp.RoleType == nil {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(ap.PrincipalID), s.PrincipalID) || value(ap.PrincipalType) != a.PrincipalTypeServicePrincipal || !strings.EqualFold(value(ap.RoleDefinitionID), s.Role) || !strings.EqualFold(value(ap.Scope), observerScope(c)) || value(ap.Condition) != "" || value(ap.ConditionVersion) != "" || value(ap.DelegatedManagedIdentityResourceID) != "" || value(rp.RoleType) != "CustomRole" || !networkDenyActions(rp.AssignableScopes, observerScope(c)) || len(rp.Permissions) != 1 || rp.Permissions[0] == nil {
		return hostedexecution.ErrOwnership
	}
	permission := rp.Permissions[0]
	if !networkDenyActions(permission.Actions, "*/read", networkReadAssociationAction) || len(permission.NotActions)+len(permission.DataActions)+len(permission.NotDataActions) != 0 || value(permission.Condition) != "" || value(permission.ConditionVersion) != "" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func observerResourceIdentity(id, name, kind, expected, collection string) bool {
	parts := strings.Split(expected, "/")
	return strings.EqualFold(id, expected) && strings.EqualFold(name, parts[len(parts)-1]) && strings.EqualFold(kind, "Microsoft.Authorization/"+collection)
}
