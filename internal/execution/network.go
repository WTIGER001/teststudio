package execution

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
)

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
