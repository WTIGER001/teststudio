package azureworker

import (
	"net/http"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureReadIdentityProtectionRequiresExactEnforcementWithoutWriteExceptions(t *testing.T) {
	_, s, f, _ := readIdentityTest(t)
	baseline := clone(f.identityProtection)
	slices.Reverse(baseline.Properties.Permissions[0].NotActions)
	if err := verifyReadIdentityProtection(s, baseline); err != nil {
		t.Fatal("read-preserving identity protection rejected", err)
	}
	for name, check := range map[string]struct {
		change func(*a.DenyAssignment)
		want   error
	}{
		"audit":          {change: func(d *a.DenyAssignment) { d.Properties.DenyAssignmentEffect = to.Ptr(a.DenyAssignmentEffectAudit) }, want: hostedexecution.ErrOwnership},
		"missing-effect": {change: func(d *a.DenyAssignment) { d.Properties.DenyAssignmentEffect = nil }, want: hostedexecution.ErrUnavailable},
		"child-bypass":   {change: func(d *a.DenyAssignment) { d.Properties.DoNotApplyToChildScopes = to.Ptr(true) }, want: hostedexecution.ErrOwnership},
		"unprotected":    {change: func(d *a.DenyAssignment) { d.Properties.IsSystemProtected = to.Ptr(false) }, want: hostedexecution.ErrOwnership},
		"foreign-scope": {change: func(d *a.DenyAssignment) {
			d.Properties.Scope = to.Ptr(s.Identity + "/federatedIdentityCredentials/other")
		}, want: hostedexecution.ErrOwnership},
		"excluded-principal": {change: func(d *a.DenyAssignment) {
			d.Properties.ExcludePrincipals = []*a.DenyAssignmentPrincipal{{ID: to.Ptr(s.PrincipalID), Type: to.Ptr("ServicePrincipal")}}
		}, want: hostedexecution.ErrOwnership},
		"conditional": {change: func(d *a.DenyAssignment) { d.Properties.Condition = to.Ptr("private-protection-condition") }, want: hostedexecution.ErrOwnership},
		"delete-only": {change: func(d *a.DenyAssignment) { d.Properties.Permissions[0].Actions = []*string{to.Ptr("*/delete")} }, want: hostedexecution.ErrOwnership},
		"assign-exception": {change: func(d *a.DenyAssignment) {
			d.Properties.Permissions[0].NotActions = append(d.Properties.Permissions[0].NotActions, to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities/assign/action"))
		}, want: hostedexecution.ErrOwnership},
		"federation-write-exception": {change: func(d *a.DenyAssignment) {
			d.Properties.Permissions[0].NotActions = append(d.Properties.Permissions[0].NotActions, to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials/write"))
		}, want: hostedexecution.ErrOwnership},
		"network-exception": {change: func(d *a.DenyAssignment) {
			d.Properties.Permissions[0].NotActions = append(d.Properties.Permissions[0].NotActions, to.Ptr(networkReadAssociationAction))
		}, want: hostedexecution.ErrOwnership},
		"missing-permissions": {change: func(d *a.DenyAssignment) { d.Properties.Permissions = nil }, want: hostedexecution.ErrUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			actual := clone(baseline)
			check.change(&actual)
			requireReadDenyError(t, verifyReadIdentityProtection(s, actual), check.want)
		})
	}
}

func TestAzureReadIdentityProtectionSDKAndSettingsRejectScopeSubstitution(t *testing.T) {
	c, s, f, g := readIdentityTest(t)
	reader := g.reader.(*SDKReadIdentityReader)
	_, err := reader.Protection(t.Context(), s.Protection.WorkerAssignment)
	requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.URL.Path = s.Protection.WorkerAssignment },
		func(r *http.Request) { r.URL.RawQuery = "api-version=2022-04-01" },
		func(r *http.Request) { r.URL.RawQuery += "&%24filter=atScope()" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.ProtectionAssignment+"?api-version="+networkDenyVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	for _, assignment := range []string{"", s.Assignment, s.Protection.WorkerAssignment, observerScope(c) + "/providers/Microsoft.Authorization/denyAssignments/66666666-ffff-6666-ffff-666666666666"} {
		changed := s
		changed.ProtectionAssignment = assignment
		_, err := NewSDKReadIdentityReader(c, changed, fixtureCredential{}, f)
		requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
	}
	if f.reads != 0 {
		t.Fatal("unselected identity protection reached transport")
	}
	actual, err := reader.Protection(t.Context(), s.ProtectionAssignment)
	if err != nil || verifyReadIdentityProtection(s, actual) != nil || f.reads != 1 {
		t.Fatal("exact identity protection read rejected", err, f.reads)
	}
	for _, mode := range []string{"unavailable", "redirect"} {
		f.mode = mode
		_, err := reader.Protection(t.Context(), s.ProtectionAssignment)
		requireReadDenyError(t, err, hostedexecution.ErrUnavailable)
	}
}

func TestAzureReadIdentityProtectionGatesFactoryAndBoundHostWithSixRequiredAssignments(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	identity := f.parent.identity
	baseline := clone(identity.identityProtection)
	identity.identityProtection = clone(baseline)
	identity.identityProtection.Properties.IsSystemProtected = to.Ptr(false)
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	if observer != nil || len(f.parent.factoryClients) != 0 || f.parent.credential.reads.Load() != 0 {
		t.Fatal("unprotected identity reached credential factory")
	}
	identity.identityProtection = clone(baseline)
	observer, err = g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrap := f.parent.bootstrapReads
	identity.identityProtection = clone(baseline)
	identity.identityProtection.Properties.Permissions[0].NotActions = append(identity.identityProtection.Properties.Permissions[0].NotActions, to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities/assign/action"))
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != 0 || f.parent.bootstrapReads != bootstrap {
		t.Fatal("changed identity protection reached host reads or repeated bootstrap")
	}
	identity.identityProtection = clone(baseline)
	if err := observer.Check(t.Context(), c); err != nil || f.reads != 4 {
		t.Fatal("fresh restored protection rejected", err, f.reads)
	}
	before := identity.reads
	f.parent.denies = f.parent.denies[:len(f.parent.denies)-1]
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if identity.reads != before || f.reads != 4 || len(f.parent.factoryClients) != 1 {
		t.Fatal("subscription scan without identity protection reached later reads")
	}
}
