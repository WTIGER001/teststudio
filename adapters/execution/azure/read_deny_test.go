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

func TestAzureReadDenyChecksInheritedScopesPrincipalsAndReadExceptions(t *testing.T) {
	c, s, f, _ := readDenyTest(t)
	baseline := clone(f.assignments)
	if err := verifyReadDenies(c, s, baseline); err != nil {
		t.Fatal("selected read-preserving protection rejected", err)
	}
	for name, check := range map[string]struct {
		change func(*a.DenyAssignment)
		want   error
	}{
		"enforced-read": {want: hostedexecution.ErrOwnership},
		"write-only": {change: func(d *a.DenyAssignment) {
			d.Properties.Permissions[0].Actions = []*string{to.Ptr("Microsoft.Network/*/write")}
		}},
		"audit":          {change: func(d *a.DenyAssignment) { d.Properties.DenyAssignmentEffect = to.Ptr(a.DenyAssignmentEffectAudit) }},
		"missing-effect": {change: func(d *a.DenyAssignment) { d.Properties.DenyAssignmentEffect = nil }, want: hostedexecution.ErrUnavailable},
		"condition":      {change: func(d *a.DenyAssignment) { d.Properties.Condition = to.Ptr("private-condition-canary") }, want: hostedexecution.ErrUnavailable},
		"permission-condition": {change: func(d *a.DenyAssignment) {
			d.Properties.Permissions[0].Condition = to.Ptr("private-condition-canary")
		}, want: hostedexecution.ErrUnavailable},
		"another-principal": {change: func(d *a.DenyAssignment) {
			d.Properties.Principals = []*a.DenyAssignmentPrincipal{{ID: to.Ptr("88888888-aaaa-bbbb-cccc-111111111111"), Type: to.Ptr("ServicePrincipal")}}
		}},
		"direct-principal": {change: func(d *a.DenyAssignment) {
			d.Properties.Principals = []*a.DenyAssignmentPrincipal{{ID: to.Ptr(s.PrincipalID), Type: to.Ptr("ServicePrincipal")}}
		}, want: hostedexecution.ErrOwnership},
		"group-uncertain": {change: func(d *a.DenyAssignment) {
			d.Properties.Principals = []*a.DenyAssignmentPrincipal{{ID: to.Ptr("88888888-aaaa-bbbb-cccc-111111111111"), Type: to.Ptr("Group")}}
		}, want: hostedexecution.ErrOwnership},
		"direct-exclusion": {change: func(d *a.DenyAssignment) {
			d.Properties.ExcludePrincipals = []*a.DenyAssignmentPrincipal{{ID: to.Ptr(s.PrincipalID), Type: to.Ptr("ServicePrincipal")}}
		}},
		"group-exclusion": {change: func(d *a.DenyAssignment) {
			d.Properties.ExcludePrincipals = []*a.DenyAssignmentPrincipal{{ID: to.Ptr("88888888-aaaa-bbbb-cccc-111111111111"), Type: to.Ptr("Group")}}
		}, want: hostedexecution.ErrOwnership},
		"unknown-principal": {change: func(d *a.DenyAssignment) { d.Properties.Principals[0].Type = to.Ptr("Unknown") }, want: hostedexecution.ErrUnavailable},
		"ancestor": {change: func(d *a.DenyAssignment) {
			setReadDenyScope(d, "/providers/Microsoft.Management/managementGroups/studio-parent")
		}, want: hostedexecution.ErrOwnership},
		"ancestor-self-only": {change: func(d *a.DenyAssignment) {
			setReadDenyScope(d, "/providers/Microsoft.Management/managementGroups/studio-parent")
			d.Properties.DoNotApplyToChildScopes = to.Ptr(true)
		}},
		"root": {change: func(d *a.DenyAssignment) { setReadDenyScope(d, "/") }, want: hostedexecution.ErrOwnership},
		"foreign-subscription": {change: func(d *a.DenyAssignment) {
			setReadDenyScope(d, "/subscriptions/99999999-1111-2222-3333-444444444444")
		}, want: hostedexecution.ErrOwnership},
		"mismatched-id": {change: func(d *a.DenyAssignment) { d.ID = to.Ptr("foreign-resource-canary") }, want: hostedexecution.ErrOwnership},
	} {
		t.Run(name, func(t *testing.T) {
			extra := f.extra(observerScope(c)+"/resourceGroups/unrelated-inventory-group", "Microsoft.Network/*/read")
			if check.change != nil {
				check.change(&extra)
			}
			observed := append(clone(baseline), extra)
			requireReadDenyError(t, verifyReadDenies(c, s, observed), check.want)
		})
	}
	for _, observed := range [][]a.DenyAssignment{nil, baseline[:4], {}} {
		requireReadDenyError(t, verifyReadDenies(c, s, observed), hostedexecution.ErrUnavailable)
	}
	requireReadDenyError(t, verifyReadDenies(c, s, append(clone(baseline), baseline[0])), hostedexecution.ErrOwnership)
	for _, check := range []struct {
		action     string
		exceptions []string
		blocked    bool
	}{
		{"*", nil, true},
		{"*", []string{"*/read", networkReadAssociationAction}, false},
		{"*", []string{"*/read"}, true},
		{"Microsoft.Network/*/read", []string{"Microsoft.Network/*/read"}, false},
		{"Microsoft.Network/virtualNetworks/read", []string{"Microsoft.Network/*"}, false},
		{"Microsoft.Network/*/rea*", nil, true},
		{"Microsoft.Network/*/delete", nil, false},
		{"Microsoft.Network/*/write", nil, false},
		{"microsoft.network/virtualNetworks/*DnsResolverPolicies/action", nil, true},
		{"Microsoft.Network/*/read", []string{"Microsoft.Network/privateDnsZones/read"}, true},
	} {
		permission := &a.DenyAssignmentPermission{Actions: []*string{to.Ptr(check.action)}}
		for _, action := range check.exceptions {
			permission.NotActions = append(permission.NotActions, to.Ptr(action))
		}
		blocked, err := readDenyPermission(permission)
		if err != nil || blocked != check.blocked {
			t.Fatal("deny action or exception semantics changed", check.action, err, blocked)
		}
	}
}

