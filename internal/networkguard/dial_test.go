package networkguard

import (
	"errors"
	"net"
	"testing"
)

func TestNetworkPoliciesSeparatePublicPrivateAndLoopbackAndAlwaysRejectMetadata(t *testing.T) {
	for _, address := range []string{"169.254.169.254", "169.254.170.2", "100.100.100.200", "168.63.129.16", "fd00:ec2::254", "::ffff:169.254.169.254", "fe80::1", "0.0.0.0", "::", "224.0.0.1"} {
		for _, scope := range []Scope{Public, Private, Loopback} {
			if PermittedAddress(net.ParseIP(address), scope) {
				t.Fatalf("%s allowed in %s", address, scope)
			}
		}
	}
	for _, address := range []string{"127.0.0.1", "::1", "::ffff:127.0.0.1"} {
		if PermittedAddress(net.ParseIP(address), Public) || !PermittedAddress(net.ParseIP(address), Loopback) {
			t.Fatalf("loopback policy: %s", address)
		}
	}
	if PermittedAddress(net.ParseIP("10.1.2.3"), Loopback) || !PermittedAddress(net.ParseIP("10.1.2.3"), Private) || !PermittedAddress(net.ParseIP("8.8.8.8"), Public) {
		t.Fatal("network scope mismatch")
	}
	if _, err := Dial(Public)(t.Context(), "tcp", "127.0.0.1:1"); !errors.Is(err, ErrScope) {
		t.Fatal("dial bypassed address validation")
	}
}
