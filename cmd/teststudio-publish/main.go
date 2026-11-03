package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
)

type savedReference struct {
	Schema  string                        `json:"schemaVersion"`
	Runner  string                        `json:"runner"`
	Request preparation.InspectionRequest `json:"request"`
}

type inspectionFile struct {
	Reference  execution.Reference    `json:"reference"`
	Schema     string                 `json:"schemaVersion"`
	Status     string                 `json:"status"`
	Project    string                 `json:"projectId"`
	Suite      string                 `json:"suiteId"`
	Inspection preparation.Inspection `json:"inspection"`
	Error      string                 `json:"error,omitempty"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, writer io.Writer) error {
	if len(args) == 0 {
		return errors.New("use teststudio-publish inspect or cleanup; server publication is not available yet")
	}
	switch args[0] {
	case "inspect":
		return inspect(ctx, args[1:], writer)
	case "cleanup":
		return cleanup(ctx, args[1:], writer)
	default:
		return errors.New("unknown publisher operation; use inspect or cleanup")
	}
}

func inspect(ctx context.Context, args []string, writer io.Writer) error {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(writer)
	image := flags.String("image", "", "Locally available immutable image digest")
	runner := flags.String("runner", "", "go or playwright")
	project := flags.String("project", "", "Destination project identifier")
	suite := flags.String("suite", "", "Destination suite identifier")
	output := flags.String("output", "", "New private inspection JSON file")
	state := flags.String("state-directory", "", "Private preparation reference directory; defaults to the user cache")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *image == "" || !domain.ValidID(*project) || !domain.ValidID(*suite) || (*runner != "go" && *runner != "playwright") || *output == "" {
		return errors.New("inspect requires image, runner, project, suite, and a new output path")
	}
	if _, err := os.Lstat(*output); !os.IsNotExist(err) {
		return errors.New("inspection output must be a new file")
	}
	if *state == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		*state = filepath.Join(cache, "TestStudio", "preparation")
	}
	driver := &localcontainer.Executor{Runner: *runner, Images: []string{*image}}
	id := domain.NewID()
	ref := driver.Reference(execution.Reference{ID: id, Run: id, Installation: "preparation-" + domain.NewID(), Project: *project, Test: "catalog", Attempt: 1, Worker: "publisher-" + domain.NewID()})
	request := preparation.InspectionRequest{Image: *image, Suite: *suite, Reference: ref}
	saved := savedReference{Schema: "teststudio.preparation-reference/v1", Runner: *runner, Request: request}
	stateFile := filepath.Join(*state, id+".json")
	if err := writeNew(stateFile, saved); err != nil {
		return fmt.Errorf("save preparation reference: %w", err)
	}
	fmt.Fprintln(writer, "Inspecting the immutable image in an isolated local worker.")
	observed, inspectErr := driver.Inspect(ctx, request)
	cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	cleanupErr := driver.CleanupInspection(cleanupContext, request)
	cancel()
	if cleanupErr == nil {
		if err := os.Remove(stateFile); err != nil {
			cleanupErr = errors.New("cleaned preparation reference could not be removed")
		}
	}
	if cleanupErr != nil {
		fmt.Fprintln(writer, "Retained cleanup reference:", stateFile)
	}
	resultErr := errors.Join(inspectErr, cleanupErr)
	result := inspectionFile{Reference: ref, Schema: "teststudio.inspection/v1", Status: "inspected", Project: *project, Suite: *suite, Inspection: observed}
	if resultErr != nil {
		result.Status, result.Error = "failed", resultErr.Error()
	}
	if err := writeNew(*output, result); err != nil {
		return errors.Join(resultErr, fmt.Errorf("save private inspection result: %w", err))
	}
	if resultErr != nil {
		return fmt.Errorf("artifact inspection failed; diagnostics: %s", *output)
	}
	fmt.Fprintf(writer, "Inspected %d catalog entries. Private result: %s\n", len(observed.Discovery.Tests), *output)
	return nil
}

func cleanup(ctx context.Context, args []string, writer io.Writer) error {
	flags := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	flags.SetOutput(writer)
	filename := flags.String("state-file", "", "Exact private preparation reference file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *filename == "" {
		return errors.New("cleanup requires one state-file")
	}
	file, err := os.Open(*filename)
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, definitions.ContentLimit+1))
	file.Close()
	if readErr != nil {
		return readErr
	}
	var state savedReference
	if err := definitions.Decode("json", string(data), &state); err != nil {
		return err
	}
	if state.Schema != "teststudio.preparation-reference/v1" || (state.Runner != "go" && state.Runner != "playwright") {
		return errors.New("unsupported preparation reference")
	}
	driver := &localcontainer.Executor{Runner: state.Runner}
	cleanupContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := driver.CleanupInspection(cleanupContext, state.Request); err != nil {
		return err
	}
	if err := os.Remove(*filename); err != nil {
		return err
	}
	fmt.Fprintln(writer, "Owned preparation container and volume are absent.")
	return nil
}

func writeNew(filename string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(filename)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".teststudio-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Link(temporary, filename)
}
