package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wtiger001/teststudio/adapters/workerexchange/relay"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string) error {
	flags := flag.NewFlagSet("teststudio-exchange", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	address := flags.String("addr", "127.0.0.1:8043", "Private relay listen address behind trusted HTTPS ingress")
	public := flags.String("public-origin", "", "Exact worker-facing HTTPS origin")
	backend := flags.String("backend-origin", "", "Fixed trusted HTTPS worker-listener origin")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 {
		return errors.New("invalid worker relay options")
	}
	handler, err := relay.New(*public, *backend, nil)
	if err != nil {
		return err
	}
	defer handler.Close()
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return errors.New("worker relay could not bind its listen address")
	}
	defer listener.Close()
	return serve(ctx, listener, handler)
}

func serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 45 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 16 << 10, ErrorLog: log.New(io.Discard, "", 0)}
	defer server.Close()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errors.New("worker relay listener stopped")
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			server.Close()
			<-done
			return errors.New("worker relay shutdown did not complete")
		}
		<-done
		return nil
	}
}
