package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type serviceIdentityPin struct{ client, principal string }

func verifyServiceIdentities(app a.ContainerApp, wanted map[string]string, pins map[string]serviceIdentityPin) error {
	if app.Identity == nil || app.Properties == nil || app.Properties.Configuration == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(app.Identity.Type) != a.ManagedServiceIdentityTypeUserAssigned || value(app.Identity.PrincipalID) != "" || value(app.Identity.TenantID) != "" || len(app.Identity.UserAssignedIdentities) != len(wanted) || len(app.Properties.Configuration.IdentitySettings) != len(wanted) {
		return hostedexecution.ErrOwnership
	}
	clients, principals, identities := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for id, identity := range app.Identity.UserAssignedIdentities {
		key := inventoryIdentityKey(id)
		if wanted[key] == "" || identities[key] {
			return hostedexecution.ErrOwnership
		}
		if identity == nil || !observerGUID(strings.ToLower(value(identity.ClientID))) || !observerGUID(strings.ToLower(value(identity.PrincipalID))) {
			return hostedexecution.ErrUnavailable
		}
		client, principal := strings.ToLower(value(identity.ClientID)), strings.ToLower(value(identity.PrincipalID))
		pin := pins[key]
		if clients[client] || principals[principal] || client == principal || pin.client != "" && client != pin.client || pin.principal != "" && principal != pin.principal {
			return hostedexecution.ErrOwnership
		}
		clients[client], principals[principal], identities[key] = true, true, true
	}
	seen := map[string]bool{}
	for _, setting := range app.Properties.Configuration.IdentitySettings {
		if setting == nil || setting.Identity == nil || setting.Lifecycle == nil {
			return hostedexecution.ErrUnavailable
		}
		key := inventoryIdentityKey(value(setting.Identity))
		if wanted[key] == "" || seen[key] || string(value(setting.Lifecycle)) != wanted[key] {
			return hostedexecution.ErrOwnership
		}
		seen[key] = true
	}
	return nil
}
