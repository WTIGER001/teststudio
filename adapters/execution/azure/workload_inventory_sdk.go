package azureworker

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const workloadInventoryItems = 512
const workloadInventoryPages = 16

type SDKWorkloadInventoryReader struct {
	worker       Config
	applications *a.ContainerAppsClient
	jobs         *a.JobsClient
	http         *armReadHTTP
	observer     *ObserverHostSettings
}

func workloadInventoryPath(c Config, kind string) string {
	return "/subscriptions/" + c.Subscription + "/providers/Microsoft.App/" + kind
}

func NewSDKWorkloadInventoryReader(c Config, credential azcore.TokenCredential, transport policy.Transporter) (*SDKWorkloadInventoryReader, error) {
	if Validate(c) != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	applications, jobs := workloadInventoryPath(c, "containerApps"), workloadInventoryPath(c, "jobs")
	h := newARMReadHTTP(c, map[string]string{applications: exchangeVersion, jobs: exchangeVersion}, transport)
	h.pages = map[string]bool{applications: true, jobs: true}
	factory, err := a.NewClientFactory(c.Subscription, credential, publicARMOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKWorkloadInventoryReader{worker: c, applications: factory.NewContainerAppsClient(), jobs: factory.NewJobsClient(), http: h}, nil
}

func (c *SDKWorkloadInventoryReader) Close() { c.http.CloseIdleConnections() }

func NewSDKObserverWorkloadInventoryReader(c Config, s ObserverHostSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKWorkloadInventoryReader, error) {
	s, err := s.canonical(c)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	reader, err := NewSDKWorkloadInventoryReader(c, credential, transport)
	if err != nil {
		return nil, err
	}
	reader.observer = &s
	return reader, nil
}

func (c *SDKWorkloadInventoryReader) Applications(ctx context.Context) ([]WorkloadPlacement, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pager := c.applications.NewListBySubscriptionPager(nil)
	placements, err := placementInventory(ctx, c.http, workloadInventoryPath(c.worker, "containerApps"), pager.More, func(ctx context.Context) ([]*a.ContainerApp, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	}, func(app *a.ContainerApp) (WorkloadPlacement, error) {
		if app.Properties == nil {
			return WorkloadPlacement{}, hostedexecution.ErrUnavailable
		}
		placement, err := inventoryProjection(c.worker, value(app.ID), value(app.Name), value(app.Type), "containerApps", value(app.Properties.ManagedEnvironmentID), value(app.Properties.EnvironmentID))
		return inventoryWithIdentities(placement, app.Identity), err
	})
	return c.observedPlacements(placements, "containerApps", err)
}

func (c *SDKWorkloadInventoryReader) Jobs(ctx context.Context) ([]WorkloadPlacement, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pager := c.jobs.NewListBySubscriptionPager(nil)
	placements, err := placementInventory(ctx, c.http, workloadInventoryPath(c.worker, "jobs"), pager.More, func(ctx context.Context) ([]*a.Job, *string, error) {
		page, err := pager.NextPage(ctx)
		return page.Value, page.NextLink, err
	}, func(job *a.Job) (WorkloadPlacement, error) {
		if job.Properties == nil {
			return WorkloadPlacement{}, hostedexecution.ErrUnavailable
		}
		placement, err := inventoryProjection(c.worker, value(job.ID), value(job.Name), value(job.Type), "jobs", value(job.Properties.EnvironmentID), "")
		return inventoryWithIdentities(placement, job.Identity), err
	})
	return c.observedPlacements(placements, "jobs", err)
}

func (c *SDKWorkloadInventoryReader) observedPlacements(placements []WorkloadPlacement, kind string, err error) ([]WorkloadPlacement, error) {
	if err != nil {
		return nil, err
	}
	if c.observer != nil {
		if err := verifyObserverPlacements(c.worker, *c.observer, placements, kind); err != nil {
			return nil, err
		}
	}
	return placements, nil
}

func placementInventory[T any](ctx context.Context, h *armReadHTTP, path string, more func() bool, fetch func(context.Context) ([]*T, *string, error), project func(*T) (WorkloadPlacement, error)) ([]WorkloadPlacement, error) {
	count := 0
	return armInventory(ctx, h, path, exchangeVersion, workloadInventoryItems, workloadInventoryPages, more, func(ctx context.Context) ([]*WorkloadPlacement, *string, error) {
		items, next, err := fetch(ctx)
		if err != nil {
			return nil, nil, err
		}
		if items == nil {
			return nil, nil, hostedexecution.ErrUnavailable
		}
		if len(items) > workloadInventoryItems-count {
			return nil, nil, hostedexecution.ErrUnavailable
		}
		count += len(items)
		placements := make([]*WorkloadPlacement, 0, len(items))
		for _, item := range items {
			if item == nil {
				return nil, nil, hostedexecution.ErrUnavailable
			}
			placement, err := project(item)
			if err != nil {
				return nil, nil, err
			}
			placements = append(placements, &placement)
		}
		return placements, next, nil
	})
}

var _ WorkloadInventoryReader = (*SDKWorkloadInventoryReader)(nil)
