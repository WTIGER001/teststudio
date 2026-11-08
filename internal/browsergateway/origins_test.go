package browsergateway

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

func TestGatewayAdditionalOriginsKeepExactScopeAndCredentialIsolation(t *testing.T) {
	assets := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "approved website module")
	}))
	defer assets.Close()
	actual, _ := url.Parse(assets.URL)
	origin := "https://assets.fixture.test:" + actual.Port()
	target := browserpolicy.Target{Origin: "http://website.fixture.test", AdditionalOrigins: []string{origin}, AllowPrivate: true}
	gateway, err := New(target)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	originalDial := gateway.dial
	gateway.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "assets.fixture.test:"+actual.Port() {
			t.Fatal("an unreviewed authority reached the network")
		}
		return originalDial(ctx, network, actual.Host)
	}
	proxy := httptest.NewServer(gateway)
	defer proxy.Close()
	address, _ := url.Parse(proxy.URL)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(address), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	response, err := client.Get(origin + "/bootstrap.js")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || string(data) != "approved website module" {
		t.Fatal("the reviewed cross-origin module did not load")
	}
	for _, foreign := range []string{"https://outside.fixture.test", "https://assets.fixture.test:8443", "http://outside.fixture.test"} {
		response, err := client.Get(foreign)
		if err == nil {
			response.Body.Close()
			if response.StatusCode != 403 {
				t.Fatal("an unreviewed origin or port was allowed")
			}
		}
	}
	target.Authenticated = true
	if _, err := New(target); err == nil {
		t.Fatal("a credential-bearing target accepted additional origins")
	}
}

func TestGatewayApprovedOriginStillCannotReachMetadataOrPrivateAddresses(t *testing.T) {
	for _, forbidden := range []string{"https://169.254.169.254", "https://10.0.0.1"} {
		gateway, err := New(browserpolicy.Target{Origin: "https://website.fixture.test", AdditionalOrigins: []string{forbidden}})
		if err != nil {
			t.Fatal(err)
		}
		proxy := httptest.NewServer(gateway)
		address, _ := url.Parse(proxy.URL)
		client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(address)}, Timeout: time.Second}
		response, err := client.Get(forbidden)
		if err == nil {
			response.Body.Close()
			t.Fatal("approved metadata or private address escaped the public network guard")
		}
		client.CloseIdleConnections()
		proxy.Close()
		gateway.Close()
	}
}
