package azureworker

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	n "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePreparationNetworkBindsFreshSourceEgressAndPrivateJournal(t *testing.T) {
	s, f, g := preparationNetworkTest(t)
	s.Network.DatabaseIP = "10.21.5.9"
	for i := 1; i <= 2; i++ {
		if err := g.Check(t.Context(), s.Worker); err != nil || f.reads != 11*i || f.host.reads != 7*i || f.host.parent.reads != 3*i {
			t.Fatal("source network was cached, rejected, or repeated its parent", err, f.reads, f.host.reads)
		}
	}
	f.o.account.Properties.PublicNetworkAccess = to.Ptr(a.PublicNetworkAccessEnabled)
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrOwnership)
	before := f.reads
	f.host.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrUnavailable)
	if f.reads != before {
		t.Fatal("failed host observation reached source network inspection")
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("trusted source network became untrusted execution admission")
	}
}

func TestAzurePreparationNetworkRejectsAlternateEgressAndStoragePaths(t *testing.T) {
	for name, mutate := range map[string]func(*preparationNetworkFixture){
		"source-route": func(f *preparationNetworkFixture) {
			f.o.source.Properties.RouteTable = &n.RouteTable{ID: to.Ptr("foreign-route")}
		},
		"source-nat": func(f *preparationNetworkFixture) {
			f.o.source.Properties.NatGateway = &n.SubResource{ID: to.Ptr("foreign-nat")}
		},
		"source-delegation": func(f *preparationNetworkFixture) {
			f.o.source.Properties.Delegations[0].Properties.ServiceName = to.Ptr("Microsoft.Web/serverFarms")
		},
		"public-database": func(f *preparationNetworkFixture) {
			f.o.sourceNSG.Properties.SecurityRules[len(f.o.sourceNSG.Properties.SecurityRules)-3].Properties.DestinationAddressPrefix = to.Ptr("*")
		},
		"custom-dns": func(f *preparationNetworkFixture) {
			f.o.network.Properties.DhcpOptions = &n.DhcpOptions{DNSServers: []*string{to.Ptr("10.21.0.9")}}
		},
		"nat-consumer": func(f *preparationNetworkFixture) {
			f.o.nat.Properties.Subnets = append(f.o.nat.Properties.Subnets, &n.SubResource{ID: to.Ptr("foreign-subnet")})
		},
		"nat-prefix": func(f *preparationNetworkFixture) {
			f.o.nat.Properties.PublicIPPrefixes = []*n.SubResource{{ID: to.Ptr("foreign-prefix")}}
		},
		"nat-backlink":        func(f *preparationNetworkFixture) { f.o.ip.Properties.NatGateway.ID = to.Ptr("foreign-nat") },
		"private-nat-address": func(f *preparationNetworkFixture) { f.o.ip.Properties.IPAddress = to.Ptr("10.21.0.10") },
		"file-group": func(f *preparationNetworkFixture) {
			f.o.endpoint.Properties.PrivateLinkServiceConnections[0].Properties.GroupIDs = []*string{to.Ptr("blob")}
		},
		"file-nic-address": func(f *preparationNetworkFixture) {
			f.o.nic.Properties.IPConfigurations[0].Properties.PrivateIPAddress = to.Ptr("10.21.4.9")
		},
		"file-public-access": func(f *preparationNetworkFixture) {
			f.o.account.Properties.PublicNetworkAccess = to.Ptr(a.PublicNetworkAccessEnabled)
		},
		"file-bypass": func(f *preparationNetworkFixture) {
			f.o.account.Properties.NetworkRuleSet.Bypass = to.Ptr(a.BypassAzureServices)
		},
		"file-extra-consumer": func(f *preparationNetworkFixture) {
			f.o.account.Properties.PrivateEndpointConnections = append(f.o.account.Properties.PrivateEndpointConnections, f.o.account.Properties.PrivateEndpointConnections[0])
		},
		"file-approval-owner": func(f *preparationNetworkFixture) {
			f.o.account.Properties.PrivateEndpointConnections[0].Properties.PrivateEndpoint.ID = to.Ptr("foreign-endpoint")
		},
		"file-other-host": func(f *preparationNetworkFixture) {
			f.o.account.Properties.PrimaryEndpoints.File = to.Ptr("https://foreign.file.core.windows.net/")
		},
		"file-https-off": func(f *preparationNetworkFixture) { f.o.account.Properties.EnableHTTPSTrafficOnly = to.Ptr(false) },
		"nfs-share": func(f *preparationNetworkFixture) {
			f.o.share.FileShareProperties.EnabledProtocols = to.Ptr(a.EnabledProtocolsNFS)
		},
	} {
		t.Run(name, func(t *testing.T) {
			s, f, g := preparationNetworkTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrOwnership)
		})
	}
}

