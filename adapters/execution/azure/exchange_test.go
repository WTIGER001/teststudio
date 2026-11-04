package azureworker

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureExchangeVerifiesFreshSelectedBindingAndSafeDefaults(t *testing.T) {
	c, s, f, g := exchangeServiceTest(t)
	if err := g.Check(t.Context(), c); err != nil || f.reads != 8 {
		t.Fatal("selected relay binding rejected", err, f.reads)
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
	if f.reads != 32 {
		t.Fatal("exchange binding reused stale observations", f.reads)
	}
	f.application.Kind = to.Ptr(a.Kind("containerapps"))
	f.application.Properties.Template.RevisionSuffix = to.Ptr("r1")
	f.application.Properties.Template.Containers[0].Resources.EphemeralStorage = to.Ptr("1Gi")
	f.application.Properties.Configuration.Dapr = &a.Dapr{Enabled: to.Ptr(false)}
	ingress := f.application.Properties.Configuration.Ingress
	ingress.ClientCertificateMode = to.Ptr(a.IngressClientCertificateModeIgnore)
	ingress.ExposedPort = to.Ptr[int32](0)
	ingress.StickySessions = &a.IngressStickySessions{Affinity: to.Ptr(a.AffinityNone)}
	slices.Reverse(f.application.Properties.Template.Containers[0].Probes)
	slices.Reverse(f.revisions)
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("inactive defaults, metadata or ordering changed authority", err)
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("selected binding became complete network admission")
	}
	before := f.reads
	previous := g.registry.reader.(*SDKRegistryReader).http.base.(*registryFixture)
	previous.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("unverified registry permitted exchange reads")
	}
	previous.mode = ""
	c.NetworkProfile = "changed"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("changed profile reached exchange resources")
	}
}

