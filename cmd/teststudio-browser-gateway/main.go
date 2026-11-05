package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wtiger001/teststudio/internal/browsergateway"
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
	mode := ""
	if len(os.Args) == 2 {
		mode = os.Args[1]
	}
	if len(os.Args) > 2 || len(os.Args) == 2 && mode == "" {
		return errors.New("gateway accepts --configure, --ready, --ready-remote, --remote, --remote-exchange=HTTPS_ORIGIN, or --remote-ecs-exchange=HTTPS_ORIGIN")
	}
	switch mode {
	case "--configure":
		deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return sendCredential(deadline, credentialSocket, os.Stdin)
	case "--ready":
		connection, err := net.DialTimeout("unix", "/output/.browser-egress.sock", time.Second)
		if err != nil {
			return err
		}
		return connection.Close()
	case "--ready-remote":
		connection, err := net.DialTimeout("tcp", "127.0.0.1:8044", time.Second)
		if err != nil {
			return errors.New("remote gateway is unavailable")
		}
		return connection.Close()
	case "", "--remote":
	default:
		if strings.HasPrefix(mode, "--remote-ecs-exchange=") {
			configuration, err := fetchECSRemoteConfiguration(ctx, strings.TrimPrefix(mode, "--remote-ecs-exchange="), os.LookupEnv)
			if err != nil {
				return err
			}
			return serveRemote(ctx, configuration)
		}
		if strings.HasPrefix(mode, "--remote-exchange=") {
			configuration, err := fetchRemoteConfiguration(ctx, strings.TrimPrefix(mode, "--remote-exchange="), os.LookupEnv)
			if err != nil {
				return err
			}
			return serveRemote(ctx, configuration)
		}
		return errors.New("gateway accepts --configure, --ready, --ready-remote, --remote, --remote-exchange=HTTPS_ORIGIN, or --remote-ecs-exchange=HTTPS_ORIGIN")
	}
	deadline, cancel := context.WithTimeout(ctx, 10*time.Second)
	configuration, err := receiveCredential(deadline, credentialSocket)
	cancel()
	if err != nil {
		return err
	}
	if mode == "--remote" {
		return serveRemote(ctx, *configuration)
	}
	if configuration.Remote != nil {
		return errors.New("local gateway does not accept remote transport configuration")
	}
	target, credential := configuration.Target, configuration.Credential
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
