package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"github.com/wtiger001/teststudio/internal/browsergateway"
)

func serveRemote(ctx context.Context, configuration browsergateway.Configuration) error {
	gateway, err := browsergateway.NewRemote(configuration)
	if err != nil {
		return err
	}
	gateway.Close()
	certificate, err := tls.LoadX509KeyPair("/tls/tls.crt", "/tls/tls.key")
	if err != nil {
		return errors.New("remote gateway TLS certificate is unavailable")
	}
	listener, err := net.Listen("tcp", ":8044")
	if err != nil {
		return errors.New("remote gateway listener is unavailable")
	}
	defer listener.Close()
	return browsergateway.ServeRemote(ctx, listener, configuration, &tls.Config{Certificates: []tls.Certificate{certificate}})
}
