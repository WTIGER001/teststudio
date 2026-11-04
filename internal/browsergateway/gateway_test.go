package browsergateway

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

func TestGatewayEnforcesOriginForHTTPAndTLSClients(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "HTTP", true: "HTTPS"}[secure], func(t *testing.T) {
			var reached atomic.Int32
			upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached.Add(1)
				if r.Header.Get("Proxy-Authorization") != "" {
					t.Error("proxy credential reached the target")
				}
				if r.URL.Path == "/redirect" {
					http.Redirect(w, r, "http://unapproved.test/", http.StatusFound)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				_, _ = io.WriteString(w, r.Method+" "+r.URL.RequestURI())
			}))
			if secure {
				upstream.StartTLS()
			} else {
				upstream.Start()
			}
			defer upstream.Close()
			u, _ := url.Parse(upstream.URL)
			u.Host = "fixture.test:" + u.Port()
			g, err := New(browserpolicy.Target{Origin: u.String(), AllowPrivate: true})
			if err != nil {
				t.Fatal(err)
			}
			defer g.Close()
			originalDial := g.dial
			actual, _ := url.Parse(upstream.URL)
			g.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
				if address != u.Host {
					t.Fatalf("unexpected upstream address %s", address)
				}
				return originalDial(ctx, network, actual.Host)
			}
			g.transport.DialContext = g.dial
			proxy := httptest.NewServer(g)
			defer proxy.Close()
			proxyURL, _ := url.Parse(proxy.URL)
			client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL), TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, Timeout: 5 * time.Second}
			defer client.CloseIdleConnections()
			request, _ := http.NewRequest("POST", u.String()+"/check?value=1", strings.NewReader("fixture"))
			if !secure {
				request.Header.Set("Proxy-Authorization", "should-not-forward")
			}
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != 200 || string(data) != "POST /check?value=1" {
				t.Fatalf("response %d: %s", response.StatusCode, data)
			}
			foreign := *u
			foreign.Host = "unapproved.test:" + u.Port()
			response, err = client.Get(foreign.String())
			if secure {
				if err == nil {
					response.Body.Close()
					t.Fatal("foreign TLS tunnel was allowed")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				response.Body.Close()
				if response.StatusCode != 403 {
					t.Fatal("foreign HTTP origin was allowed")
				}
				response, err = client.Get(u.String() + "/redirect")
				if err != nil {
					t.Fatal(err)
				}
				response.Body.Close()
				if response.StatusCode != 403 {
					t.Fatal("off-site redirect escaped the selected origin")
				}
			}
			expected := int32(1)
			if !secure {
				connection, err := net.DialTimeout("tcp", proxyURL.Host, time.Second)
				if err != nil {
					t.Fatal(err)
				}
				defer connection.Close()
				_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
				_, _ = fmt.Fprintf(connection, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", u.Host, u.Host)
				reader := bufio.NewReader(connection)
				status, err := reader.ReadString('\n')
				if err != nil || !strings.Contains(status, " 200 ") {
					t.Fatal(status, err)
				}
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						t.Fatal(err)
					}
					if line == "\r\n" {
						break
					}
				}
				_, _ = fmt.Fprintf(connection, "GET /tunnel HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", u.Host)
				tunneled, err := http.ReadResponse(reader, &http.Request{Method: "GET"})
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(tunneled.Body)
				tunneled.Body.Close()
				if err != nil || string(data) != "GET /tunnel" {
					t.Fatal(string(data), err)
				}
				expected = 3
			}
			if reached.Load() != expected {
				t.Fatal("a denied request reached the target")
			}
		})
	}
}
