package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

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
	flags := flag.NewFlagSet("teststudio-job", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	origin := flags.String("exchange-origin", "", "Private HTTPS worker exchange origin")
	operation := flags.String("operation", "", "Worker operation: empty for execution or discover")
	runner := flags.String("runner", "", "Immutable image runner: go or playwright")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 || (*runner != "go" && *runner != "playwright") || *operation != "" && *operation != "discover" {
		return errors.New("job supervisor requires a private HTTPS origin and a go or playwright runner")
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
	client, err := workerclient.New(*origin, execution, instance, token, nil)
	if err != nil {
		return errors.New("job exchange configuration is invalid")
	}
	defer client.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	supervisor := &jobsupervisor.Supervisor{Operation: *operation, Transport: client, Runner: &jobsupervisor.Process{Runner: *runner, Directory: "/suite", Output: "/output", PrivateValues: []string{token}}}
	outcome, err := supervisor.Run(ctx)
	if err != nil {
		return err
	}
	if *operation == "discover" {
		fmt.Println("Preparation discovery report accepted.")
	} else if outcome.EvidenceIncomplete {
		fmt.Println("Test report accepted; private evidence is incomplete.")
	} else {
		fmt.Println("Test report and private evidence accepted.")
	}
	return nil
}
