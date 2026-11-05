package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ServiceWorkloadBinding struct {
	Application, Environment string
	Identities               map[string]ServiceIdentityPin
}

func VerifyServiceWorkloadInventory(subscription string, bindings []ServiceWorkloadBinding, applications []a.ContainerApp, jobs []a.Job) error {
	if !observerGUID(strings.ToLower(subscription)) || len(bindings) == 0 {
		return hostedexecution.ErrConfiguration
	}
	if applications == nil || jobs == nil || len(applications) > workloadInventoryItems || len(jobs) > workloadInventoryItems {
		return hostedexecution.ErrUnavailable
	}
	c := Config{Subscription: subscription}
	apps := make([]WorkloadPlacement, 0, len(applications))
	tasks := make([]WorkloadPlacement, 0, len(jobs))
	for _, app := range applications {
		if app.Properties == nil {
			return hostedexecution.ErrUnavailable
		}
		placement, err := inventoryProjection(c, value(app.ID), value(app.Name), value(app.Type), "containerApps", value(app.Properties.ManagedEnvironmentID), value(app.Properties.EnvironmentID))
		if err != nil {
			return err
		}
		apps = append(apps, inventoryWithIdentities(placement, app.Identity))
	}
	for _, job := range jobs {
		if job.Properties == nil {
			return hostedexecution.ErrUnavailable
		}
		placement, err := inventoryProjection(c, value(job.ID), value(job.Name), value(job.Type), "jobs", value(job.Properties.EnvironmentID), "")
		if err != nil {
			return err
		}
		tasks = append(tasks, inventoryWithIdentities(placement, job.Identity))
	}
	for _, binding := range bindings {
		if err := verifyWorkloadPlacements(c, apps, "containerApps", binding.Environment, binding.Application); err != nil {
			return err
		}
		if err := verifyWorkloadPlacements(c, tasks, "jobs", binding.Environment, binding.Application); err != nil {
			return err
		}
		for _, app := range applications {
			if strings.EqualFold(value(app.ID), binding.Application) {
				if err := VerifyServiceIdentityPins(app, binding.Identities); err != nil {
					return err
				}
			}
		}
		for id, pin := range binding.Identities {
			host := identityHostBinding{binding.Application, binding.Environment, id, pin.Client, pin.Principal}
			if err := host.verify(c, apps, "containerApps"); err != nil {
				return err
			}
			if err := host.verify(c, tasks, "jobs"); err != nil {
				return err
			}
		}
	}
	return nil
}
