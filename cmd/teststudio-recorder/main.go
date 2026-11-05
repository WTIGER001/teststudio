package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wtiger001/teststudio/adapters/recording/companion"
	"github.com/wtiger001/teststudio/adapters/recording/localplaywright"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, input io.Reader, output io.Writer) (err error) {
	flags := flag.NewFlagSet("teststudio-recorder", flag.ContinueOnError)
	flags.SetOutput(output)
	server := flags.String("server", "", "TestStudio HTTPS origin; loopback HTTP is allowed for local verification")
	resume := flags.String("resume", "", "Recover an interrupted session without opening another browser")
	directory := flags.String("directory", "", "Private desktop connection and captured-source directory")
	cli := flags.String("playwright-cli", "runners/playwright/node_modules/playwright/cli.js", "Trusted installed Playwright CLI path")
	helper := flags.String("helper", "scripts/record-browser.mjs", "Trusted recording helper path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("connection codes must be pasted at the prompt, not passed as command arguments")
	}
	if *directory == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return errors.New("choose a private desktop recording directory")
		}
		*directory = filepath.Join(cache, "TestStudio", "recordings")
	}
	journal, err := companion.NewJournal(*directory)
	if err != nil {
		return err
	}
	lock, err := journal.Lock()
	if err != nil {
		return err
	}
	defer lock.Close()
	var state companion.State
	if *resume != "" {
		state, err = journal.Read(*resume)
		if err != nil {
			return err
		}
		if *server != "" {
			origin, err := companion.Origin(*server)
			if err != nil || origin != state.Server {
				return errors.New("resume must use the session's original TestStudio server")
			}
		}
	} else {
		if _, err := companion.Origin(*server); err != nil {
			return err
		}
		fmt.Fprintln(output, "Paste the connection code from TestStudio, then press Enter:")
		reader := bufio.NewReader(io.LimitReader(input, 1025))
		code, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF || len(code) > 1024 {
			return errors.New("the connection code could not be read")
		}
		state, err = companion.NewState(*server, code)
		if err != nil {
			return err
		}
		if err := journal.Create(state); err != nil {
			return err
		}
	}
	remote, err := companion.NewClient(state.Server)
	if err != nil {
		return err
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return errors.New("install Node.js and the trusted TestStudio recorder dependencies first")
	}
	driver, err := localplaywright.New(ctx, node, *cli, *helper, filepath.Join(*directory, "captures"))
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := driver.Close(); closeErr != nil {
			err = closeErr
		}
	}()
	fmt.Fprintln(output, "Recording session:", state.ID)
	fmt.Fprintln(output, "Use TestStudio to stop and keep the recording. Ctrl+C stops the owned browser and retains local source.")
	runner := &companion.Runner{Remote: remote, Driver: driver, Journal: journal, RecoveryDelay: 12 * time.Second}
	if err := runner.Run(ctx, &state); err != nil {
		fmt.Fprintln(output, "Local connection and captured source are retained. To recover, run this command again with --resume", state.ID)
		return err
	}
	fmt.Fprintln(output, "The recording has ended. Return to TestStudio to review captured steps.")
	return nil
}
