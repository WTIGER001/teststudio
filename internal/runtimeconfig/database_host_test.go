package runtimeconfig_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

func TestDatabaseHostBindsSingleDestinationAndVerifiedTLSBeforeDriverUse(t *testing.T) {
	host := "database.example.test"
	connection := "postgresql://" + url.UserPassword("studio@installation", "private-canary: /?&%").String() + "@" + host + ":5432/studio?sslmode=verify-full&sslrootcert=system&connect_timeout=5&application_name=teststudio"
	if err := runtimeconfig.ValidateDatabaseHost(connection, host, []string{"TESTSTUDIO_DATABASE=private-canary", "PGHOST="}); err != nil {
		t.Fatal("complete bound settings rejected", err)
	}
	config, err := pgx.ParseConfig(connection)
	if err != nil || config.Host != host || config.Port != 5432 || config.TLSConfig == nil || config.TLSConfig.InsecureSkipVerify || config.TLSConfig.ServerName != host || len(config.Fallbacks) != 0 || config.Password != "private-canary: /?&%" {
		t.Fatal("admitted connection does not retain its destination and TLS boundary")
	}
	if runtimeconfig.ValidateDatabaseHost("local.db", "", []string{"PGHOST=another-host"}) != nil {
		t.Fatal("optional guard changed existing local behavior")
	}
	for _, raw := range []string{"db", "localhost", "127.0.0.1", "DATABASE.example.test", "database.example.test.", "db..example", "-db.example", "db_.example", "db.example:5432", "db.example,other.example"} {
		if runtimeconfig.ValidDatabaseHost(raw) {
			t.Fatal("noncanonical hostname accepted")
		}
	}
}

func TestDatabaseHostRejectsFallbackRoutingAndAmbientSettingsWithoutSecretDisclosure(t *testing.T) {
	host := "database.example.test"
	base := "postgres://studio:private-canary@" + host + "/studio?sslmode=verify-full"
	for _, connection := range []string{
		"local.db", strings.Replace(base, host, "foreign.example.test", 1), strings.Replace(base, host, host+":6543", 1), strings.Replace(base, host, host+",foreign.example.test", 1), strings.Replace(base, "/studio?", "/studio/other?", 1), strings.Replace(base, ":private-canary", "", 1), strings.Replace(base, "studio:private-canary", "studio:", 1), strings.Replace(base, "/studio?", "/?", 1), strings.Replace(base, "verify-full", "require", 1), strings.Replace(base, "verify-full", "prefer", 1), strings.Replace(base, "verify-full", "disable", 1), base + "&host=foreign.example.test", base + "&hostaddr=127.0.0.1", base + "&service=private-canary", base + "&servicefile=/private-canary", base + "&sslmode=disable", base + "&sslcert=/private-canary", base + "&options=private-canary", base + "&sslrootcert=relative", base + "&connect_timeout=0", base + "&connect_timeout=31", base + "&connect_timeout=01", base + "&bad=%XX", base + "#private-canary", base + "&application_name=%00private-canary", base + "&sslrootcert=", base + "&sslrootcert=/a/../private-canary",
	} {
		err := runtimeconfig.ValidateDatabaseHost(connection, host, nil)
		if err == nil || strings.Contains(err.Error(), "private-canary") {
			t.Fatal("unsafe connection admitted or diagnostics disclosed", err)
		}
	}
	for _, name := range []string{"PGHOST", "PGPORT", "PGSERVICE", "PGSERVICEFILE", "PGPASSWORD", "PGSSLMODE", "PGSSLROOTCERT", "PGOPTIONS", "PGTARGETSESSIONATTRS", "PG_FUTURE_ROUTING_SETTING"} {
		err := runtimeconfig.ValidateDatabaseHost(base, host, []string{name + "=private-canary"})
		if err == nil || strings.Contains(err.Error(), "private-canary") {
			t.Fatal("ambient driver override accepted or disclosed", err)
		}
	}
}
