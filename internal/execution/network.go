package execution

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/netip"
	"strings"
	"time"
)

func permittedAddress(ip net.IP, private bool) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsUnspecified() || addr.IsMulticast() {
		return false
	}
	if strings.HasPrefix(addr.String(), "169.254.") || addr.String() == "100.100.100.200" || addr.String() == "168.63.129.16" || addr.String() == "fd00:ec2::254" {
		return false
	}
	return private || !(addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast())
}

func guardedDial(private bool) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(strings.ToLower(host), ".")
		if name == "metadata.google.internal" || name == "metadata.azure.internal" {
			return nil, errors.New("metadata targets are forbidden")
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("target has no resolved addresses")
		}
		for _, ip := range ips {
			if !permittedAddress(ip.IP, private) {
				return nil, errors.New("target resolves outside the permitted network scope")
			}
		}
		dialer := net.Dialer{Timeout: 5 * time.Second}
		var last error
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			last = err
		}
		return nil, last
	}
}

func failureKind(err error) string {
	kind := "connection"
	var dns *net.DNSError
	var verification *tls.CertificateVerificationError
	var network net.Error
	if errors.As(err, &dns) {
		kind = "dns"
	}
	if errors.As(err, &verification) {
		kind = "tls"
	}
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &network) && network.Timeout()) {
		kind = "timeout"
	}
	if errors.Is(err, context.Canceled) {
		kind = "cancelled"
	}
	return kind
}
