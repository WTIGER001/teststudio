package azureworker

import "strings"

type protectionStackBinding struct {
	id         string
	resources  []networkDenyBinding
	exceptions []string
}

func (s ReadIdentitySettings) stackBindings() []protectionStackBinding {
	return []protectionStackBinding{
		{s.NetworkProtectionStack, s.Protection.bindings(), []string{"*/read", "Microsoft.Authorization/locks/delete", networkReadAssociationAction}},
		{s.IdentityProtectionStack, []networkDenyBinding{s.identityProtection()}, []string{"*/read", "Microsoft.Authorization/locks/delete"}},
	}
}

func observerStackID(c Config, id string) bool {
	parts := strings.Split(id, "/")
	if len(parts) == 9 {
		return c.ownsResource(id, "Microsoft.Resources", "deploymentStacks")
	}
	return len(parts) == 7 && parts[0] == "" && parts[1] == "subscriptions" && parts[2] == c.Subscription && parts[3] == "providers" && parts[4] == "Microsoft.Resources" && parts[5] == "deploymentStacks" && resourceName.MatchString(parts[6])
}

func observerStackScope(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) == 9 {
		return strings.Join(parts[:5], "/")
	}
	return strings.Join(parts[:3], "/")
}
