package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControllerDatabaseHostRejectsDriftBeforeStorageOrListeners(t *testing.T) {
	for _, mode := range []string{"sqlite", "hostname", "tls", "routing", "environment"} {
		t.Run(mode, func(t *testing.T) {
			originalFlags, originalArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
			flag.CommandLine = flag.NewFlagSet("owned-database-host-check", flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			root := t.TempDir()
			database := filepath.Join(root, "must-not-exist.db")
			connection := "postgres://studio:private-canary@database.example.test/studio?sslmode=verify-full"
			switch mode {
			case "sqlite":
				connection = database
			case "hostname":
				connection = strings.Replace(connection, "database.example.test", "foreign.example.test", 1)
			case "tls":
				connection = strings.Replace(connection, "verify-full", "require", 1)
			case "routing":
				connection += "&host=foreign.example.test"
			case "environment":
				t.Setenv("PGHOST", "private-canary")
			}
			t.Setenv("TESTSTUDIO_DATABASE_HOST_CHECK", connection)
			os.Args = []string{"teststudio", "--serve-role=controller", "--addr=127.0.0.1:0", "--database-env=TESTSTUDIO_DATABASE_HOST_CHECK", "--database-host=database.example.test", "--artifact-directory=" + filepath.Join(root, "must-not-exist")}
			err := run()
			if err == nil || !strings.Contains(err.Error(), "database-host requires") || strings.Contains(err.Error(), "private-canary") {
				t.Fatal("database host guard did not reject before startup or disclosed input", err)
			}
			for _, path := range []string{database, filepath.Join(root, "must-not-exist")} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("rejected configuration created state", path, err)
				}
			}
		})
	}
}