func TestAzurePreparationNetworkPreservesUncertaintyAndRestrictsSDKOperations(t *testing.T) {
	s, f, g := preparationNetworkTest(t)
	for _, mode := range []string{"unavailable", "redirect"} {
		f.mode = mode
		requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrUnavailable)
	}
	f.mode = ""
	f.o.share.FileShareProperties.EnabledProtocols = nil
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrUnavailable)
	f.o.share.FileShareProperties.EnabledProtocols = to.Ptr(a.EnabledProtocolsSMB)
	f.o.account.Properties.PrivateEndpointConnections[0].Properties.PrivateLinkServiceConnectionState.Status = to.Ptr(a.PrivateEndpointServiceConnectionStatusPending)
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrUnavailable)
	f.o.account.Properties.PrivateEndpointConnections[0].Properties.PrivateLinkServiceConnectionState.Status = nil
	requireDeadlineError(t, g.Check(t.Context(), s.Worker), hostedexecution.ErrUnavailable)
	reader := g.reader.(*SDKPreparationNetworkReader)
	before := f.reads
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "POST" },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Path += "/fileServices/default/shares" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$expand=kerb" },
	} {
		r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Network.StorageAccount+"?api-version="+preparationStorageVersion, nil)
		mutate(r)
		_, err := reader.http.Do(r)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err := reader.Account(t.Context(), s.Network.StorageAccount+"-foreign")
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != before {
		t.Fatal("unselected storage or network operation reached transport")
	}
}

func TestAzurePreparationNetworkPublisherGatesWriterAndClosesAllReaders(t *testing.T) {
	s, f, g := preparationNetworkTest(t)
	transport := preparationPublisherTransport(t, g.host.writer)
	for id := range g.host.reader.(*SDKPreparationHostReader).http.resources {
		transport.routes[id] = f.host
	}
	for id := range g.reader.(*SDKPreparationNetworkReader).http.resources {
		transport.routes[id] = f
	}
	for id := range g.dns.(*SDKPrivateDNSReader).http.resources {
		transport.routes[id] = f.dns
	}
	publisher, err := s.OpenPublisher(fixtureCredential{}, transport)
	if err != nil || publisher == nil || len(publisher.readers) != 13 || transport.reads != 0 {
		t.Fatal("native publisher lost a reader or performed constructor requests", err)
	}
	t.Cleanup(publisher.Close)
	credentials := 0
	publisher.guard.credential = func(string) (azcore.TokenCredential, error) { credentials++; return fixtureCredential{}, nil }
	client, err := publisher.OpenRegistry(t.Context(), s.Worker)
	if err != nil || client == nil || credentials != 1 {
		t.Fatal("complete source-network observation rejected", err)
	}
	client.Close()
	f.o.share.FileShareProperties.EnabledProtocols = to.Ptr(a.EnabledProtocolsNFS)
	_, err = publisher.OpenRegistry(t.Context(), s.Worker)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if credentials != 1 {
		t.Fatal("unverified journal network reached writer credential selection")
	}
	publisher.Close()
	publisher.Close()
	if transport.closed != 13 {
		t.Fatal("source network publisher did not release every owned reader once", transport.closed)
	}
}

func TestAzurePreparationNetworkSettingsAndDefinitionsKeepSeparateRoles(t *testing.T) {
	s, _, _ := preparationNetworkTest(t)
	for _, mutate := range []func(*PreparationNetworkSettings){
		func(p *PreparationNetworkSettings) { p.FileSubnetPrefix = s.Writer.Registry.SourceSubnetPrefix },
		func(p *PreparationNetworkSettings) { p.DatabaseIP = "10.20.1.4" },
		func(p *PreparationNetworkSettings) { p.SourceSecurityGroup = s.Writer.Registry.SecurityGroup },
		func(p *PreparationNetworkSettings) { p.FileEndpoint = s.Writer.Registry.Registry.PreparationEndpoint },
		func(p *PreparationNetworkSettings) { p.FileIP = "10.21.4.1" },
		func(p *PreparationNetworkSettings) { p.StorageAccount += "-foreign" },
	} {
		changed := *s.Network
		mutate(&changed)
		copy := s
		copy.Network = &changed
		requireDeadlineError(t, copy.ValidateNative(), hostedexecution.ErrConfiguration)
	}
	group, err := s.PreparationSourceSecurityGroupDefinition()
	if err != nil {
		t.Fatal(err)
	}
	group.Properties.SecurityRules[0].Properties.SourceAddressPrefix = to.Ptr("*")
	fresh, _ := s.PreparationSourceSecurityGroupDefinition()
	if value(fresh.Properties.SecurityRules[0].Properties.SourceAddressPrefix) == "*" {
		t.Fatal("caller mutation changed the source network policy")
	}
	link, _ := s.PreparationFileLinkDefinition()
	records, _ := s.PreparationFileRecordDefinitions()
	if value(link.Properties.VirtualNetwork.ID) != s.Writer.Registry.network() || value(link.Properties.RegistrationEnabled) || len(records) != 1 || value(records[s.Host.JournalAccount].Properties.ARecords[0].IPv4Address) != s.Network.FileIP {
		t.Fatal("file DNS definitions did not select the controller and journal address")
	}
}
