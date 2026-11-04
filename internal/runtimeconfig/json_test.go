package runtimeconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

func TestConfigurationReferencesAreBoundedStrictAndDoNotEchoValues(t *testing.T) {
	for _, fixture := range []struct {
		name, file, variable, value string
		exists, valid               bool
	}{
		{name: "unconfigured", valid: true},
		{name: "valid", variable: "SETTINGS", value: `{"name":"ready"}`, exists: true, valid: true},
		{name: "conflict", file: "unused", variable: "SETTINGS", value: `{"name":"must-not-echo"}`, exists: true},
		{name: "invalid name", variable: "not-a-variable"},
		{name: "missing", variable: "SETTINGS"},
		{name: "blank", variable: "SETTINGS", value: " \n", exists: true},
		{name: "large", variable: "SETTINGS", value: strings.Repeat("must-not-echo", 6000), exists: true},
		{name: "unknown", variable: "SETTINGS", value: `{"must-not-echo":true}`, exists: true},
		{name: "trailing", variable: "SETTINGS", value: `{"name":"must-not-echo"} {}`, exists: true},
		{name: "malformed", variable: "SETTINGS", value: `must-not-echo`, exists: true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			var destination struct {
				Name string `json:"name"`
			}
			configured, err := runtimeconfig.LoadJSON(fixture.file, fixture.variable, "test", func(string) (string, bool) { return fixture.value, fixture.exists }, &destination)
			if (err == nil) != fixture.valid || (err != nil && strings.Contains(err.Error(), "must-not-echo")) {
				t.Fatal("incorrect configuration outcome", err)
			}
			if fixture.name == "valid" && (!configured || destination.Name != "ready") {
				t.Fatal("environment configuration was not loaded")
			}
			if fixture.name == "unconfigured" && configured {
				t.Fatal("missing configuration was selected")
			}
		})
	}
}

func TestFileConfigurationUsesTheSameStrictReadLimit(t *testing.T) {
	file := filepath.Join(t.TempDir(), "settings.json")
	for _, data := range []string{`{"name":"ready"}`, strings.Repeat("x", (64<<10)+1), `{"name":"must-not-echo"} []`} {
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		var destination struct {
			Name string `json:"name"`
		}
		_, err := runtimeconfig.LoadJSON(file, "", "test", os.LookupEnv, &destination)
		if (err == nil) != (data == `{"name":"ready"}`) {
			t.Fatal("incorrect file configuration outcome", err)
		}
	}
}
