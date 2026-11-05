package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	workerclient "github.com/wtiger001/teststudio/adapters/workerexchange/client"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/jobsupervisor"
)

func main() {
	if err := run(os.Args[1:], os.LookupEnv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(arguments []string, lookup func(string) (string, bool)) error {
	options, err := parseOptions(arguments, time.Now())
	if err != nil {
		return err
	}
	if options.sandbox || !options.idleUntil.IsZero() {
		if err := sandboxIdentity(); err != nil {
			return err
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if !options.idleUntil.IsZero() {
		idleSandbox(ctx, options.idleUntil)
		return nil
	}
	execution, found := lookup("TESTSTUDIO_JOB_EXECUTION")
	if !found || !domain.ValidID(execution) {
		return errors.New("job execution identity is missing or invalid")
	}
	token, found := lookup("TESTSTUDIO_JOB_TOKEN")
	if !found || len(token) != 43 {
		return errors.New("job ticket is missing or invalid")
	}
	instance, found := lookup("TESTSTUDIO_JOB_INSTANCE")
	if !found {
		instance = domain.NewID()
	}
	if !domain.ValidID(instance) {
		return errors.New("job instance identity is invalid")
	}
	client, err := workerclient.New(options.origin, execution, instance, token, nil)
	if err != nil {
		return errors.New("job exchange configuration is invalid")
	}
	defer client.Close()
	supervisor := &jobsupervisor.Supervisor{GatewaySocket: "/output/.browser-egress.sock", Operation: options.operation, Transport: client, Runner: &jobsupervisor.Process{Runner: options.runner, Directory: "/suite", Output: "/output", PrivateValues: []string{token}}}
	outcome, err := supervisor.Run(ctx)
	if err != nil {
		return err
	}
	if options.operation == "discover" {
		fmt.Println("Preparation discovery report accepted.")
	} else if outcome.EvidenceIncomplete {
		fmt.Println("Test report accepted; private evidence is incomplete.")
	} else {
		fmt.Println("Test report and private evidence accepted.")
	}
	return nil
}
