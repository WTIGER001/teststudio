package azureworker

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureRegistryServiceVerifiesFreshPrivateAuthorityAndRegion(t *testing.T) {
	c, s, f, g := registryServiceTest(t)
	if c.Region == s.region() {
		t.Fatal("fixture no longer distinguishes registry and worker regions")
	}
	if err := g.Check(t.Context(), c); err != nil || f.reads != 4 {
		t.Fatal("private registry rejected", err, f.reads)
	}
	s.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.PrivateEndpointIPs[0] = "8.8.8.8"
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			if err := g.Check(t.Context(), c); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if f.reads != 16 {
		t.Fatal("registry inspection reused stale state", f.reads)
	}
	slices.Reverse(f.connections)
	slices.Reverse(f.links[0].Properties.RequiredMembers)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("inventory order changed authority", err)
	}
	f.replicas = []*a.Replication{{ID: to.Ptr(s.replicas() + "/home"), Name: to.Ptr("home"), Type: to.Ptr("Microsoft.ContainerRegistry/registries/replications"), Location: to.Ptr("westus2"), Properties: &a.ReplicationProperties{ProvisioningState: to.Ptr(a.ProvisioningStateSucceeded), RegionEndpointEnabled: to.Ptr(true)}}}
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("explicit home-region observation rejected", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("registry service became complete network admission")
	}
	before := f.reads
	previous := g.endpoints.reader.(*SDKEndpointReader).http.base.(*endpointFixture)
	previous.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("unverified endpoint permitted registry reads")
	}
	previous.mode = ""
	c.NetworkProfile = "changed"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("configuration mismatch reached registry")
	}
}

