package azureworker

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyReadDenies(c Config, s ReadIdentitySettings, assignments []a.DenyAssignment) error {
	if assignments == nil || len(assignments) > readDenyItems {
		return hostedexecution.ErrUnavailable
	}
	wanted := map[string]networkDenyBinding{}
	for _, binding := range append(s.Protection.bindings(), s.identityProtection()) {
		wanted[inventoryIdentityKey(binding.assignment)] = binding
	}
	seen := map[string]bool{}
	for _, assignment := range assignments {
		ancestor, err := readDenyIdentity(c, assignment)
		if err != nil {
			return err
		}
		key := inventoryIdentityKey(value(assignment.ID))
		if seen[key] {
			return hostedexecution.ErrOwnership
		}
		seen[key] = true
		if binding, ok := wanted[key]; ok {
			var err error
			if strings.EqualFold(binding.assignment, s.ProtectionAssignment) {
				err = verifyReadIdentityProtection(s, assignment)
			} else {
				err = verifyNetworkDeny(binding, assignment)
			}
			if err != nil {
				return err
			}
			delete(wanted, key)
		}
		if err := readDenyBlocks(s.PrincipalID, ancestor, assignment.Properties); err != nil {
			return err
		}
	}
	if len(wanted) != 0 {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func readDenyIdentity(c Config, assignment a.DenyAssignment) (bool, error) {
	p := assignment.Properties
	if p == nil || p.Scope == nil || p.DenyAssignmentEffect == nil || p.DoNotApplyToChildScopes == nil || len(p.Principals) == 0 || len(p.Principals) > 256 || len(p.ExcludePrincipals) > 256 || len(p.Permissions) == 0 || len(p.Permissions) > 32 {
		return false, hostedexecution.ErrUnavailable
	}
	scope := value(p.Scope)
	if len(scope) > 2048 || strings.ContainsAny(scope, "?#%\\ \t\r\n") || strings.Contains(scope, "//") {
		return false, hostedexecution.ErrOwnership
	}
	ancestor := scope == "/"
	if !ancestor {
		id, err := arm.ParseResourceID(scope)
		if err != nil || strings.HasSuffix(scope, "/") {
			return false, hostedexecution.ErrOwnership
		}
		parts := strings.Split(scope, "/")
		ancestor = len(parts) == 5 && strings.EqualFold(parts[1], "providers") && strings.EqualFold(parts[2], "Microsoft.Management") && strings.EqualFold(parts[3], "managementGroups") && groupName.MatchString(parts[4])
		if !ancestor && !strings.EqualFold(id.SubscriptionID, c.Subscription) {
			return false, hostedexecution.ErrOwnership
		}
	}
	name := value(assignment.Name)
	expected := strings.TrimSuffix(scope, "/") + "/providers/Microsoft.Authorization/denyAssignments/" + name
	if !observerGUID(strings.ToLower(name)) || !strings.EqualFold(value(assignment.ID), expected) || !strings.EqualFold(value(assignment.Type), "Microsoft.Authorization/denyAssignments") {
		return false, hostedexecution.ErrOwnership
	}
	return ancestor, nil
}

func readDenyBlocks(principal string, ancestor bool, p *a.DenyAssignmentProperties) error {
	effect := value(p.DenyAssignmentEffect)
	if effect != a.DenyAssignmentEffectEnforced && effect != a.DenyAssignmentEffectAudit {
		return hostedexecution.ErrUnavailable
	}
	if effect == a.DenyAssignmentEffectAudit || ancestor && value(p.DoNotApplyToChildScopes) {
		return nil
	}
	applies, err := readDenyPrincipals(p.Principals, principal, false)
	if err != nil {
		return err
	}
	excluded, err := readDenyPrincipals(p.ExcludePrincipals, principal, true)
	if err != nil {
		return err
	}
	if !applies || excluded {
		return nil
	}
	for _, permission := range p.Permissions {
		blocked, err := readDenyPermission(permission, readDenyAssociationScope(value(p.Scope)))
		if err != nil {
			return err
		}
		if blocked {
			if value(p.Condition) != "" || value(p.ConditionVersion) != "" {
				return hostedexecution.ErrUnavailable
			}
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}

func readDenyAssociationScope(scope string) bool {
	id, err := arm.ParseResourceID(scope)
	if err != nil || id.SubscriptionID == "" {
		return true
	}
	resourceType := id.ResourceType.String()
	return strings.EqualFold(resourceType, "Microsoft.Resources/subscriptions") || strings.EqualFold(resourceType, "Microsoft.Resources/resourceGroups") || strings.EqualFold(resourceType, "Microsoft.Network/virtualNetworks")
}

func readDenyPrincipals(principals []*a.DenyAssignmentPrincipal, target string, exclusions bool) (bool, error) {
	matched := false
	for _, principal := range principals {
		if principal == nil || !guid.MatchString(strings.ToLower(value(principal.ID))) {
			return false, hostedexecution.ErrUnavailable
		}
		id := value(principal.ID)
		switch strings.ToLower(value(principal.Type)) {
		case "systemdefined":
			if id != allDenyPrincipals {
				return false, hostedexecution.ErrUnavailable
			}
			matched = true
		case "group", "foreigngroup":
			if id == allDenyPrincipals {
				return false, hostedexecution.ErrUnavailable
			}
			matched = matched || !exclusions
		case "serviceprincipal", "user", "device":
			if id == allDenyPrincipals || strings.EqualFold(id, target) && !strings.EqualFold(value(principal.Type), "ServicePrincipal") {
				return false, hostedexecution.ErrUnavailable
			}
			matched = matched || strings.EqualFold(id, target)
		default:
			return false, hostedexecution.ErrUnavailable
		}
	}
	return matched, nil
}
