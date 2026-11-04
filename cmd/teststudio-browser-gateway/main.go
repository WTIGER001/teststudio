package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wtiger001/teststudio/internal/apicredential"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if len(os.Args) == 2 && os.Args[1] == "--configure" {
		deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return sendCredential(deadline, credentialSocket, os.Stdin)
	}
	if len(os.Args) == 2 && os.Args[1] == "--ready" {
		connection, err := net.DialTimeout("unix", "/output/.browser-egress.sock", time.Second)
		if err != nil {
			return err
		}
		return connection.Close()
	}
	var target browserpolicy.Target
	if err := json.Unmarshal([]byte(os.Getenv("TESTSTUDIO_BROWSER_TARGET")), &target); err != nil {
		return fmt.Errorf("invalid browser gateway policy")
	}
	if err := browserpolicy.ValidateTarget(target); err != nil {
		return err
	}
	var credential *apicredential.Value
	if target.Authenticated {
		deadline, cancel := context.WithTimeout(ctx, 10*time.Second)
		value, err := receiveCredential(deadline, credentialSocket)
		cancel()
		if err != nil {
			return err
		}
		credential = value
	}
	g, err := browsergateway.NewAuthenticated(target, credential)
	if err != nil {
		return err
	}
	g.Close()
	listener, err := net.Listen("unix", "/output/.browser-egress.sock")
	if err != nil {
		return err
	}
	defer listener.Close()
	if err := os.Chmod("/output/.browser-egress.sock", 0600); err != nil {
		return err
	}
	return browsergateway.ServeAuthenticated(ctx, listener, target, credential)
}