func TestAzureReadDenySDKRejectsIncompleteInventoryAndUnsafePaging(t *testing.T) {
	c, s, f, g := readDenyTest(t)
	reader := g.reader.(*SDKReadDenyReader)
	f.mode = "paged"
	assignments, err := reader.Assignments(t.Context())
	if err != nil || len(assignments) != 5 || f.reads != 2 || verifyReadDenies(c, s, assignments) != nil {
		t.Fatal("bounded same-scope paging rejected", err, f.reads)
	}
	before := f.reads
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.URL.Path += "/foreign" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&%24filter=atScope()" },
		func(r *http.Request) { r.URL.RawQuery = "api-version=2022-04-01" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+f.path+"?api-version="+networkDenyVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != before {
		t.Fatal("unselected deny query reached transport")
	}
	for mode, want := range map[string]error{
		"missing-value": hostedexecution.ErrUnavailable, "null-value": hostedexecution.ErrUnavailable,
		"null-item": hostedexecution.ErrOwnership, "foreign-page": hostedexecution.ErrOwnership,
		"wrong-api-page": hostedexecution.ErrOwnership, "endless": hostedexecution.ErrUnavailable,
		"redirect": hostedexecution.ErrUnavailable, "unavailable": hostedexecution.ErrUnavailable,
	} {
		f.mode, f.reads = mode, 0
		_, err := reader.Assignments(t.Context())
		requireReadDenyError(t, err, want)
		if mode == "endless" && f.reads != readDenyPages || mode != "endless" && f.reads != 1 {
			t.Fatal("deny inventory exceeded its page boundary", mode, f.reads)
		}
	}
	_, err = NewSDKReadDenyReader(c, nil, nil)
	requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
}

func TestAzureReadDenyGatesFreshParentAndRejectsMissingSelectedProtection(t *testing.T) {
	c, _, f, g := readDenyTest(t)
	for i := 1; i <= 2; i++ {
		slices.Reverse(f.assignments)
		if err := g.Check(t.Context(), c); err != nil || f.reads != i || g.identity.reader.(*SDKReadIdentityReader).http.base.(*readIdentityFixture).reads != i*3 {
			t.Fatal("fresh deny inspection repeated or skipped parent", err, f.reads)
		}
	}
	previous := g.identity.reader.(*SDKReadIdentityReader).http.base.(*readIdentityFixture)
	f.assignments = append(f.assignments, f.extra(observerScope(c), "*/read"))
	requireReadDenyError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if previous.reads != 6 {
		t.Fatal("blocked read coverage reached parent inspection")
	}
	f.assignments = f.assignments[:3]
	requireReadDenyError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if previous.reads != 6 {
		t.Fatal("incomplete deny inventory reached parent inspection")
	}
	before := f.reads
	c.NetworkProfile = "other-profile"
	requireReadDenyError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("runtime mismatch reached deny collection")
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("reported read deny metadata enabled native network admission")
	}
}

func setReadDenyScope(d *a.DenyAssignment, scope string) {
	d.Properties.Scope = to.Ptr(scope)
	d.ID = to.Ptr(strings.TrimSuffix(scope, "/") + "/providers/Microsoft.Authorization/denyAssignments/" + value(d.Name))
}

func requireReadDenyError(t *testing.T, actual, expected error) {
	t.Helper()
	if expected == nil {
		if actual != nil {
			t.Fatal("read-preserving deny state rejected", actual)
		}
		return
	}
	requireDeadlineError(t, actual, expected)
}
