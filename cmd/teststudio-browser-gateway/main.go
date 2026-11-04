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
	if _, err := browsergateway.New(target); err != nil {
		return err
	}
	listener, err := net.Listen("unix", "/output/.browser-egress.sock")
	if err != nil {
		return err
	}
	defer listener.Close()
	if err := os.Chmod("/output/.browser-egress.sock", 0600); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	return browsergateway.Serve(ctx, listener, target)
}
