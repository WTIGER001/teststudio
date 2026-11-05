package azureworker

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureNetworkDenyRequiresFreshEnforcedProtectionAfterInventory(t *testing.T) {
	c, s, f, g := networkDenyTest(t)
	s.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.RegistryEndpointIPs[0] = "8.8.8.8"
	for i := 1; i <= 2; i++ {
		if i == 2 {
			permission := f.assignments[s.WorkerAssignment].Properties.Permissions[0]
			slices.Reverse(permission.NotActions)
			permission.NotActions[0] = to.Ptr(strings.ToLower(value(permission.NotActions[0])))
		}
		if err := g.Check(t.Context(), c); err != nil || f.reads != i*5 || f.inventory.reads != i*4 || f.inventory.selected.reads != i*3 {
			t.Fatal("network protection rejected or parent repeated", err, f.reads, f.inventory.reads)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("selected network protection became complete network admission")
	}
	f.assignments[s.WorkerAssignment].Properties.DenyAssignmentEffect = to.Ptr(a.DenyAssignmentEffectAudit)
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 15 {
		t.Fatal("audit-only update reused cached enforcement")
	}
	f.inventory.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 15 {
		t.Fatal("unverified reported inventory reached deny reads")
	}
	c.NetworkProfile = "other-profile"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 15 {
		t.Fatal("foreign profile reached deny reads")
	}
}

func TestAzureNetworkDenyRejectsJoinExceptionsPrincipalBypassesAndConditions(t *testing.T) {
	for name, mutate := range map[string]func(*a.DenyAssignmentProperties){
		"audit":                func(p *a.DenyAssignmentProperties) { p.DenyAssignmentEffect = to.Ptr(a.DenyAssignmentEffectAudit) },
		"unprotected":          func(p *a.DenyAssignmentProperties) { p.IsSystemProtected = to.Ptr(false) },
		"children-unprotected": func(p *a.DenyAssignmentProperties) { p.DoNotApplyToChildScopes = to.Ptr(true) },
		"foreign-scope": func(p *a.DenyAssignmentProperties) {
			p.Scope = to.Ptr("/subscriptions/foreign/resourceGroups/private-canary")
		},
		"specific-principal": func(p *a.DenyAssignmentProperties) {
			p.Principals[0].ID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
		},
		"excluded-group": func(p *a.DenyAssignmentProperties) {
			p.ExcludePrincipals = []*a.DenyAssignmentPrincipal{{ID: to.Ptr("aaaaaaaa-1111-2222-3333-444444444444"), Type: to.Ptr("Group")}}
		},
		"assignment-condition": func(p *a.DenyAssignmentProperties) {
			p.Condition = to.Ptr("private-condition-canary")
			p.ConditionVersion = to.Ptr("2.0")
		},
		"permission-condition": func(p *a.DenyAssignmentProperties) { p.Permissions[0].Condition = to.Ptr("private-condition-canary") },
		"delete-only":          func(p *a.DenyAssignmentProperties) { p.Permissions[0].Actions = []*string{to.Ptr("*/delete")} },
		"join-exception": func(p *a.DenyAssignmentProperties) {
			p.Permissions[0].NotActions = append(p.Permissions[0].NotActions, to.Ptr("Microsoft.Network/virtualNetworks/join/action"))
		},
		"network-wildcard":    func(p *a.DenyAssignmentProperties) { p.Permissions[0].NotActions[0] = to.Ptr("Microsoft.Network/*") },
		"duplicate-exception": func(p *a.DenyAssignmentProperties) { p.Permissions[0].NotActions[2] = p.Permissions[0].NotActions[0] },
		"extra-permission":    func(p *a.DenyAssignmentProperties) { p.Permissions = append(p.Permissions, clone(p.Permissions[0])) },
		"data-permission":     func(p *a.DenyAssignmentProperties) { p.Permissions[0].DataActions = []*string{to.Ptr("*")} },
	} {
		t.Run(name, func(t *testing.T) {
			c, s, f, g := networkDenyTest(t)
			mutate(f.assignments[s.RelayAssignment].Properties)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
	c, s, f, g := networkDenyTest(t)
	f.assignments[s.WorkerRegistryZoneAssignment].Properties.Permissions[0].NotActions = append(f.assignments[s.WorkerRegistryZoneAssignment].Properties.Permissions[0].NotActions, to.Ptr("Microsoft.Network/privateDnsZones/A/write"))
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
}

func TestAzureNetworkDenyPreservesUnknownStateAndExactSDKScope(t *testing.T) {
	c, s, f, g := networkDenyTest(t)
	reader := g.reader.(*SDKNetworkDenyReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) {
			r.URL.Path = strings.TrimSuffix(s.WorkerAssignment, "/aaaaaaaa-1111-2222-3333-444444444444")
		},
		func(r *http.Request) { r.URL.Path = s.bindings()[0].scope },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery = "api-version=2022-04-01" },
		func(r *http.Request) { r.URL.RawQuery += "&$filter=approved" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.WorkerAssignment+"?api-version="+networkDenyVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.Assignment(t.Context(), s.bindings()[1].scope)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected deny request reached transport")
	}
	for _, mode := range []string{"unavailable", "missing", "redirect"} {
		f.mode = mode
		requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	}
	f.mode = ""
	for _, mutate := range []func(*a.DenyAssignmentProperties){
		func(p *a.DenyAssignmentProperties) { p.DenyAssignmentEffect = nil },
		func(p *a.DenyAssignmentProperties) { p.IsSystemProtected = nil },
		func(p *a.DenyAssignmentProperties) { p.DoNotApplyToChildScopes = nil },
		func(p *a.DenyAssignmentProperties) { p.Permissions = nil },
		func(p *a.DenyAssignmentProperties) { p.Principals = nil },
	} {
		original := clone(f.assignments[s.WorkerAssignment].Properties)
		mutate(f.assignments[s.WorkerAssignment].Properties)
		requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
		assignment := f.assignments[s.WorkerAssignment]
		assignment.Properties = original
		f.assignments[s.WorkerAssignment] = assignment
	}
}

func TestAzureNetworkDenySettingsBindSelectedResourcesAndParent(t *testing.T) {
	c, s, _, g := networkDenyTest(t)
	for _, id := range []string{s.RelayAssignment, strings.Replace(s.WorkerAssignment, c.Subscription, "99999999-2222-3333-4444-555555555555", 1), s.WorkerAssignment + "?secret=private", s.bindings()[0].scope, ""} {
		changed := s
		changed.WorkerAssignment = id
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	changed := s
	changed.PrivateDNS.RegistryZone = strings.Replace(s.PrivateDNS.RegistryZone, "/resourceGroups/relay-private-dns/", "/resourceGroups/other-private-dns/", 1)
	_, err := NewNetworkDenyGuard(c, changed, g.inventory, g.reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	_, err = NewSDKNetworkDenyReader(c, s, nil, nil)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	root := "/subscriptions/" + c.Subscription + "/providers/Microsoft.Authorization/denyAssignments/cccccccc-1111-2222-3333-444444444444"
	if networkDenyAssignmentID(root, s.bindings()[0].scope) {
		t.Fatal("broader scope substituted for selected resource protection")
	}
	missing := s
	missing.RelayRegistryZoneAssignment = ""
	_, err = missing.canonical(c)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
}
