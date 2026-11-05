package azuresandbox

import (
	"encoding/json"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func namedResource(r placementResource, id, kind string) bool {
	name := id[strings.LastIndex(id, "/")+1:]
	return strings.EqualFold(r.ID, id) && (r.Name == name || r.Name == strings.Join(strings.Split(id, "/")[8:], "/")) && strings.EqualFold(r.Type, kind)
}

func properties(r placementResource, value any) error {
	if len(r.Properties) == 0 || string(r.Properties) == "null" || json.Unmarshal(r.Properties, value) != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func provisioned(state string) error {
	if state != "Succeeded" {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (s Settings) verifyPlacement(values []placementResource) error {
	if len(values) != 7 {
		return hostedexecution.ErrUnavailable
	}
	checks := []func() error{
		func() error { return s.verifyGroup(values[0]) }, func() error { return s.verifyConnection(values[1]) }, func() error { return s.verifySubnet(values[2]) },
		func() error { return s.verifyPullIdentity(values[3]) }, func() error { return s.verifyRegistry(values[4]) }, func() error { return s.verifyPullAssignment(values[5]) }, func() error { return s.verifyPullRole(values[6]) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (s Settings) verifyGroup(r placementResource) error {
	if !namedResource(r, s.Worker.groupID(), "Microsoft.App/sandboxGroups") || !strings.EqualFold(r.Location, s.Worker.Region) {
		return hostedexecution.ErrOwnership
	}
	var props map[string]json.RawMessage
	if err := properties(r, &props); err != nil {
		return err
	}
	for key := range props {
		switch key {
		case "provisioningState", "defaultCpu", "defaultMemory", "defaultDisk", "maxSandboxCount", "defaultTimeoutSeconds":
		default:
			return hostedexecution.ErrOwnership
		}
	}
	var state string
	if json.Unmarshal(props["provisioningState"], &state) != nil || provisioned(state) != nil {
		return hostedexecution.ErrUnavailable
	}
	var identity struct {
		Type        string `json:"type"`
		PrincipalID string `json:"principalId"`
		TenantID    string `json:"tenantId"`
		Users       map[string]struct {
			ClientID    string `json:"clientId"`
			PrincipalID string `json:"principalId"`
		} `json:"userAssignedIdentities"`
	}
	if json.Unmarshal(r.Identity, &identity) != nil {
		return hostedexecution.ErrUnavailable
	}
	if identity.Type != "UserAssigned" || identity.PrincipalID != "" || identity.TenantID != "" && !strings.EqualFold(identity.TenantID, s.TenantID) || len(identity.Users) != 1 {
		return hostedexecution.ErrOwnership
	}
	for id, user := range identity.Users {
		if !strings.EqualFold(id, s.PullIdentity) || !strings.EqualFold(user.ClientID, s.Placement.PullClientID) || !strings.EqualFold(user.PrincipalID, s.Placement.PullPrincipalID) {
			return hostedexecution.ErrOwnership
		}
	}
	return nil
}

func (s Settings) verifyConnection(r placementResource) error {
	if !namedResource(r, s.Worker.connectionID(), "Microsoft.App/sandboxGroups/vnetConnections") || !strings.EqualFold(r.Location, s.Worker.Region) {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		Subnet string `json:"subnetId"`
		State  string `json:"provisioningState"`
	}
	if err := properties(r, &props); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(r.Properties, &fields) != nil {
		return hostedexecution.ErrUnavailable
	}
	for key := range fields {
		if key != "subnetId" && key != "provisioningState" {
			return hostedexecution.ErrOwnership
		}
	}
	if !strings.EqualFold(props.Subnet, s.Placement.Subnet) {
		return hostedexecution.ErrOwnership
	}
	return provisioned(props.State)
}

func (s Settings) verifySubnet(r placementResource) error {
	p := s.Placement
	if !namedResource(r, p.Subnet, "Microsoft.Network/virtualNetworks/subnets") {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		State         string   `json:"provisioningState"`
		Prefix        string   `json:"addressPrefix"`
		Prefixes      []string `json:"addressPrefixes"`
		SecurityGroup struct {
			ID string `json:"id"`
		} `json:"networkSecurityGroup"`
		RouteTable struct {
			ID string `json:"id"`
		} `json:"routeTable"`
		NAT         *struct{}         `json:"natGateway"`
		Outbound    *bool             `json:"defaultOutboundAccess"`
		Endpoints   []json.RawMessage `json:"privateEndpoints"`
		Services    []json.RawMessage `json:"serviceEndpoints"`
		Delegations []struct {
			Properties struct {
				Service string `json:"serviceName"`
			} `json:"properties"`
		} `json:"delegations"`
	}
	if err := properties(r, &props); err != nil {
		return err
	}
	if err := provisioned(props.State); err != nil {
		return err
	}
	if props.Prefix == "" && len(props.Prefixes) == 1 {
		props.Prefix = props.Prefixes[0]
		props.Prefixes = nil
	}
	if props.Prefix != p.SubnetPrefix || len(props.Prefixes) != 0 || !strings.EqualFold(props.SecurityGroup.ID, p.SecurityGroup) || !strings.EqualFold(props.RouteTable.ID, p.RouteTable) || props.NAT != nil || props.Outbound == nil || *props.Outbound || len(props.Endpoints) != 0 || len(props.Services) != 0 || len(props.Delegations) != 1 || props.Delegations[0].Properties.Service != "Microsoft.App/environments" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (s Settings) verifyPullIdentity(r placementResource) error {
	if !namedResource(r, s.PullIdentity, "Microsoft.ManagedIdentity/userAssignedIdentities") || !strings.EqualFold(r.Location, s.Worker.Region) {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		ClientID    string `json:"clientId"`
		PrincipalID string `json:"principalId"`
		TenantID    string `json:"tenantId"`
	}
	if err := properties(r, &props); err != nil {
		return err
	}
	if props.ClientID == "" || props.PrincipalID == "" || props.TenantID == "" {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(props.ClientID, s.Placement.PullClientID) || !strings.EqualFold(props.PrincipalID, s.Placement.PullPrincipalID) || !strings.EqualFold(props.TenantID, s.TenantID) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (s Settings) verifyRegistry(r placementResource) error {
	if !namedResource(r, s.Placement.Registry, "Microsoft.ContainerRegistry/registries") || !strings.EqualFold(r.Location, s.Worker.Region) || r.SKU.Name != "Premium" {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		State     string `json:"provisioningState"`
		Login     string `json:"loginServer"`
		Public    string `json:"publicNetworkAccess"`
		Bypass    string `json:"networkRuleBypassOptions"`
		Mode      string `json:"roleAssignmentMode"`
		Admin     *bool  `json:"adminUserEnabled"`
		Anonymous *bool  `json:"anonymousPullEnabled"`
		Data      *bool  `json:"dataEndpointEnabled"`
	}
	if err := properties(r, &props); err != nil {
		return err
	}
	if err := provisioned(props.State); err != nil {
		return err
	}
	if props.Login != s.Worker.Registry || props.Public != "Disabled" || props.Bypass != "None" || props.Mode != "LegacyRegistryPermissions" || props.Admin == nil || *props.Admin || props.Anonymous == nil || *props.Anonymous || props.Data == nil || !*props.Data {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (s Settings) verifyPullAssignment(r placementResource) error {
	if !namedResource(r, s.Placement.PullAssignment, "Microsoft.Authorization/roleAssignments") {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		PrincipalID   string `json:"principalId"`
		PrincipalType string `json:"principalType"`
		Scope         string `json:"scope"`
		Role          string `json:"roleDefinitionId"`
		Condition     string `json:"condition"`
		Version       string `json:"conditionVersion"`
		Delegated     string `json:"delegatedManagedIdentityResourceId"`
	}
	if err := properties(r, &props); err != nil {
		return err
	}
	if !strings.EqualFold(props.PrincipalID, s.Placement.PullPrincipalID) || props.PrincipalType != "ServicePrincipal" || !strings.EqualFold(props.Scope, s.Placement.Registry) || !s.pullRoleID(props.Role) || props.Condition != "" || props.Version != "" || props.Delegated != "" {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (s Settings) pullRoleID(id string) bool {
	return strings.EqualFold(id, pullRole) || strings.EqualFold(id, "/subscriptions/"+s.Worker.Subscription+pullRole)
}

func (s Settings) verifyPullRole(r placementResource) error {
	if !s.pullRoleID(r.ID) || !namedResource(r, r.ID, "Microsoft.Authorization/roleDefinitions") {
		return hostedexecution.ErrOwnership
	}
	var props struct {
		Type        string   `json:"type"`
		Name        string   `json:"roleName"`
		Scopes      []string `json:"assignableScopes"`
		Permissions []struct {
			Actions        []string `json:"actions"`
			NotActions     []string `json:"notActions"`
			DataActions    []string `json:"dataActions"`
			NotDataActions []string `json:"notDataActions"`
			Condition      string   `json:"condition"`
			Version        string   `json:"conditionVersion"`
		} `json:"permissions"`
	}
	if err := properties(r, &props); err != nil {
		return err
	}
	if props.Type != "BuiltInRole" || props.Name != "AcrPull" || len(props.Scopes) != 1 || props.Scopes[0] != "/" || len(props.Permissions) != 1 {
		return hostedexecution.ErrOwnership
	}
	p := props.Permissions[0]
	if len(p.Actions) != 1 || !strings.EqualFold(p.Actions[0], "Microsoft.ContainerRegistry/registries/pull/read") || len(p.NotActions)+len(p.DataActions)+len(p.NotDataActions) != 0 || p.Condition != "" || p.Version != "" {
		return hostedexecution.ErrOwnership
	}
	return nil
}
