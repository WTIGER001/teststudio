package azureworker

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureObserverPlacementsReuseCompleteSDKInventoryAndRejectSharing(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	inventory := f.parent.workloads
	inventory.mode = "paged"
	reader, err := NewSDKObserverWorkloadInventoryReader(c, s, fixtureCredential{}, inventory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	s.Identity.ClientID = "aaaaaaaa-1111-2222-3333-444444444444"
	apps, err := reader.Applications(t.Context())
	if err != nil || len(apps) != len(inventory.apps) {
		t.Fatal("complete paged app identities rejected", err)
	}
	jobs, err := reader.Jobs(t.Context())
	if err != nil || len(jobs) != 1 || !jobs[0].IdentityKnown || len(jobs[0].Identities) != 0 || inventory.reads != 3 {
		t.Fatal("explicit no-identity job or existing collection cost changed", err, inventory.reads)
	}
	data, _ := json.Marshal([][]WorkloadPlacement{apps, jobs})
	if strings.Contains(string(data), "canary") || strings.Contains(string(data), "configuration") {
		t.Fatal("identity projection retained unrelated configuration")
	}
	inventory.apps[1].Identity = observerAttachedIdentity(f.settings.Identity)
	inventory.apps[1].Properties.Configuration.IdentitySettings = []*a.IdentitySettings{{Identity: to.Ptr(f.settings.Identity.Identity), Lifecycle: to.Ptr(a.IdentitySettingsLifeCycleNone)}}
	_, err = reader.Applications(t.Context())
	requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	if inventory.reads != 5 {
		t.Fatal("sharing check repeated a subscription scan")
	}
	inventory.apps[1].Identity = &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeNone)}
	if _, err := reader.Applications(t.Context()); err != nil || inventory.reads != 7 {
		t.Fatal("fresh removal did not clear reported sharing", err, inventory.reads)
	}
	inventory.jobs[0].Identity = observerAttachedIdentity(f.settings.Identity)
	_, err = reader.Jobs(t.Context())
	requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	if inventory.reads != 8 {
		t.Fatal("job sharing check repeated a subscription scan")
	}
	_, err = NewSDKObserverWorkloadInventoryReader(c, f.settings, nil, nil)
	requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
}

func TestAzureObserverPlacementsPreserveUnknownMetadataAndReportedAliases(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	host, err := inventoryProjection(c, s.Application, value(f.application.Name), value(f.application.Type), "containerApps", c.ControllerEnvironment, "")
	if err != nil {
		t.Fatal(err)
	}
	host = inventoryWithIdentities(host, f.application.Identity)
	other := host
	other.ResourceID = c.resourceID("Microsoft.App", "containerApps", "other-app")
	otherIdentity := c.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "other-identity")
	for name, check := range map[string]struct {
		identity *a.ManagedServiceIdentity
		want     error
	}{
		"omitted":                  {want: hostedexecution.ErrUnavailable},
		"missing-type":             {identity: &a.ManagedServiceIdentity{}, want: hostedexecution.ErrUnavailable},
		"unknown-type":             {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityType("Unknown"))}, want: hostedexecution.ErrUnavailable},
		"none":                     {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeNone)}},
		"empty-user-map":           {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned)}, want: hostedexecution.ErrUnavailable},
		"null-user-entry":          {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{otherIdentity: nil}}, want: hostedexecution.ErrUnavailable},
		"empty-other-metadata":     {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{otherIdentity: {}}}},
		"case-duplicate":           {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{otherIdentity: {}, strings.ToUpper(otherIdentity): {}}}, want: hostedexecution.ErrUnavailable},
		"invalid-reference":        {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{"foreign-reference-canary": {}}}, want: hostedexecution.ErrUnavailable},
		"bad-client":               {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{otherIdentity: {ClientID: to.Ptr("private-client-canary")}}}, want: hostedexecution.ErrUnavailable},
		"client-alias":             {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{otherIdentity: {ClientID: to.Ptr(s.Identity.ClientID)}}}, want: hostedexecution.ErrOwnership},
		"principal-alias":          {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{otherIdentity: {PrincipalID: to.Ptr(s.Identity.PrincipalID)}}}, want: hostedexecution.ErrOwnership},
		"case-selected-reference":  {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{strings.ToUpper(s.Identity.Identity): {}}}, want: hostedexecution.ErrOwnership},
		"system-other":             {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeSystemAssigned), PrincipalID: to.Ptr("88888888-1111-2222-3333-444444444444")}},
		"system-alias":             {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeSystemAssigned), PrincipalID: to.Ptr(s.Identity.PrincipalID)}, want: hostedexecution.ErrOwnership},
		"system-missing-principal": {identity: &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeSystemAssigned)}, want: hostedexecution.ErrUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			placement := inventoryWithIdentities(other, check.identity)
			requireReadDenyError(t, verifyObserverPlacements(c, s, []WorkloadPlacement{host, placement}, "containerApps"), check.want)
		})
	}
	unknown := inventoryWithIdentities(host, nil)
	if unknown.IdentityKnown || unknown.Identities != nil || unknown.SystemPrincipalID != "" {
		t.Fatal("missing identity retained a previous complete observation")
	}
	requireReadDenyError(t, verifyObserverPlacements(c, s, []WorkloadPlacement{host, host}, "containerApps"), hostedexecution.ErrOwnership)
	requireReadDenyError(t, verifyObserverPlacements(c, s, []WorkloadPlacement{}, "containerApps"), hostedexecution.ErrUnavailable)
	requireReadDenyError(t, verifyObserverPlacements(c, s, nil, "jobs"), hostedexecution.ErrUnavailable)
	if err := verifyObserverPlacements(c, s, []WorkloadPlacement{}, "jobs"); err != nil {
		t.Fatal("explicitly empty job inventory rejected", err)
	}
	withoutObserver := inventoryWithIdentities(host, &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeNone)})
	requireReadDenyError(t, verifyObserverPlacements(c, s, []WorkloadPlacement{withoutObserver}, "containerApps"), hostedexecution.ErrUnavailable)
}

func TestAzureObserverPlacementsGateBoundHostWithoutAnotherSubscriptionScan(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	inventory := f.parent.workloads
	before := inventory.reads
	inventory.jobs[0].Identity = observerAttachedIdentity(s.Identity)
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if inventory.reads != before+2 || f.reads != 0 {
		t.Fatal("reported sharing repeated inventory or reached host reads", inventory.reads, f.reads)
	}
	inventory.jobs[0].Identity = &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeNone)}
	if err := observer.Check(t.Context(), c); err != nil || inventory.reads != before+4 || f.reads != 4 {
		t.Fatal("fresh sharing removal or single subscription scan failed", err, inventory.reads, f.reads)
	}
	inventory.apps = inventory.apps[:len(inventory.apps)-1]
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if inventory.reads != before+6 || f.reads != 4 {
		t.Fatal("missing reported host repeated inventory or reached host reads")
	}
}

func observerAttachedIdentity(s ReadIdentitySettings) *a.ManagedServiceIdentity {
	return &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeUserAssigned), UserAssignedIdentities: map[string]*a.UserAssignedIdentity{s.Identity: {ClientID: to.Ptr(s.ClientID), PrincipalID: to.Ptr(s.PrincipalID)}}}
}
