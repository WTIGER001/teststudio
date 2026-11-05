package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ReadIdentitySettings struct {
	Protection              NetworkDenySettings `json:"protection"`
	Identity                string              `json:"identity"`
	ClientID                string              `json:"clientId"`
	PrincipalID             string              `json:"principalId"`
	TenantID                string              `json:"tenantId"`
	Assignment              string              `json:"assignment"`
	Role                    string              `json:"role"`
	ProtectionAssignment    string              `json:"protectionAssignment"`
	NetworkProtectionStack  string              `json:"networkProtectionStack"`
	IdentityProtectionStack string              `json:"identityProtectionStack"`
}

func (s ReadIdentitySettings) canonical(c Config) (ReadIdentitySettings, error) {
	var err error
	s.Protection, err = s.Protection.canonical(c)
	if err != nil || !c.ownsResource(s.Identity, "Microsoft.ManagedIdentity", "userAssignedIdentities") || !observerGUID(s.ClientID) || !observerGUID(s.PrincipalID) || !observerGUID(s.TenantID) || s.ClientID == s.PrincipalID || !observerAuthorizationID(c, s.Assignment, "roleAssignments") || !observerAuthorizationID(c, s.Role, "roleDefinitions") || !networkDenyAssignmentID(s.ProtectionAssignment, s.Identity) {
		return s, hostedexecution.ErrConfiguration
	}
	backend := s.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend
	for _, id := range []string{c.PullIdentity, backend.PullIdentity, backend.StartupIdentity, backend.RuntimeIdentity} {
		if strings.EqualFold(s.Identity, id) {
			return s, hostedexecution.ErrConfiguration
		}
	}
	if strings.EqualFold(s.NetworkProtectionStack, s.IdentityProtectionStack) {
		return s, hostedexecution.ErrConfiguration
	}
	for _, binding := range s.stackBindings() {
		if !observerStackID(c, binding.id) {
			return s, hostedexecution.ErrConfiguration
		}
		for _, resource := range binding.resources {
			if !strings.HasPrefix(strings.ToLower(resource.scope), strings.ToLower(observerStackScope(binding.id))+"/") {
				return s, hostedexecution.ErrConfiguration
			}
		}
	}
	return s, nil
}

func (s ReadIdentitySettings) identityProtection() networkDenyBinding {
	return networkDenyBinding{assignment: s.ProtectionAssignment, scope: s.Identity}
}

func observerGUID(id string) bool { return guid.MatchString(id) && id != allDenyPrincipals }

func observerScope(c Config) string { return "/subscriptions/" + c.Subscription }

func observerAuthorizationID(c Config, id, kind string) bool {
	prefix := observerScope(c) + "/providers/Microsoft.Authorization/" + kind + "/"
	return strings.HasPrefix(id, prefix) && observerGUID(strings.TrimPrefix(id, prefix))
}
