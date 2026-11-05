package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreparationDatabaseHostRejectsDriftBeforeSettingsOrState(t *testing.T) {
	for _, mode := range []string{"sqlite", "hostname", "tls", "routing", "environment"} {
		t.Run(mode, func(t *testing.T) {
			originalFlags, originalArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
			flag.CommandLine = flag.NewFlagSet("owned-preparation-database-host-check", flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			database := filepath.Join(t.TempDir(), "must-not-exist.db")
			connection := "postgres://studio:private-canary@database.example.test/studio?sslmode=verify-full"
			switch mode {
			case "sqlite":
				connection = database
			case "hostname":
				connection = strings.Replace(connection, "database.example.test", "foreign.example.test", 1)
			case "tls":
				connection = strings.Replace(connection, "verify-full", "prefer", 1)
			case "routing":
				connection += "&service=private-canary"
			case "environment":
				t.Setenv("PGSERVICEFILE", "private-canary")
			}
			t.Setenv("TESTSTUDIO_PREPARATION_DATABASE_CHECK", connection)
			os.Args = []string{"teststudio-prepare", "--discovery-only", "--execution-location=kubernetes", "--kubernetes-workers-config=must-not-read", "--database-env=TESTSTUDIO_PREPARATION_DATABASE_CHECK", "--database-host=database.example.test"}
			err := run()
			if err == nil || !strings.Contains(err.Error(), "database-host requires") || strings.Contains(err.Error(), "private-canary") {
				t.Fatal("database host guard did not reject before settings or disclosed input", err)
			}
			if _, err := os.Stat(database); !os.IsNotExist(err) {
				t.Fatal("rejected preparation created state", err)
			}
		})
	}
}
