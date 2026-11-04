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
	"syscall"

	permissionpg "github.com/wtiger001/teststudio/adapters/permissions/postgres"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/transfer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "help") {
		_, err := fmt.Fprintln(output, "teststudio-data export --database SQLITE_PATH --archive PRIVATE_FILE\nteststudio-data import --database-env CONNECTION_VARIABLE --archive PRIVATE_FILE\n\nStop APIs and workers for cutover. Export includes private metadata and credentials, not artifact bytes. Import requires an empty destination. Existing archives are never overwritten.")
		return err
	}
	if len(args) == 0 || (args[0] != "export" && args[0] != "import") {
		return errors.New("use teststudio-data export or import with a database and archive path")
	}
	mode := args[0]
	flags := flag.NewFlagSet("teststudio-data "+mode, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	database := flags.String("database", "", "SQLite path or PostgreSQL URL")
	environment := flags.String("database-env", "", "Environment variable containing the database connection")
	archive := flags.String("archive", "", "Private installation archive path")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return run(ctx, []string{"--help"}, output)
	} else if err != nil {
		return errors.New("unsupported transfer flags")
	}
	if flags.NArg() != 0 || *archive == "" || (*database == "") == (*environment == "") {
		return errors.New("provide --archive and exactly one of --database or --database-env")
	}
	connection := *database
	if *environment != "" {
		connection = os.Getenv(*environment)
		if connection == "" {
			return errors.New("the database environment variable is empty")
		}
	}
	if mode == "export" {
		source, err := transfer.OpenSource(ctx, connection)
		if err != nil {
			return err
		}
		defer source.Close()
		stats, err := transfer.ExportFile(ctx, source, *archive)
		if errors.Is(err, transfer.ErrBusy) {
			return err
		}
		if err != nil {
			return errors.New("installation export failed; check schema, database access, and the unused archive path")
		}
		return json.NewEncoder(output).Encode(stats)
	}
	file, err := os.Open(*archive)
	if err != nil {
		return errors.New("installation archive cannot be read")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return errors.New("use a regular private archive file with permissions 0600")
	}
	destination, err := storage.Open(ctx, connection)
	if err != nil {
		return errors.New("destination database cannot be opened")
	}
	defer destination.Close()
	if destination.Dialect == "postgres" {
		native, err := permissionpg.Open(ctx, connection)
		if err != nil {
			return errors.New("destination permission schema cannot be initialized")
		}
		defer native.Close()
	}
	stats, err := transfer.Import(ctx, destination, file)
	if errors.Is(err, transfer.ErrNotEmpty) {
		return err
	}
	if err != nil {
		return errors.New("installation import failed; destination data was not committed")
	}
	return json.NewEncoder(output).Encode(stats)
}