func TestAzureRegistryServiceRejectsAccessAndDataBoundaryChanges(t *testing.T) {
	for name, mutate := range map[string]func(*registryFixture){
		"public-access": func(f *registryFixture) {
			f.registry.Properties.PublicNetworkAccess = to.Ptr(a.PublicNetworkAccessEnabled)
		},
		"anonymous-pull": func(f *registryFixture) { f.registry.Properties.AnonymousPullEnabled = to.Ptr(true) },
		"admin-password": func(f *registryFixture) { f.registry.Properties.AdminUserEnabled = to.Ptr(true) },
		"trusted-service-bypass": func(f *registryFixture) {
			f.registry.Properties.NetworkRuleBypassOptions = to.Ptr(a.NetworkRuleBypassOptionsAzureServices)
		},
		"task-bypass": func(f *registryFixture) { f.registry.Properties.NetworkRuleBypassAllowedForTasks = to.Ptr(true) },
		"export": func(f *registryFixture) {
			f.registry.Properties.Policies.ExportPolicy.Status = to.Ptr(a.ExportPolicyStatusEnabled)
		},
		"incompatible-pull-policy": func(f *registryFixture) {
			f.registry.Properties.Policies.AzureADAuthenticationAsArmPolicy.Status = to.Ptr(a.AzureADAuthenticationAsArmPolicyStatusDisabled)
		},
		"incompatible-role-mode": func(f *registryFixture) {
			f.registry.Properties.RoleAssignmentMode = to.Ptr(a.RoleAssignmentModeAbacRepositoryPermissions)
		},
		"public-ip-exception": func(f *registryFixture) {
			f.registry.Properties.NetworkRuleSet.IPRules = []*a.IPRule{{IPAddressOrRange: to.Ptr("0.0.0.0/0"), Action: to.Ptr(a.ActionAllow)}}
		},
		"wrong-primary-region": func(f *registryFixture) { f.registry.Location = to.Ptr("eastus2") },
		"wrong-data-region": func(f *registryFixture) {
			f.registry.Properties.DataEndpointHostNames = []*string{to.Ptr("studio.eastus2.data.azurecr.io")}
		},
		"foreign-data-host": func(f *registryFixture) {
			f.registry.Properties.DataEndpointHostNames = []*string{to.Ptr("foreign.westus2.data.azurecr.io")}
		},
		"foreign-login": func(f *registryFixture) { f.registry.Properties.LoginServer = to.Ptr("foreign.azurecr.io") },
		"different-sku": func(f *registryFixture) { f.registry.SKU.Name = to.Ptr(a.SKUNameStandard) },
		"extra-region-member": func(f *registryFixture) {
			f.links[0].Properties.RequiredMembers = append(f.links[0].Properties.RequiredMembers, to.Ptr("registry_data_eastus2"))
		},
		"different-zone": func(f *registryFixture) {
			f.links[0].Properties.RequiredZoneNames = []*string{to.Ptr("foreign.invalid")}
		},
		"foreign-replica": func(f *registryFixture) {
			f.replicas = []*a.Replication{{ID: to.Ptr(f.settings.replicas() + "/eastus2"), Name: to.Ptr("eastus2"), Type: to.Ptr("Microsoft.ContainerRegistry/registries/replications"), Location: to.Ptr("eastus2"), Properties: &a.ReplicationProperties{ProvisioningState: to.Ptr(a.ProvisioningStateSucceeded), RegionEndpointEnabled: to.Ptr(true)}}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := registryServiceTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureRegistryServiceRejectsForeignOrUnapprovedConnections(t *testing.T) {
	for name, mutate := range map[string]func(*registryFixture){
		"foreign-endpoint": func(f *registryFixture) {
			f.connections[0].Properties.PrivateEndpoint.ID = to.Ptr(f.settings.Endpoints.ExchangeEndpoint)
		},
		"duplicate-endpoint": func(f *registryFixture) {
			f.connections[1].Properties.PrivateEndpoint.ID = f.connections[0].Properties.PrivateEndpoint.ID
		},
		"duplicate-case-insensitive-id": func(f *registryFixture) {
			f.connections[1].Name = to.Ptr(strings.ToUpper(value(f.connections[0].Name)))
			f.connections[1].ID = to.Ptr(strings.ToUpper(value(f.connections[0].ID)))
		},
		"foreign-parent": func(f *registryFixture) { f.connections[0].ID = to.Ptr("foreign") },
		"rejected": func(f *registryFixture) {
			f.connections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.ConnectionStatusRejected)
		},
		"disconnected-preparation": func(f *registryFixture) {
			f.connections[1].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.ConnectionStatusDisconnected)
		},
		"extra-connection": func(f *registryFixture) { f.connections = append(f.connections, clone(f.connections[0])) },
		"embedded-foreign": func(f *registryFixture) {
			f.registry.Properties.PrivateEndpointConnections[0].Properties.PrivateEndpoint.ID = to.Ptr(f.settings.Endpoints.ExchangeEndpoint)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := registryServiceTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureRegistryServicePreservesIncompleteStateAndBoundedPaging(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "redirect", "oversized", "pending-registry", "unknown-public-access", "unknown-task-bypass", "unknown-policy", "unknown-policy-status", "missing-data-host", "pending-connection", "required-action", "missing-connection", "missing-embedded", "mismatched-snapshot", "missing-link", "cancelled", "paged", "foreign-page", "foreign-list", "cycle", "endless"} {
		t.Run(mode, func(t *testing.T) {
			c, s, f, g := registryServiceTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "pending-registry":
				f.registry.Properties.ProvisioningState = to.Ptr(a.ProvisioningStateUpdating)
			case "unknown-public-access":
				f.registry.Properties.PublicNetworkAccess = nil
			case "unknown-task-bypass":
				f.registry.Properties.NetworkRuleBypassAllowedForTasks = nil
			case "unknown-policy":
				f.registry.Properties.Policies = nil
			case "unknown-policy-status":
				f.registry.Properties.Policies.AzureADAuthenticationAsArmPolicy.Status = nil
			case "missing-data-host":
				f.registry.Properties.DataEndpointHostNames = nil
			case "pending-connection":
				f.connections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.ConnectionStatusPending)
			case "required-action":
				f.connections[0].Properties.PrivateLinkServiceConnectionState.ActionsRequired = to.Ptr(a.ActionsRequiredRecreate)
			case "missing-connection":
				f.connections = f.connections[:1]
			case "missing-embedded":
				f.registry.Properties.PrivateEndpointConnections = nil
			case "mismatched-snapshot":
				f.registry.Properties.PrivateEndpointConnections[0].Name = to.Ptr("another.approval")
				f.registry.Properties.PrivateEndpointConnections[0].ID = to.Ptr(s.connections() + "/another.approval")
			case "missing-link":
				f.links = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := g.Check(ctx, c)
			if mode == "paged" {
				if err != nil || f.pages[s.connections()] != 2 {
					t.Fatal("safe complete paging rejected", err)
				}
				return
			}
			expected := hostedexecution.ErrUnavailable
			if mode == "foreign-page" || mode == "foreign-list" || mode == "cycle" {
				expected = hostedexecution.ErrOwnership
			}
			requireDeadlineError(t, err, expected)
			if f.reads > 6 || f.pages[s.connections()] > 3 {
				t.Fatal("registry inventory escaped bounds", f.reads)
			}
		})
	}
}
