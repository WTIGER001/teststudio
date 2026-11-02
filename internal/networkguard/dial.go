package networkguard

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"time"
)

type Scope string

const (
	Public   Scope = "public"
	Private  Scope = "private"
	Loopback Scope = "loopback"
)

var ErrScope = errors.New("target resolves outside the permitted network scope")

func PermittedAddress(ip net.IP, scope Scope) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsUnspecified() || addr.IsMulticast() || addr.IsLinkLocalUnicast() {
		return false
	}
	if strings.HasPrefix(addr.String(), "169.254.") || addr.String() == "100.100.100.200" || addr.String() == "168.63.129.16" || addr.String() == "fd00:ec2::254" {
		return false
	}
	switch scope {
	case Loopback:
		return addr.IsLoopback()
	case Private:
		return true
	case Public:
		return !(addr.IsPrivate() || addr.IsLoopback())
	default:
		return false
	}
}
func Dial(scope Scope) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(strings.ToLower(host), ".")
		if name == "metadata.google.internal" || name == "metadata.azure.internal" {
			return nil, ErrScope
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("target has no resolved addresses")
		}
		for _, ip := range ips {
			if !PermittedAddress(ip.IP, scope) {
				return nil, ErrScope
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
