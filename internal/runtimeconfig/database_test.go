package runtimeconfig_test

import (
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

func TestDatabaseEnvironmentKeepsDefaultsAndRejectsAmbiguousOrMissingSettings(t *testing.T) {
	secret := "postgres://user:private-database-canary@db.example.test/studio"
	lookup := func(name string) (string, bool) { return secret, name == "STUDIO_DATABASE" }
	value, err := runtimeconfig.DatabaseConnection("local.db", "", false, lookup)
	if err != nil || value != "local.db" {
		t.Fatal(value, err)
	}
	value, err = runtimeconfig.DatabaseConnection("local.db", "STUDIO_DATABASE", false, lookup)
	if err != nil || value != secret {
		t.Fatal("configured environment was not selected", err)
	}
	for _, item := range []struct {
		variable string
		explicit bool
	}{{"STUDIO_DATABASE", true}, {"STUDIO-MALFORMED", false}, {"MISSING", false}} {
		value, err := runtimeconfig.DatabaseConnection(secret, item.variable, item.explicit, lookup)
		if err == nil || value != "" || strings.Contains(err.Error(), "private-database-canary") {
			t.Fatal("invalid configuration was accepted or disclosed", err)
		}
	}
	if _, err := runtimeconfig.DatabaseConnection("local.db", "EMPTY", false, func(string) (string, bool) { return " ", true }); err == nil {
		t.Fatal("blank database environment accepted")
	}
}

func TestDatabaseEnvironmentErrorsDoNotRetainPrivateUpstreamDiagnostics(t *testing.T) {
	private := errors.New("private-database-canary")
	err := runtimeconfig.DatabaseError(private, "STUDIO_DATABASE")
	if err == nil || strings.Contains(err.Error(), private.Error()) || errors.Is(err, private) {
		t.Fatal("database environment failure disclosed its input", err)
	}
	if runtimeconfig.DatabaseError(nil, "STUDIO_DATABASE") != nil || runtimeconfig.DatabaseError(private, "") != private {
		t.Fatal("unrelated error behavior changed")
	}
}

func TestDatabaseFlagConflictUsesExplicitArgumentsInsteadOfTheDefaultValue(t *testing.T) {
	for _, arguments := range [][]string{{}, {"--database=local.db"}} {
		flags := flag.NewFlagSet("database", flag.ContinueOnError)
		flags.String("database", "local.db", "")
		if err := flags.Parse(arguments); err != nil {
			t.Fatal(err)
		}
		if runtimeconfig.ExplicitFlag(flags, "database") != (len(arguments) > 0) {
			t.Fatal("explicit default value was missed")
		}
	}
}

func TestHostedDatabaseGuardPreventsAnAccidentalLocalDatabaseFallback(t *testing.T) {
	for _, connection := range []string{"local.db", "/data/private-database-canary", ""} {
		err := runtimeconfig.RequirePostgres(connection, true)
		if err == nil || strings.Contains(err.Error(), connection) && connection != "" {
			t.Fatal("hosted database guard accepted or disclosed a local fallback", err)
		}
		if runtimeconfig.RequirePostgres(connection, false) != nil {
			t.Fatal("local database behavior changed")
		}
	}
	for _, connection := range []string{"postgres://user@database/studio", "postgresql://user@database/studio"} {
		if err := runtimeconfig.RequirePostgres(connection, true); err != nil {
			t.Fatal(err)
		}
	}
}
