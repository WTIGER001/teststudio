package azureworker

import (
	"context"
	"strings"
	"time"

	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeploymentstacks/v2"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const protectionStackVersion = "2025-07-01"

func (c *SDKReadIdentityReader) Stack(ctx context.Context, id string) (d.DeploymentStack, error) {
	if id != c.settings.NetworkProtectionStack && id != c.settings.IdentityProtectionStack {
		return d.DeploymentStack{}, hostedexecution.ErrOwnership
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	parts := strings.Split(id, "/")
	if len(parts) == 7 {
		response, err := c.stacks.GetAtSubscription(ctx, parts[6], nil)
		if err != nil {
			return d.DeploymentStack{}, sdkError(err, false)
		}
		return projectProtectionStack(&response.DeploymentStack)
	}
	response, err := c.stacks.GetAtResourceGroup(ctx, parts[4], parts[8], nil)
	if err != nil {
		return d.DeploymentStack{}, sdkError(err, false)
	}
	return projectProtectionStack(&response.DeploymentStack)
}

func projectProtectionStack(actual *d.DeploymentStack) (d.DeploymentStack, error) {
	if actual == nil {
		return d.DeploymentStack{}, hostedexecution.ErrUnavailable
	}
	result := d.DeploymentStack{ID: actual.ID, Name: actual.Name, Type: actual.Type, Location: actual.Location}
	p := actual.Properties
	if p == nil {
		return result, nil
	}
	if p.Error != nil || len(p.FailedResources) != 0 || value(p.BypassStackOutOfSyncError) {
		return d.DeploymentStack{}, hostedexecution.ErrUnavailable
	}
	properties := &d.DeploymentStackProperties{DenySettings: p.DenySettings, DeploymentScope: p.DeploymentScope, ProvisioningState: p.ProvisioningState}
	if p.Resources != nil {
		properties.Resources = make([]*d.ManagedResourceReference, 0, len(p.Resources))
		for _, resource := range p.Resources {
			if resource == nil {
				properties.Resources = append(properties.Resources, nil)
				continue
			}
			if resource.Extension != nil || resource.Identifiers != nil {
				return d.DeploymentStack{}, hostedexecution.ErrUnavailable
			}
			properties.Resources = append(properties.Resources, &d.ManagedResourceReference{ID: resource.ID, Type: resource.Type, Status: resource.Status, DenyStatus: resource.DenyStatus})
		}
	}
	result.Properties = properties
	return result, nil
}
