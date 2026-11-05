package azureworker

import (
	"context"
	"net/http"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureObserverHostRoutesRequireExplicitEmptySinglePage(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	reader, err := NewSDKObserverHostReader(c, s, f.parent.credential, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	for _, check := range []struct {
		mode string
		want error
	}{
		{"present", hostedexecution.ErrOwnership},
		{"present-and-next", hostedexecution.ErrOwnership},
		{"missing-array", hostedexecution.ErrUnavailable},
		{"null-array", hostedexecution.ErrUnavailable},
		{"null-row", hostedexecution.ErrUnavailable},
		{"same-next", hostedexecution.ErrUnavailable},
		{"foreign-next", hostedexecution.ErrUnavailable},
		{"missing", hostedexecution.ErrUnavailable},
		{"unavailable", hostedexecution.ErrUnavailable},
		{"redirect", hostedexecution.ErrUnavailable},
		{"", nil},
	} {
		t.Run(check.mode, func(t *testing.T) {
			f.routesMode = check.mode
			before := f.routeReads
			err := reader.RoutesAbsent(t.Context(), c.ControllerEnvironment)
			if check.want == nil {
				if err != nil {
					t.Fatal("explicit complete empty route collection rejected", err)
				}
			} else {
				requireReadDenyError(t, err, check.want)
			}
			if f.routeReads != before+1 {
				t.Fatal("route absence repeated a request or followed a continuation", f.routeReads-before)
			}
		})
	}
	if f.parent.bootstrapReads != 0 {
		t.Fatal("direct route observation repeated parent inspection")
	}
}

func TestAzureObserverHostRoutesSDKRejectsOtherScopesAndContinuationRequests(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	reader, err := NewSDKObserverHostReader(c, s, f.parent.credential, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	for _, id := range []string{c.WorkerEnvironment, s.Application, c.ControllerEnvironment + "/httpRouteConfigs"} {
		requireReadDenyError(t, reader.RoutesAbsent(t.Context(), id), hostedexecution.ErrOwnership)
	}
	for _, change := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.URL.Path += "/foreign" },
		func(r *http.Request) { r.URL.Path = c.WorkerEnvironment + "/httpRouteConfigs" },
		func(r *http.Request) { r.URL.RawQuery += "&%24skipToken=private-next-canary" },
		func(r *http.Request) { r.URL.RawQuery += "&%24filter=private-filter-canary" },
		func(r *http.Request) { r.URL.RawQuery += "&%24top=1" },
	} {
		request, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+c.ControllerEnvironment+"/httpRouteConfigs?api-version="+exchangeVersion, nil)
		change(request)
		_, err := reader.service.http.Do(request)
		requireReadDenyError(t, err, hostedexecution.ErrOwnership)
	}
	if f.reads != 0 || f.routeReads != 0 {
		t.Fatal("unselected route operation reached transport")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	requireReadDenyError(t, reader.RoutesAbsent(ctx, c.ControllerEnvironment), hostedexecution.ErrUnavailable)
}

func TestAzureObserverHostRoutesRefreshBoundCheckWithoutBootstrapReuse(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrap := f.parent.bootstrapReads
	if f.reads != 0 || f.routeReads != 0 {
		t.Fatal("host binding requested route state before observer inspection")
	}
	if err := observer.Check(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	f.routesMode = "present"
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	f.routesMode = "same-next"
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.routesMode = ""
	if err := observer.Check(t.Context(), c); err != nil {
		t.Fatal("restored route absence rejected", err)
	}
	if f.reads != 16 || f.routeReads != 4 || f.parent.bootstrapReads != bootstrap || len(f.parent.factoryClients) != 1 {
		t.Fatal("route observation repeated bootstrap, followed pagination, or reused prior state", f.reads, f.routeReads)
	}
	f.parent.unavailable = true
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 16 || f.routeReads != 4 {
		t.Fatal("unverified parent chain reached controller route observation")
	}
}
