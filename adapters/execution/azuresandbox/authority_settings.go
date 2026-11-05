package azuresandbox

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type AuthorityIdentity struct {
	Resource    string `json:"resource"`
	PrincipalID string `json:"principalId"`
}

type AuthorityGrant struct {
	Assignment string `json:"assignment"`
	Role       string `json:"role"`
}

type AuthoritySettings struct {
	Control            AuthorityIdentity `json:"control"`
	Registry           AuthorityIdentity `json:"registry"`
	Metadata           AuthorityIdentity `json:"metadata"`
	StopClientID       string            `json:"stopClientId"`
	StopPrincipalID    string            `json:"stopPrincipalId"`
	ControlGrant       AuthorityGrant    `json:"controlGrant"`
	StopGrant          AuthorityGrant    `json:"stopGrant"`
	WorkflowGrant      AuthorityGrant    `json:"workflowGrant"`
	AssignGrant        AuthorityGrant    `json:"assignGrant"`
	MetadataGrant      AuthorityGrant    `json:"metadataGrant"`
	RegistryAssignment string            `json:"registryAssignment"`
}

func assignmentID(scope, id string) bool {
	prefix := scope + "/providers/Microsoft.Authorization/roleAssignments/"
	return strings.HasPrefix(id, prefix) && nonzeroGUID(strings.TrimPrefix(id, prefix))
}

func customRoleID(c Config, id string) bool {
	prefix := "/subscriptions/" + c.Subscription + "/providers/Microsoft.Authorization/roleDefinitions/"
	return strings.HasPrefix(id, prefix) && nonzeroGUID(strings.TrimPrefix(id, prefix))
}

func resourceGroupID(id string) string { return strings.Join(strings.Split(id, "/")[:5], "/") }

func (a AuthoritySettings) validate(s Settings) error {
	c := s.Worker
	resources := map[string]bool{}
	principals := map[string]bool{}
	clients := map[string]bool{}
	identities := []struct{ resource, client, principal string }{
		{a.Control.Resource, s.ControlClientID, a.Control.PrincipalID}, {a.Registry.Resource, s.RegistryClientID, a.Registry.PrincipalID},
		{a.Metadata.Resource, s.MetadataClientID, a.Metadata.PrincipalID}, {s.Timer.StopIdentity, a.StopClientID, a.StopPrincipalID}, {s.PullIdentity, s.Placement.PullClientID, s.Placement.PullPrincipalID},
	}
	for _, identity := range identities {
		key := strings.ToLower(identity.resource)
		if !c.validPullIdentity(identity.resource) || !nonzeroGUID(identity.client) || !nonzeroGUID(identity.principal) || identity.client == identity.principal || resources[key] || clients[identity.client] || principals[identity.principal] {
			return hostedexecution.ErrConfiguration
		}
		resources[key], clients[identity.client], principals[identity.principal] = true, true, true
	}
	roles, assignments := map[string]bool{}, map[string]bool{}
	for _, spec := range s.authorityRoles() {
		grant := spec.grant
		if !customRoleID(c, grant.Role) || !assignmentID(spec.scope, grant.Assignment) || roles[grant.Role] || assignments[grant.Assignment] {
			return hostedexecution.ErrConfiguration
		}
		roles[grant.Role], assignments[grant.Assignment] = true, true
	}
	if !assignmentID(s.Placement.Registry, a.RegistryAssignment) || a.RegistryAssignment == s.Placement.PullAssignment {
		return hostedexecution.ErrConfiguration
	}
	return nil
}
