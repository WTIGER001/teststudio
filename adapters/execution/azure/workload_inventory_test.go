package azureworker

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureWorkloadInventoryChecksFreshPlacementAndDropsConfiguration(t *testing.T) {
	c, s, f, g := workloadInventoryTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 2 {
		t.Fatal("valid observed placement rejected", err, f.reads)
	}
	s.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			if err := g.Check(t.Context(), c); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.reads != 8 {
		t.Fatal("workload inventory reused stale observations", f.reads)
	}
	reader := g.reader.(*SDKWorkloadInventoryReader)
	apps, err := reader.Applications(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := reader.Jobs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal([][]WorkloadPlacement{apps, jobs})
	if strings.Contains(string(data), "canary") || strings.Contains(string(data), "configuration") || len(apps) != 2 || len(jobs) != 1 {
		t.Fatal("inventory retained unrelated configuration")
	}
	f.apps[0].ID = to.Ptr(strings.ToUpper(value(f.apps[0].ID)))
	f.apps[1].ID = to.Ptr(strings.Replace(value(f.apps[1].ID), "/resourceGroups/studio/", "/resourceGroups/試験-資源/", 1))
	f.apps[1].Properties.ManagedEnvironmentID = nil
	f.apps[1].Properties.EnvironmentID = to.Ptr(c.resourceID("Microsoft.App", "connectedEnvironments", "unrelated-connected"))
	f.jobs = []*a.Job{}
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("case, Unicode resource group, legacy connected placement or explicit empty jobs changed observation", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("observed inventory became full network authority")
	}
	before := f.reads
	previous := g.exchange.reader.(*SDKExchangeReader).http.base.(*exchangeFixture)
	previous.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("unverified exchange permitted inventory reads")
	}
	previous.mode = ""
	c.NetworkProfile = "changed"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("changed profile reached subscription inventory")
	}
}

func TestAzureWorkloadInventoryRejectsOtherOccupantsAndConflictingIdentity(t *testing.T) {
	for name, mutate := range map[string]func(*workloadInventoryFixture){
		"app-in-other-resource-group": func(f *workloadInventoryFixture) {
			f.apps[1].ID = to.Ptr(strings.Replace(value(f.apps[1].ID), "/resourceGroups/studio/", "/resourceGroups/other-project/", 1))
			f.apps[1].Properties.ManagedEnvironmentID = to.Ptr(f.settings.environment())
		},
		"stopped-app": func(f *workloadInventoryFixture) {
			f.apps[1].Properties.ManagedEnvironmentID = to.Ptr(f.settings.environment())
			f.apps[1].Properties.RunningStatus = to.Ptr(a.ContainerAppRunningStatusStopped)
		},
		"job-in-other-resource-group": func(f *workloadInventoryFixture) {
			f.jobs[0].ID = to.Ptr(strings.Replace(value(f.jobs[0].ID), "/resourceGroups/studio/", "/resourceGroups/other-project/", 1))
			f.jobs[0].Properties.EnvironmentID = to.Ptr(f.settings.environment())
		},
		"duplicate-app-id-case": func(f *workloadInventoryFixture) {
			f.apps[1] = clone(f.apps[0])
			f.apps[1].ID = to.Ptr(strings.ToUpper(value(f.apps[0].ID)))
		},
		"duplicate-app-id-unicode-fold": func(f *workloadInventoryFixture) {
			f.apps[1] = clone(f.apps[0])
			f.apps[1].ID = to.Ptr(strings.Replace(value(f.apps[0].ID), "/resourceGroups/studio/", "/resourceGroups/ſtudio/", 1))
		},
		"duplicate-job": func(f *workloadInventoryFixture) { f.jobs = append(f.jobs, clone(f.jobs[0])) },
		"moved-selected-app": func(f *workloadInventoryFixture) {
			f.apps[0].Properties.ManagedEnvironmentID = to.Ptr(f.worker.ControllerEnvironment)
		},
		"conflicting-environment-alias": func(f *workloadInventoryFixture) {
			f.apps[1].Properties.EnvironmentID = to.Ptr(f.settings.environment())
		},
		"foreign-subscription-record": func(f *workloadInventoryFixture) {
			f.apps[1].ID = to.Ptr(strings.Replace(value(f.apps[1].ID), f.worker.Subscription, "99999999-2222-3333-4444-555555555555", 1))
		},
		"wrong-record-kind":   func(f *workloadInventoryFixture) { f.apps[1].Type = to.Ptr("Microsoft.App/jobs") },
		"wrong-resource-name": func(f *workloadInventoryFixture) { f.apps[1].Name = to.Ptr("different") },
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := workloadInventoryTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureWorkloadInventoryPreservesUnknownScopeAndBoundedPaging(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "forbidden", "redirect", "oversized", "missing-value", "null-jobs", "missing-selected-app", "missing-app-properties", "missing-job-properties", "missing-placement", "missing-type", "nil-app", "ambiguous-environment", "cancelled", "paged", "foreign-page", "other-collection", "foreign-subscription", "cycle", "endless"} {
		t.Run(mode, func(t *testing.T) {
			c, _, f, g := workloadInventoryTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "null-jobs":
				f.jobs = nil
			case "missing-selected-app":
				f.apps = f.apps[1:]
			case "missing-app-properties":
				f.apps[1].Properties = nil
			case "missing-job-properties":
				f.jobs[0].Properties = nil
			case "missing-placement":
				f.apps[1].Properties.ManagedEnvironmentID = nil
			case "missing-type":
				f.apps[1].Type = nil
			case "nil-app":
				f.apps[1] = nil
			case "ambiguous-environment":
				f.apps[1].Properties.ManagedEnvironmentID = to.Ptr("/subscriptions/unknown/environment")
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := g.Check(ctx, c)
			appPath := workloadInventoryPath(c, "containerApps")
			if mode == "paged" {
				if err != nil || f.pages[appPath] != 2 {
					t.Fatal("complete safe inventory paging rejected", err)
				}
				return
			}
			expected := hostedexecution.ErrUnavailable
			if mode == "foreign-page" || mode == "other-collection" || mode == "foreign-subscription" || mode == "cycle" {
				expected = hostedexecution.ErrOwnership
			}
			requireDeadlineError(t, err, expected)
			if f.reads > 17 || f.pages[appPath] > 16 {
				t.Fatal("inventory exceeded its page bound", f.reads)
			}
		})
	}
}

func TestAzureWorkloadInventoryBoundsItemsWithoutClaimingPartialAbsence(t *testing.T) {
	c, _, f, g := workloadInventoryTest(t)
	for len(f.apps) < 512 {
		name := "unrelated-" + strconv.Itoa(len(f.apps))
		f.apps = append(f.apps, &a.ContainerApp{ID: to.Ptr(c.resourceID("Microsoft.App", "containerApps", name)), Name: to.Ptr(name), Type: to.Ptr("Microsoft.App/containerApps"), Properties: &a.ContainerAppProperties{ManagedEnvironmentID: to.Ptr(c.ControllerEnvironment)}})
	}
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("bounded complete inventory rejected", err)
	}
	f.apps = append(f.apps, clone(f.apps[1]))
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 4 {
		t.Fatal("item bound retried or truncated inventory", f.reads)
	}
}
