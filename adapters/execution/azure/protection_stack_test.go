package azureworker

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	d "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeploymentstacks/v2"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureProtectionStacksRequireManagedProtectedResourcesAndFixedPolicy(t *testing.T) {
	c, s, f, _ := readIdentityTest(t)
	bindings := s.stackBindings()
	for _, binding := range bindings {
		actual := clone(f.stacks[binding.id])
		slices.Reverse(actual.Properties.Resources)
		actual.Properties.DenySettings.ExcludedActions = append(actual.Properties.DenySettings.ExcludedActions, to.Ptr("*/READ"), to.Ptr("Microsoft.Authorization/locks/delete"))
		if err := verifyProtectionStack(c, binding, actual); err != nil {
			t.Fatal("managed resource set or automatic default exceptions rejected", err)
		}
	}
	for name, check := range map[string]struct {
		change func(*d.DeploymentStack)
		want   error
	}{
		"wrong-id":    {change: func(s *d.DeploymentStack) { s.ID = to.Ptr("foreign-stack-canary") }, want: hostedexecution.ErrOwnership},
		"missing-id":  {change: func(s *d.DeploymentStack) { s.ID = nil }, want: hostedexecution.ErrUnavailable},
		"incomplete":  {change: func(s *d.DeploymentStack) { s.Properties.ProvisioningState = nil }, want: hostedexecution.ErrUnavailable},
		"bypassed":    {change: func(s *d.DeploymentStack) { s.Properties.BypassStackOutOfSyncError = to.Ptr(true) }, want: hostedexecution.ErrUnavailable},
		"delete-only": {change: func(s *d.DeploymentStack) { s.Properties.DenySettings.Mode = to.Ptr(d.DenySettingsModeDenyDelete) }, want: hostedexecution.ErrOwnership},
		"no-children": {change: func(s *d.DeploymentStack) { s.Properties.DenySettings.ApplyToChildScopes = to.Ptr(false) }, want: hostedexecution.ErrOwnership},
		"excluded-principal": {change: func(s *d.DeploymentStack) {
			s.Properties.DenySettings.ExcludedPrincipals = []*string{to.Ptr("11111111-aaaa-bbbb-cccc-111111111111")}
		}, want: hostedexecution.ErrOwnership},
		"write-exception": {change: func(s *d.DeploymentStack) {
			s.Properties.DenySettings.ExcludedActions = append(s.Properties.DenySettings.ExcludedActions, to.Ptr("Microsoft.Network/*/write"))
		}, want: hostedexecution.ErrOwnership},
		"scope-override": {change: func(s *d.DeploymentStack) {
			s.Properties.DeploymentScope = to.Ptr("/providers/Microsoft.Management/managementGroups/foreign")
		}, want: hostedexecution.ErrOwnership},
		"missing-resource": {change: func(s *d.DeploymentStack) { s.Properties.Resources = s.Properties.Resources[1:] }, want: hostedexecution.ErrUnavailable},
		"duplicate-resource": {change: func(s *d.DeploymentStack) {
			s.Properties.Resources = append(s.Properties.Resources, clone(s.Properties.Resources[0]))
		}, want: hostedexecution.ErrOwnership},
		"unknown-deny": {change: func(s *d.DeploymentStack) { s.Properties.Resources[0].DenyStatus = to.Ptr(d.DenyStatusModeUnknown) }, want: hostedexecution.ErrUnavailable},
		"removed-deny": {change: func(s *d.DeploymentStack) {
			s.Properties.Resources[0].DenyStatus = to.Ptr(d.DenyStatusModeRemovedBySystem)
		}, want: hostedexecution.ErrUnavailable},
		"failed-management": {change: func(s *d.DeploymentStack) {
			s.Properties.Resources[0].Status = to.Ptr(d.ResourceStatusModeRemoveDenyFailed)
		}, want: hostedexecution.ErrUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			actual := clone(f.stacks[bindings[0].id])
			check.change(&actual)
			requireReadDenyError(t, verifyProtectionStack(c, bindings[0], actual), check.want)
		})
	}
	changed := s
	changed.NetworkProtectionStack = c.resourceID("Microsoft.Resources", "deploymentStacks", "local-network-protection")
	_, err := changed.canonical(c)
	requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
	changed = s
	changed.IdentityProtectionStack = s.NetworkProtectionStack
	_, err = changed.canonical(c)
	requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
}

func TestAzureProtectionStacksSDKLimitsSelectedScopesAndDropsTemplateData(t *testing.T) {
	c, s, f, g := readIdentityTest(t)
	reader := g.reader.(*SDKReadIdentityReader)
	_, err := reader.Stack(t.Context(), c.resourceID("Microsoft.Resources", "deploymentStacks", "other-stack"))
	requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.URL.Path += "/exportTemplate" },
		func(r *http.Request) {
			r.URL.Path = observerStackScope(s.NetworkProtectionStack) + "/providers/Microsoft.Resources/deploymentStacks"
		},
		func(r *http.Request) { r.URL.RawQuery += "&%24filter=atScope()" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.NetworkProtectionStack+"?api-version="+protectionStackVersion, nil)
		mutate(request)
		_, err := reader.http.Do(request)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 {
		t.Fatal("unselected stack operation reached transport")
	}
	for _, binding := range s.stackBindings() {
		actual, err := reader.Stack(t.Context(), binding.id)
		if err != nil || verifyProtectionStack(c, binding, actual) != nil {
			t.Fatal("selected stack GET rejected", err)
		}
		data, _ := json.Marshal(actual)
		if strings.Contains(string(data), "canary") || strings.Contains(string(data), "template") || strings.Contains(string(data), "outputs") || strings.Contains(string(data), "parameters") {
			t.Fatal("stack observation retained deployment configuration")
		}
	}
	if f.reads != 2 {
		t.Fatal("stack observations repeated selected GETs")
	}
	for _, mode := range []string{"unavailable", "redirect"} {
		f.mode = mode
		_, err := reader.Stack(t.Context(), s.NetworkProtectionStack)
		requireReadDenyError(t, err, hostedexecution.ErrUnavailable)
	}
}

func TestAzureProtectionStacksRefreshPolicyAndGateCredentialsAndBoundHost(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	identity := f.parent.identity
	id := s.Identity.IdentityProtectionStack
	baseline := clone(identity.stacks[id])
	invalid := clone(baseline)
	invalid.Properties.DenySettings.Mode = to.Ptr(d.DenySettingsModeDenyDelete)
	identity.stacks[id] = invalid
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	if observer != nil || len(f.parent.factoryClients) != 0 {
		t.Fatal("inconsistent protection stack reached credential factory")
	}
	identity.stacks[id] = clone(baseline)
	observer, err = g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrap := f.parent.bootstrapReads
	invalid = clone(baseline)
	invalid.Properties.Resources[0].DenyStatus = to.Ptr(d.DenyStatusModeRemovedBySystem)
	identity.stacks[id] = invalid
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 0 || f.parent.bootstrapReads != bootstrap {
		t.Fatal("lost stack protection reached host reads or repeated bootstrap")
	}
	identity.stacks[id] = clone(baseline)
	if err := observer.Check(t.Context(), c); err != nil || f.reads != 3 {
		t.Fatal("fresh stack restoration rejected", err, f.reads)
	}
	identity.stacks[id] = d.DeploymentStack{}
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 3 || len(f.parent.factoryClients) != 1 {
		t.Fatal("missing stack metadata reused a successful host observation")
	}
}
