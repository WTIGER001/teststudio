package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyObserverHost(c Config, s ObserverHostSettings, app a.ContainerApp, revisions []a.Revision) error {
	if !perimeterIdentity(value(app.ID), value(app.Name), value(app.Type), s.Application, "Microsoft.App/containerApps", value(app.Location), c.Region) {
		return hostedexecution.ErrOwnership
	}
	p := app.Properties
	if p == nil || p.Configuration == nil || p.Template == nil || p.ProvisioningState == nil || p.RunningStatus == nil || app.Identity == nil || app.Identity.Type == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(p.ProvisioningState) != a.ContainerAppProvisioningStateSucceeded || value(p.RunningStatus) != a.ContainerAppRunningStatusRunning && value(p.RunningStatus) != a.ContainerAppRunningStatusReady {
		return hostedexecution.ErrUnavailable
	}
	placement, err := inventoryProjection(c, value(app.ID), value(app.Name), value(app.Type), "containerApps", value(p.ManagedEnvironmentID), value(p.EnvironmentID))
	if err != nil {
		return err
	}
	if !strings.EqualFold(placement.EnvironmentID, c.ControllerEnvironment) || value(app.Identity.Type) != a.ManagedServiceIdentityTypeUserAssigned || value(app.Identity.PrincipalID) != "" || value(app.Identity.TenantID) != "" || value(p.Configuration.ActiveRevisionsMode) != a.ActiveRevisionsModeSingle || len(p.Template.Containers) != 1 || p.Template.Containers[0] == nil || len(p.Template.InitContainers) != 0 || value(p.Template.Containers[0].Image) != s.Image {
		return hostedexecution.ErrOwnership
	}
	if err := verifyObserverHostIdentities(s, app); err != nil {
		return err
	}
	return verifyServiceRevisions(app, revisions, s.Application, p.Template, exchangeTemplateDocument)
}

func verifyObserverHostIdentities(s ObserverHostSettings, app a.ContainerApp) error {
	wanted := s.lifecycles()
	if len(app.Identity.UserAssignedIdentities) != len(wanted) || len(app.Properties.Configuration.IdentitySettings) != len(wanted) {
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
		if clients[client] || principals[principal] || client == principal || key == inventoryIdentityKey(s.Identity.Identity) && (client != s.Identity.ClientID || principal != s.Identity.PrincipalID) {
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
