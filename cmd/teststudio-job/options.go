package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"
)

type jobOptions struct {
	origin    string
	runner    string
	operation string
	idleUntil time.Time
	sandbox   bool
}

func parseOptions(arguments []string, now time.Time) (jobOptions, error) {
	flags := flag.NewFlagSet("teststudio-job", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options jobOptions
	var idle string
	flags.StringVar(&options.origin, "exchange-origin", "", "Private HTTPS worker exchange origin")
	flags.StringVar(&options.operation, "operation", "", "Worker operation: empty for execution or discover")
	flags.StringVar(&options.runner, "runner", "", "Immutable image runner: go or playwright")
	flags.StringVar(&idle, "sandbox-idle-until", "", "Absolute deadline for an idle sandbox")
	flags.BoolVar(&options.sandbox, "sandbox-command", false, "Require the isolated sandbox execution identity")
	err := errors.New("job supervisor requires a bounded sandbox mode or a private HTTPS origin and a go or playwright runner")
	if flags.Parse(arguments) != nil || flags.NArg() != 0 {
		return jobOptions{}, err
	}
	if idle != "" {
		at, parseErr := time.Parse(time.RFC3339Nano, idle)
		if parseErr != nil || !at.After(now) || at.Sub(now) > 2*time.Hour || at.UTC().Format(time.RFC3339Nano) != idle || options.runner != "" || options.origin != "" || options.operation != "" || options.sandbox {
			return jobOptions{}, err
		}
		options.idleUntil = at
		return options, nil
	}
	if options.runner != "go" && options.runner != "playwright" || options.operation != "" && options.operation != "discover" {
		return jobOptions{}, err
	}
	return options, nil
}

func idleSandbox(ctx context.Context, at time.Time) {
	wall, cancel := context.WithDeadline(ctx, at)
	defer cancel()
	<-wall.Done()
}
