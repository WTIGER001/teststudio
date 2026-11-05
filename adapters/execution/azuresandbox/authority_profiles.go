package azuresandbox

import "strings"

type authorityRole struct {
	grant      AuthorityGrant
	scope      string
	assignable string
	principal  string
	name       string
	actions    []string
	data       []string
}

func sandboxActions(names ...string) []string {
	values := make([]string, len(names))
	for i, name := range names {
		values[i] = "Microsoft.App/sandboxGroups/" + name
	}
	return values
}

func (s Settings) authorityRoles() []authorityRole {
	a := s.Authority
	group := s.Worker.groupID()
	timers := "/subscriptions/" + s.Worker.Subscription + "/resourceGroups/" + s.Timer.ResourceGroup
	subscription := "/subscriptions/" + s.Worker.Subscription
	controller := sandboxActions("diskimages/read", "sandboxes/read", "sandboxes/write", "sandboxes/delete", "sandboxes/executeCommand/action", "sandboxes/stop/action")
	if s.Mode == "discovery" {
		controller = append(controller, sandboxActions("diskimages/write")...)
	}
	name := "TestStudio " + hash(group)[:12] + " "
	return []authorityRole{
		{a.ControlGrant, group, resourceGroupID(group), a.Control.PrincipalID, name + s.Mode, nil, controller},
		{a.StopGrant, group, resourceGroupID(group), a.StopPrincipalID, name + "deadline stop", nil, sandboxActions("sandboxes/read", "sandboxes/stop/action")},
		{a.WorkflowGrant, timers, timers, a.Control.PrincipalID, name + "deadline administration", []string{"Microsoft.Logic/workflows/read", "Microsoft.Logic/workflows/write", "Microsoft.Logic/workflows/delete"}, nil},
		{a.AssignGrant, s.Timer.StopIdentity, resourceGroupID(s.Timer.StopIdentity), a.Control.PrincipalID, name + "deadline identity", []string{"Microsoft.ManagedIdentity/userAssignedIdentities/read", "Microsoft.ManagedIdentity/userAssignedIdentities/assign/action"}, nil},
		{a.MetadataGrant, subscription, subscription, a.Metadata.PrincipalID, name + "metadata", []string{"*/read", "Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action"}, nil},
	}
}

func (s Settings) AuthorityDefinitions() (map[string]any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	result := map[string]any{}
	for _, role := range s.authorityRoles() {
		permission := map[string]any{"actions": nonNilStrings(role.actions), "notActions": []string{}, "dataActions": nonNilStrings(role.data), "notDataActions": []string{}}
		properties := map[string]any{"roleName": role.name, "type": "CustomRole", "assignableScopes": []string{role.assignable}, "permissions": []any{permission}}
		result[role.grant.Role] = map[string]any{"id": role.grant.Role, "name": role.grant.Role[strings.LastIndex(role.grant.Role, "/")+1:], "type": "Microsoft.Authorization/roleDefinitions", "properties": properties}
		result[role.grant.Assignment] = assignmentDefinition(role.grant.Assignment, role.scope, role.principal, role.grant.Role)
	}
	result[s.Authority.RegistryAssignment] = assignmentDefinition(s.Authority.RegistryAssignment, s.Placement.Registry, s.Authority.Registry.PrincipalID, "/subscriptions/"+s.Worker.Subscription+pullRole)
	return result, nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string{}, values...)
}

func assignmentDefinition(id, scope, principal, role string) map[string]any {
	return map[string]any{"id": id, "name": id[strings.LastIndex(id, "/")+1:], "type": "Microsoft.Authorization/roleAssignments", "properties": map[string]any{"principalId": principal, "principalType": "ServicePrincipal", "roleDefinitionId": role, "scope": scope}}
}
