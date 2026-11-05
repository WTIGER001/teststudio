package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"os"
	"time"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func fetchRemoteConfiguration(ctx context.Context, origin string, lookup func(string) (string, bool)) (browsergateway.Configuration, error) {
	execution, _ := lookup("TESTSTUDIO_GATEWAY_EXECUTION")
	token, _ := lookup("TESTSTUDIO_GATEWAY_TOKEN")
	instance, _ := lookup("TESTSTUDIO_GATEWAY_INSTANCE")
	client, err := workerclient.New(origin, execution, instance, token, nil)
	if err != nil {
		return browsergateway.Configuration{}, errors.New("remote gateway initialization identity is invalid")
	}
	defer client.Close()
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for bounded.Err() == nil {
		configuration, err := client.GatewayInput(bounded)
		if err == nil {
			gateway, err := browsergateway.NewRemote(configuration)
			if err != nil {
				return browsergateway.Configuration{}, errors.New("remote gateway initialization configuration is invalid")
			}
			gateway.Close()
			return configuration, nil
		}
		if !errors.Is(err, workerprotocol.ErrUnavailable) && !errors.Is(err, workerprotocol.ErrConflict) {
			break
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-bounded.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	return browsergateway.Configuration{}, errors.New("remote gateway private initialization is unavailable")
}

func serveRemote(ctx context.Context, configuration browsergateway.Configuration) error {
	gateway, err := browsergateway.NewRemote(configuration)
	if err != nil {
		return err
	}
	gateway.Close()
	certificate, err := remoteCertificate(os.LookupEnv)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", ":8044")
	if err != nil {
		return errors.New("remote gateway listener is unavailable")
	}
	defer listener.Close()
	return browsergateway.ServeRemote(ctx, listener, configuration, &tls.Config{Certificates: []tls.Certificate{certificate}})
}