func TestAzureExchangeRejectsRuntimeAuthorityAndRouteChanges(t *testing.T) {
	cases := map[string]func(*exchangeFixture){
		"public-environment": func(f *exchangeFixture) {
			f.environment.Properties.PublicNetworkAccess = to.Ptr(a.PublicNetworkAccessEnabled)
		},
		"wrong-subnet": func(f *exchangeFixture) {
			f.environment.Properties.VnetConfiguration.InfrastructureSubnetID = to.Ptr(f.settings.Registry.Endpoints.Subnet)
		},
		"environment-identity": func(f *exchangeFixture) {
			f.environment.Identity = &a.ManagedServiceIdentity{Type: to.Ptr(a.ManagedServiceIdentityTypeSystemAssigned)}
		},
		"runtime-identity": func(f *exchangeFixture) {
			f.application.Properties.Configuration.IdentitySettings[0].Lifecycle = to.Ptr(a.IdentitySettingsLifeCycleMain)
		},
		"extra-identity": func(f *exchangeFixture) {
			f.application.Identity.UserAssignedIdentities["foreign"] = &a.UserAssignedIdentity{}
		},
		"credential-env": func(f *exchangeFixture) {
			f.application.Properties.Template.Containers[0].Env = []*a.EnvironmentVar{{Name: to.Ptr("TESTSTUDIO_DATABASE"), Value: to.Ptr("private-database-canary")}}
		},
		"credential-secret": func(f *exchangeFixture) {
			f.application.Properties.Configuration.Secrets = []*a.Secret{{Name: to.Ptr("private"), Value: to.Ptr("private-secret-canary")}}
		},
		"registry-password": func(f *exchangeFixture) {
			f.application.Properties.Configuration.Registries[0].PasswordSecretRef = to.Ptr("private")
		},
		"api-entrypoint": func(f *exchangeFixture) {
			f.application.Properties.Template.Containers[0].Command = []*string{to.Ptr("/app/teststudio")}
		},
		"foreign-image": func(f *exchangeFixture) {
			f.application.Properties.Template.Containers[0].Image = to.Ptr("foreign.invalid/service@sha256:" + strings.Repeat("a", 64))
		},
		"foreign-backend": func(f *exchangeFixture) {
			f.application.Properties.Template.Containers[0].Args[2] = to.Ptr("--backend-origin=https://foreign.invalid")
		},
		"sidecar": func(f *exchangeFixture) {
			p := f.application.Properties.Template
			p.Containers = append(p.Containers, clone(p.Containers[0]))
		},
		"init-container": func(f *exchangeFixture) {
			f.application.Properties.Template.InitContainers = []*a.InitContainer{{Name: to.Ptr("foreign"), Image: to.Ptr("foreign.invalid/tool")}}
		},
		"volume": func(f *exchangeFixture) {
			f.application.Properties.Template.Volumes = []*a.Volume{{Name: to.Ptr("private")}}
		},
		"insecure-ingress": func(f *exchangeFixture) { f.application.Properties.Configuration.Ingress.AllowInsecure = to.Ptr(true) },
		"extra-port": func(f *exchangeFixture) {
			f.application.Properties.Configuration.Ingress.AdditionalPortMappings = []*a.IngressPortMapping{{External: to.Ptr(true), TargetPort: to.Ptr[int32](8040)}}
		},
		"missing-domain-object": func(f *exchangeFixture) { f.application.Properties.Configuration.Ingress.CustomDomains[0] = nil },
		"extra-domain": func(f *exchangeFixture) {
			p := f.application.Properties.Configuration.Ingress
			p.CustomDomains = append(p.CustomDomains, clone(p.CustomDomains[0]))
		},
		"easy-auth":         func(f *exchangeFixture) { f.auth.Properties.Platform.Enabled = to.Ptr(true) },
		"environment-route": func(f *exchangeFixture) { f.routes = []*a.HTTPRouteConfig{{Name: to.Ptr("foreign")}} },
		"active-revision-command": func(f *exchangeFixture) {
			f.revisions[0].Properties.Template.Containers[0].Command = []*string{to.Ptr("/app/teststudio")}
		},
		"duplicate-revision-identity-case": func(f *exchangeFixture) {
			first := value(f.revisions[0].Name)
			f.revisions[1].Name = to.Ptr(strings.TrimSuffix(first, "r1") + "R1")
			f.revisions[1].ID = to.Ptr(strings.ToUpper(value(f.revisions[0].ID)))
		},
		"second-active-revision": func(f *exchangeFixture) { f.revisions[1].Properties.Active = to.Ptr(true) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := exchangeServiceTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzureExchangeRejectsForeignApprovalAndTLSBindings(t *testing.T) {
	for name, mutate := range map[string]func(*exchangeFixture){
		"foreign-endpoint": func(f *exchangeFixture) {
			f.connections[0].Properties.PrivateEndpoint.ID = to.Ptr(f.settings.Registry.PreparationEndpoint)
		},
		"rejected-endpoint": func(f *exchangeFixture) {
			f.connections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.PrivateEndpointServiceConnectionStatusRejected)
		},
		"extra-endpoint":        func(f *exchangeFixture) { f.connections = append(f.connections, clone(f.connections[0])) },
		"foreign-service-group": func(f *exchangeFixture) { f.links[0].Properties.GroupID = to.Ptr("sites") },
		"foreign-service-zone": func(f *exchangeFixture) {
			f.links[0].Properties.RequiredZoneNames = []*string{to.Ptr("privatelink.foreign.invalid")}
		},
		"wrong-certificate": func(f *exchangeFixture) {
			f.application.Properties.Configuration.Ingress.CustomDomains[0].CertificateID = to.Ptr(f.settings.Certificate + "-foreign")
		},
		"disabled-tls-binding": func(f *exchangeFixture) {
			f.application.Properties.Configuration.Ingress.CustomDomains[0].BindingType = to.Ptr(a.BindingTypeDisabled)
		},
		"wrong-certificate-host": func(f *exchangeFixture) {
			f.certificate.Properties.SubjectAlternativeNames = []*string{to.Ptr("foreign.invalid")}
		},
		"private-key-material":       func(f *exchangeFixture) { f.certificate.Properties.Value = []byte("private-pfx-canary") },
		"foreign-certificate-parent": func(f *exchangeFixture) { f.certificate.ID = to.Ptr(f.settings.Registry.Endpoints.ExchangeEndpoint) },
		"foreign-default-domain": func(f *exchangeFixture) {
			f.application.Properties.Configuration.Ingress.Fqdn = to.Ptr("foreign.invalid")
		},
		"foreign-auth-object": func(f *exchangeFixture) { f.auth.ID = to.Ptr("foreign") },
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, g := exchangeServiceTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
	c, _, f, g := exchangeServiceTest(t)
	f.certificate.Properties.SubjectAlternativeNames = []*string{to.Ptr("*.studio.internal.example.test")}
	if err := g.Check(t.Context(), c); err != nil {
		t.Fatal("standard single-label wildcard certificate rejected", err)
	}
}

func TestAzureExchangePreservesPendingSnapshotsAndBoundedRevisionPaging(t *testing.T) {
	for _, mode := range []string{"unavailable", "missing", "redirect", "oversized", "pending-environment", "pending-approval", "approval-update", "missing-approval", "missing-embedded", "changed-snapshot", "unknown-auth", "pending-certificate", "short-certificate-life", "unknown-certificate-life", "pending-application", "rollout", "missing-revisions", "inactive-replicas", "unknown-inactive-replicas", "pending-revision", "cancelled", "paged", "foreign-page", "other-list", "cycle", "endless"} {
		t.Run(mode, func(t *testing.T) {
			c, s, f, g := exchangeServiceTest(t)
			f.mode = mode
			ctx := t.Context()
			switch mode {
			case "pending-environment":
				f.environment.Properties.ProvisioningState = to.Ptr(a.EnvironmentProvisioningStateWaiting)
			case "pending-approval":
				f.connections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.PrivateEndpointServiceConnectionStatusPending)
			case "approval-update":
				f.connections[0].Properties.PrivateLinkServiceConnectionState.ActionsRequired = to.Ptr("Recreate")
			case "missing-approval":
				f.connections = nil
			case "missing-embedded":
				f.environment.Properties.PrivateEndpointConnections = nil
			case "changed-snapshot":
				f.environment.Properties.PrivateEndpointConnections[0].Name = to.Ptr("changed")
				f.environment.Properties.PrivateEndpointConnections[0].ID = to.Ptr(s.connections() + "/changed")
			case "unknown-auth":
				f.auth.Properties.Platform.Enabled = nil
			case "pending-certificate":
				f.certificate.Properties.ProvisioningState = to.Ptr(a.CertificateProvisioningStatePending)
			case "short-certificate-life":
				f.certificate.Properties.ExpirationDate = to.Ptr(time.Now().Add(time.Hour))
			case "unknown-certificate-life":
				f.certificate.Properties.ExpirationDate = nil
			case "pending-application":
				f.application.Properties.ProvisioningState = to.Ptr(a.ContainerAppProvisioningStateInProgress)
			case "rollout":
				f.application.Properties.LatestReadyRevisionName = to.Ptr("old")
			case "missing-revisions":
				f.revisions = nil
			case "inactive-replicas":
				f.revisions[1].Properties.Replicas = to.Ptr[int32](1)
			case "unknown-inactive-replicas":
				f.revisions[1].Properties.Replicas = nil
			case "pending-revision":
				f.revisions[0].Properties.ProvisioningState = to.Ptr(a.RevisionProvisioningStateProvisioning)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := g.Check(ctx, c)
			if mode == "paged" {
				if err != nil || f.pages[s.revisions()] != 2 {
					t.Fatal("complete revision paging rejected", err)
				}
				return
			}
			expected := hostedexecution.ErrUnavailable
			if mode == "foreign-page" || mode == "other-list" || mode == "cycle" {
				expected = hostedexecution.ErrOwnership
			}
			requireDeadlineError(t, err, expected)
			if f.reads > 10 || f.pages[s.revisions()] > 3 {
				t.Fatal("exchange inspection escaped its read bound", f.reads)
			}
		})
	}
}
