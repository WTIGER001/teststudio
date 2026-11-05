package azureworker

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeploymentstacks/v2"
)

func protectionStackFixtures(c Config, s ReadIdentitySettings) map[string]d.DeploymentStack {
	stacks := map[string]d.DeploymentStack{}
	for i, binding := range s.stackBindings() {
		parts := strings.Split(binding.id, "/")
		stack := d.DeploymentStack{ID: to.Ptr(binding.id), Name: to.Ptr(parts[len(parts)-1]), Type: to.Ptr("Microsoft.Resources/deploymentStacks"), Location: to.Ptr(c.Region), Properties: &d.DeploymentStackProperties{ProvisioningState: to.Ptr(d.DeploymentStackProvisioningStateSucceeded), DenySettings: &d.DenySettings{Mode: to.Ptr(d.DenySettingsModeDenyWriteAndDelete), ApplyToChildScopes: to.Ptr(true)}, Template: map[string]any{"private-template-canary": "private-stack-template"}, Outputs: map[string]any{"private-output-canary": "private-stack-output"}}}
		if i == 0 {
			stack.Properties.DenySettings.ExcludedActions = []*string{to.Ptr(networkReadAssociationAction)}
		}
		for _, resource := range binding.resources {
			stack.Properties.Resources = append(stack.Properties.Resources, &d.ManagedResourceReference{ID: to.Ptr(resource.scope), Status: to.Ptr(d.ResourceStatusModeManaged), DenyStatus: to.Ptr(d.DenyStatusModeDenyWriteAndDelete)})
		}
		stacks[binding.id] = stack
	}
	return stacks
}
