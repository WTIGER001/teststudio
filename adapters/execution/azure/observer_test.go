package azureworker

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureObserverBindsAllClientsWithoutBootstrapReuseAndClosesReaders(t *testing.T) {
	c, s, f, g := observerTest(t)
	observer, err := g.BindObserver(t.Context(), c, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrapReads := f.bootstrapReads
	if bootstrapReads == 0 || f.identity.reads != 5 || f.reads != 0 || f.credential.reads.Load() != 0 || len(f.factoryClients) != 1 || f.factoryClients[0] != s.ClientID || len(observer.readers) != f.bootstrapReaders+1 {
		t.Fatal("observer binding did not preserve one bootstrap inspection and a pinned credential")
	}
	if _, ok := any(observer).(Network); ok {
		t.Fatal("credential binding became complete native network admission")
	}
	s.ClientID = "aaaaaaaa-1111-2222-3333-444444444444"
	s.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.RegistryEndpointIPs[0] = "8.8.8.8"
	for i := 1; i <= 2; i++ {
		if err := observer.Check(t.Context(), c); err != nil || f.reads != (bootstrapReads+1)*i || f.bootstrapReads != bootstrapReads || len(f.factoryClients) != 1 {
			t.Fatal("bound clients rejected fresh state or reused bootstrap", err, f.reads, bootstrapReads)
		}
	}
	if f.credential.reads.Load() == 0 {
		t.Fatal("bound SDK clients did not request their selected test credential")
	}
	f.identity.assignment.Properties.PrincipalID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
	requireDeadlineError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	if f.reads != (bootstrapReads+1)*3 || f.bootstrapReads != bootstrapReads {
		t.Fatal("bound observer cached a successful assignment")
	}
	before := f.reads
	changed := c
	changed.NetworkProfile = "other-profile"
	requireDeadlineError(t, observer.Check(t.Context(), changed), hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("changed runtime configuration reached a bound SDK reader")
	}
	observer.Close()
	if f.closed != f.bootstrapReaders+1 {
		t.Fatal("owned SDK readers did not close", f.closed, f.bootstrapReaders+1)
	}
	observer.Close()
	requireDeadlineError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.closed != f.bootstrapReaders+1 || f.reads != before {
		t.Fatal("closed observer reused transport or repeated cleanup")
	}
}

func TestAzureObserverPreservesBootstrapCredentialAndInspectionFailures(t *testing.T) {
	for _, mode := range []string{"bootstrap", "credential", "nil-credential", "token", "transport", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			c, _, f, g := observerTest(t)
			ctx := t.Context()
			want := hostedexecution.ErrUnavailable
			switch mode {
			case "bootstrap":
				f.identity.identity.Properties.PrincipalID = nil
			case "credential":
				g.credential = func(string) (azcore.TokenCredential, error) {
					return nil, errors.New("private-observer-factory-canary")
				}
			case "nil-credential":
				g.credential = func(string) (azcore.TokenCredential, error) { return nil, nil }
				want = hostedexecution.ErrConfiguration
			case "token":
				f.credential.fail.Store(true)
			case "transport":
				f.unavailable = true
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			observer, err := g.BindObserver(ctx, c, f)
			if mode == "token" || mode == "transport" {
				if err != nil {
					t.Fatal("bound observer rejected before its selected credential was used", err)
				}
				t.Cleanup(observer.Close)
				err = observer.Check(ctx, c)
			} else if observer != nil || f.reads != 0 || f.credential.reads.Load() != 0 {
				t.Fatal("rejected binding reached observer transport or credential")
			}
			requireDeadlineError(t, err, want)
			if mode == "bootstrap" && len(f.factoryClients) != 0 {
				t.Fatal("unverified bootstrap reached credential factory")
			}
			if mode == "token" && f.reads != 0 {
				t.Fatal("token failure reached management transport")
			}
		})
	}
}
