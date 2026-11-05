package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const allDenyPrincipals = "00000000-0000-0000-0000-000000000000"
const networkReadAssociationAction = "Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action"

func verifyNetworkDeny(binding networkDenyBinding, actual a.DenyAssignment) error {
	return verifyResourceDeny(binding, actual, "*/read", "Microsoft.Authorization/locks/delete", networkReadAssociationAction)
}

func verifyResourceDeny(binding networkDenyBinding, actual a.DenyAssignment, exceptions ...string) error {
	parts := strings.Split(binding.assignment, "/")
	if !strings.EqualFold(value(actual.ID), binding.assignment) || !strings.EqualFold(value(actual.Name), parts[len(parts)-1]) || !strings.EqualFold(value(actual.Type), "Microsoft.Authorization/denyAssignments") {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if p == nil || p.Scope == nil || p.DenyAssignmentEffect == nil || p.DoNotApplyToChildScopes == nil || p.IsSystemProtected == nil || len(p.Permissions) == 0 || len(p.Principals) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(p.Scope), binding.scope) || value(p.DenyAssignmentEffect) != a.DenyAssignmentEffectEnforced || value(p.DoNotApplyToChildScopes) || !value(p.IsSystemProtected) || len(p.ExcludePrincipals) != 0 || value(p.Condition) != "" || value(p.ConditionVersion) != "" || len(p.Principals) != 1 || p.Principals[0] == nil || value(p.Principals[0].ID) != allDenyPrincipals || !strings.EqualFold(value(p.Principals[0].Type), "SystemDefined") || len(p.Permissions) != 1 || p.Permissions[0] == nil {
		return hostedexecution.ErrOwnership
	}
	permission := p.Permissions[0]
	if !networkDenyActions(permission.Actions, "*") || !networkDenyActions(permission.NotActions, exceptions...) || len(permission.DataActions)+len(permission.NotDataActions) != 0 || value(permission.Condition) != "" || value(permission.ConditionVersion) != "" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func networkDenyActions(actual []*string, wanted ...string) bool {
	if len(actual) != len(wanted) {
		return false
	}
	remaining := map[string]bool{}
	for _, action := range wanted {
		remaining[inventoryIdentityKey(action)] = true
	}
	for _, action := range actual {
		if action == nil || !remaining[inventoryIdentityKey(value(action))] {
			return false
		}
		delete(remaining, inventoryIdentityKey(value(action)))
	}
	return len(remaining) == 0
}
