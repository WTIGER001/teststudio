package browsergateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestPinnedRemoteAddressKeepsHostnameAndCertificateVerification(t *testing.T) {
	session := remoteTestSession()
	target := browserpolicy.Target{Origin: "http://application.fixture.test"}
	gateway, err := NewRemote(Configuration{Target: target, Remote: &session})
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	authority := testkit.NewCertificateAuthority(t, "pinned gateway")
	var delivered atomic.Int32
	server := testkit.NewTLSServer(t, authority, "gateway.fixture.test", time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delivered.Add(1)
		gateway.ServeHTTP(w, r)
	}))
	address, _ := url.Parse(server.URL)
	connection := RemoteConnection{Origin: "https://gateway.fixture.test:" + address.Port(), Address: "127.0.0.1", CertificateAuthority: authority.PEM, Session: session}
	for _, mode := range []string{"valid", "wrong hostname", "wrong authority", "wrong capability"} {
		t.Run(mode, func(t *testing.T) {
			selected := connection
			want := http.StatusForbidden
			switch mode {
			case "wrong hostname":
				selected.Origin = "https://another.fixture.test:" + address.Port()
				want = http.StatusBadGateway
			case "wrong authority":
				selected.CertificateAuthority = testkit.NewCertificateAuthority(t, "unrelated gateway").PEM
				want = http.StatusBadGateway
			case "wrong capability":
				selected.Session.Token = remoteTestSession().Token[:42] + "A"
				want = http.StatusProxyAuthRequired
			}
			bridge, err := newRemoteBridge(target, selected)
			if err != nil {
				t.Fatal(err)
			}
			if bridge.address != address.Host || bridge.tls.ServerName != selectedOriginHost(t, selected.Origin) {
				t.Fatal("address pinning changed TLS identity or selected another connection address")
			}
			before := delivered.Load()
			response := httptest.NewRecorder()
			bridge.ServeHTTP(response, httptest.NewRequest("GET", "http://outside.fixture.test/status", nil))
			if response.Code != want {
				t.Fatal("pinned gateway verification returned an unexpected status", response.Code, want)
			}
			if want == http.StatusBadGateway && delivered.Load() != before || want != http.StatusBadGateway && delivered.Load() != before+1 {
				t.Fatal("certificate failure reached the gateway handler or valid pin did not reach it")
			}
		})
	}
}

func selectedOriginHost(t *testing.T, origin string) string {
	t.Helper()
	value, err := url.Parse(origin)
	if err != nil {
		t.Fatal(err)
	}
	return value.Hostname()
}

func TestRemoteEndpointRejectsUnverifiedAddressForms(t *testing.T) {
	for _, address := range []string{"other.fixture.test", "8.8.8.8", "169.254.169.254", "0.0.0.0", "224.0.0.1", "10.1.2.3:8044", "::ffff:10.1.2.3", "fd00::1", "10.01.2.3"} {
		if ValidateRemoteEndpoint("https://gateway.fixture.test:8044", address) == nil {
			t.Fatal("an unsupported pinned address was accepted", address)
		}
	}
	for _, address := range []string{"", "10.1.2.3", "172.20.1.2", "192.168.3.4", "127.0.0.1"} {
		if err := ValidateRemoteEndpoint("https://gateway.fixture.test:8044", address); err != nil {
			t.Fatal("canonical private or local fixture endpoint rejected", address, err)
		}
	}
}
