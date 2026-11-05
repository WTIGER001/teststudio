package azureworker

import (
	"net/http"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type preparationPublisherFixture struct {
	t      *testing.T
	mu     sync.Mutex
	routes map[string]policy.Transporter
	reads  int
	closed int
}

func (f *preparationPublisherFixture) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.reads++
	base := f.routes[r.URL.Path]
	f.mu.Unlock()
	if base == nil {
		f.t.Error("publisher reached an unselected resource")
		return deadlineResponse(r, 404, nil), nil
	}
	return base.Do(r)
}

func (f *preparationPublisherFixture) CloseIdleConnections() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
}

func TestAzurePreparationPublisherComposesOwnedReadersAndClosesWithoutPublication(t *testing.T) {
	c, s, writer, network, g := preparationWriterTest(t)
	registry := g.registry.registry
	endpoints := registry.endpoints
	zones := endpoints.dns
	dns := zones.dns
	firewall := dns.firewall
	perimeter := firewall.perimeter
	topology := perimeter.topology
	f := &preparationPublisherFixture{t: t, routes: map[string]policy.Transporter{}}
	for _, h := range []*armReadHTTP{
		topology.reader.(*SDKTopologyReader).http,
		perimeter.reader.(*SDKPerimeterReader).http,
		firewall.reader.(*SDKFirewallReader).http,
		dns.reader.(*SDKDNSReader).http,
		zones.reader.(*SDKPrivateDNSReader).http,
		endpoints.reader.(*SDKEndpointReader).http,
		registry.reader.(*SDKRegistryReader).http,
		g.registry.reader.(*SDKPreparationRegistryReader).http,
		g.registry.dns.(*SDKPrivateDNSReader).http,
		g.reader.(*SDKPreparationWriterReader).http,
	} {
		for id := range h.resources {
			f.routes[id] = h.base
		}
	}
	settings := PreparationSettings{Worker: c, Writer: s}
	publisher, err := settings.OpenPublisher(fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(publisher.Close)
	if len(publisher.readers) != 10 || f.reads != 0 {
		t.Fatal("publisher construction made provider calls or lost an owned reader", len(publisher.readers), f.reads)
	}
	var selected string
	publisher.guard.credential = func(id string) (azcore.TokenCredential, error) {
		selected = id
		return fixtureCredential{}, nil
	}
	if err := publisher.Check(t.Context(), c); err != nil || selected != "" {
		t.Fatal("readiness check selected a writer credential", err)
	}
	client, err := publisher.OpenRegistry(t.Context(), c)
	if err != nil || client == nil || selected != writer.settings.ClientID || network.reads == 0 {
		t.Fatal("composed publisher failed selected admission", err, selected)
	}
	client.Close()
	publisher.Close()
	publisher.Close()
	if f.closed != 10 {
		t.Fatal("publisher did not close each owned reader once", f.closed)
	}
	before := f.reads
	_, err = publisher.OpenRegistry(t.Context(), c)
	requireDeadlineError(t, err, hostedexecution.ErrUnavailable)
	requireDeadlineError(t, publisher.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("closed publisher performed a provider operation")
	}
	_, err = settings.OpenPublisher(nil, f)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
}
