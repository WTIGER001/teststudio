package relay

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

func validBackendAddress(raw string) bool {
	ip, err := netip.ParseAddr(raw)
	return err == nil && ip.Is4() && ip.IsPrivate() && ip.String() == raw
}

func newRelayTransport(backend *url.URL, address string, dial func(context.Context, string, string) (net.Conn, error)) *http.Transport {
	if dial == nil {
		dial = (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	port := backend.Port()
	if port == "" {
		port = "443"
	}
	return &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, network, requested string) (net.Conn, error) {
		if !strings.EqualFold(requested, net.JoinHostPort(backend.Hostname(), port)) {
			return nil, errConfiguration
		}
		if address != "" {
			requested = net.JoinHostPort(address, port)
		}
		return dial(ctx, network, requested)
	}, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: backend.Hostname()}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 15 * time.Second, IdleConnTimeout: 30 * time.Second, MaxConnsPerHost: 8, MaxIdleConnsPerHost: 8, MaxIdleConns: 8, DisableCompression: true}
}
