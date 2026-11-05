package azuresandbox

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s Settings) verifyAuthorityIdentity(r placementResource, resource, client, principal string) error {
	if !namedResource(r, resource, "Microsoft.ManagedIdentity/userAssignedIdentities") || !strings.EqualFold(r.Location, s.Worker.Region) {
		return hostedexecution.ErrOwnership
	}
	var p struct {
		Client    string `json:"clientId"`
		Principal string `json:"principalId"`
		Tenant    string `json:"tenantId"`
	}
	if err := properties(r, &p); err != nil {
		return err
	}
	if p.Client == "" || p.Principal == "" || p.Tenant == "" {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(p.Client, client) || !strings.EqualFold(p.Principal, principal) || !strings.EqualFold(p.Tenant, s.TenantID) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func exactActions(actual, want []string) bool {
	if len(actual) != len(want) {
		return false
	}
	seen := map[string]bool{}
	for _, value := range actual {
		key := strings.ToLower(value)
		if seen[key] {
			return false
		}
		seen[key] = true
	}
	for _, value := range want {
		if !seen[strings.ToLower(value)] {
			return false
		}
	}
	return true
}

func verifyAuthorityAssignment(r placementResource, id, scope, principal, role string) error {
	if !namedResource(r, id, "Microsoft.Authorization/roleAssignments") {
		return hostedexecution.ErrOwnership
	}
	var p struct {
		Principal string `json:"principalId"`
		Type      string `json:"principalType"`
		Scope     string `json:"scope"`
		Role      string `json:"roleDefinitionId"`
		Condition string `json:"condition"`
		Version   string `json:"conditionVersion"`
		Delegated string `json:"delegatedManagedIdentityResourceId"`
	}
	if err := properties(r, &p); err != nil {
		return err
	}
	roleMatches := strings.EqualFold(p.Role, role) || strings.HasSuffix(strings.ToLower(role), strings.ToLower(pullRole)) && strings.EqualFold(p.Role, pullRole)
	if !strings.EqualFold(p.Principal, principal) || p.Type != "ServicePrincipal" || !strings.EqualFold(p.Scope, scope) || !roleMatches || p.Condition != "" || p.Version != "" || p.Delegated != "" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func verifyAuthorityRole(r placementResource, expected authorityRole) error {
	if !namedResource(r, expected.grant.Role, "Microsoft.Authorization/roleDefinitions") {
		return hostedexecution.ErrOwnership
	}
	var p struct {
		Type        string   `json:"type"`
		Name        string   `json:"roleName"`
		Scopes      []string `json:"assignableScopes"`
		Permissions []struct {
			Actions    []string `json:"actions"`
			NotActions []string `json:"notActions"`
			Data       []string `json:"dataActions"`
			NotData    []string `json:"notDataActions"`
			Condition  string   `json:"condition"`
			Version    string   `json:"conditionVersion"`
		} `json:"permissions"`
	}
	if err := properties(r, &p); err != nil {
		return err
	}
	if p.Type != "CustomRole" || p.Name != expected.name || !exactActions(p.Scopes, []string{expected.assignable}) || len(p.Permissions) != 1 {
		return hostedexecution.ErrOwnership
	}
	permission := p.Permissions[0]
	if !exactActions(permission.Actions, expected.actions) || !exactActions(permission.Data, expected.data) || len(permission.NotActions)+len(permission.NotData) != 0 || permission.Condition != "" || permission.Version != "" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (s Settings) verifyAuthority(resources map[string]placementResource) error {
	a := s.Authority
	identities := []struct{ resource, client, principal string }{
		{a.Control.Resource, s.ControlClientID, a.Control.PrincipalID}, {a.Registry.Resource, s.RegistryClientID, a.Registry.PrincipalID},
		{a.Metadata.Resource, s.MetadataClientID, a.Metadata.PrincipalID}, {s.Timer.StopIdentity, a.StopClientID, a.StopPrincipalID},
	}
	for _, identity := range identities {
		if err := s.verifyAuthorityIdentity(resources[identity.resource], identity.resource, identity.client, identity.principal); err != nil {
			return err
		}
	}
	for _, role := range s.authorityRoles() {
		if err := verifyAuthorityRole(resources[role.grant.Role], role); err != nil {
			return err
		}
		if err := verifyAuthorityAssignment(resources[role.grant.Assignment], role.grant.Assignment, role.scope, role.principal, role.grant.Role); err != nil {
			return err
		}
	}
	return verifyAuthorityAssignment(resources[a.RegistryAssignment], a.RegistryAssignment, s.Placement.Registry, a.Registry.PrincipalID, "/subscriptions/"+s.Worker.Subscription+pullRole)
}
