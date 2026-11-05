package azureworker

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeploymentstacks/v2"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyProtectionStack(c Config, binding protectionStackBinding, actual d.DeploymentStack) error {
	if !observerStackID(c, binding.id) {
		return hostedexecution.ErrConfiguration
	}
	if value(actual.ID) == "" || value(actual.Name) == "" || value(actual.Type) == "" || value(actual.Location) == "" {
		return hostedexecution.ErrUnavailable
	}
	parts := strings.Split(binding.id, "/")
	if !strings.EqualFold(value(actual.ID), binding.id) || !strings.EqualFold(value(actual.Name), parts[len(parts)-1]) || !strings.EqualFold(value(actual.Type), "Microsoft.Resources/deploymentStacks") || value(actual.Location) != c.Region {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if p == nil || p.ProvisioningState == nil || p.DenySettings == nil || p.DenySettings.Mode == nil || p.DenySettings.ApplyToChildScopes == nil || p.Resources == nil || len(p.Resources) > 256 {
		return hostedexecution.ErrUnavailable
	}
	if value(p.ProvisioningState) != d.DeploymentStackProvisioningStateSucceeded || value(p.BypassStackOutOfSyncError) || p.Error != nil || len(p.FailedResources) != 0 {
		return hostedexecution.ErrUnavailable
	}
	if value(p.DenySettings.Mode) != d.DenySettingsModeDenyWriteAndDelete || !value(p.DenySettings.ApplyToChildScopes) || len(p.DenySettings.ExcludedPrincipals) != 0 || value(p.DeploymentScope) != "" && !strings.EqualFold(value(p.DeploymentScope), observerStackScope(binding.id)) {
		return hostedexecution.ErrOwnership
	}
	if err := verifyStackExceptions(p.DenySettings.ExcludedActions, binding.exceptions); err != nil {
		return err
	}
	wanted, seen := map[string]string{}, map[string]bool{}
	for _, resource := range binding.resources {
		id, err := arm.ParseResourceID(resource.scope)
		if err != nil {
			return hostedexecution.ErrConfiguration
		}
		wanted[inventoryIdentityKey(resource.scope)] = id.ResourceType.String()
	}
	for _, resource := range p.Resources {
		if resource == nil || resource.ID == nil || resource.Status == nil || resource.DenyStatus == nil {
			return hostedexecution.ErrUnavailable
		}
		key := inventoryIdentityKey(value(resource.ID))
		kind, expected := wanted[key]
		if !expected || seen[key] || resource.Type != nil && value(resource.Type) != "" && !strings.EqualFold(value(resource.Type), kind) || resource.Extension != nil || resource.Identifiers != nil {
			return hostedexecution.ErrOwnership
		}
		if value(resource.Status) != d.ResourceStatusModeManaged || value(resource.DenyStatus) != d.DenyStatusModeDenyWriteAndDelete {
			return hostedexecution.ErrUnavailable
		}
		seen[key] = true
	}
	if len(seen) != len(wanted) {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func verifyStackExceptions(actions []*string, expected []string) error {
	if len(actions) > 200 {
		return hostedexecution.ErrUnavailable
	}
	effective := map[string]string{}
	for _, action := range append([]*string{to.Ptr("*/read"), to.Ptr("Microsoft.Authorization/locks/delete")}, actions...) {
		if action == nil || !readDenyPattern(value(action)) {
			return hostedexecution.ErrUnavailable
		}
		effective[inventoryIdentityKey(value(action))] = value(action)
	}
	normalized := make([]*string, 0, len(effective))
	for _, action := range effective {
		normalized = append(normalized, &action)
	}
	if !networkDenyActions(normalized, expected...) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
