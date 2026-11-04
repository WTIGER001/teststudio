package runtimeconfig

import (
	"errors"
	"flag"
	"regexp"
	"strings"
)

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ExplicitFlag(flags *flag.FlagSet, name string) bool {
	found := false
	flags.Visit(func(value *flag.Flag) {
		if value.Name == name {
			found = true
		}
	})
	return found
}

func DatabaseConnection(direct, variable string, directProvided bool, lookup func(string) (string, bool)) (string, error) {
	if variable == "" {
		return direct, nil
	}
	if directProvided {
		return "", errors.New("choose either --database or --database-env")
	}
	if !environmentName.MatchString(variable) {
		return "", errors.New("database-env must name an environment variable")
	}
	value, found := lookup(variable)
	if !found || strings.TrimSpace(value) == "" {
		return "", errors.New("the database environment variable is missing or empty")
	}
	return value, nil
}

func DatabaseError(err error, variable string) error {
	if err == nil || variable == "" {
		return err
	}
	return errors.New("database initialization failed; check the configured database connection and service access")
}

func RequirePostgres(connection string, required bool) error {
	if required && !strings.HasPrefix(connection, "postgres://") && !strings.HasPrefix(connection, "postgresql://") {
		return errors.New("this deployment requires a PostgreSQL database connection")
	}
	return nil
}
