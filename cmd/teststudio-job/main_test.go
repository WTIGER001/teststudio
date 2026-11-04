package main

import (
	"strings"
	"testing"
)

func TestJobConfigurationUsesOnlyEnvironmentTicketsAndExcludesValuesFromErrors(t *testing.T) {
	canary := "private-job-ticket-canary"
	for _, test := range []struct {
		args        []string
		environment map[string]string
	}{
		{[]string{"--runner=go", "--token=" + canary}, nil},
		{[]string{"--runner=unknown-" + canary}, nil},
		{[]string{"--runner=go", "--exchange-origin=https://jobs.example.test"}, map[string]string{"TESTSTUDIO_JOB_EXECUTION": "execution", "TESTSTUDIO_JOB_TOKEN": canary}},
		{[]string{"--runner=playwright", "--exchange-origin=http://jobs.example.test"}, map[string]string{"TESTSTUDIO_JOB_EXECUTION": "execution", "TESTSTUDIO_JOB_TOKEN": strings.Repeat("x", 43)}},
	} {
		err := run(test.args, func(name string) (string, bool) { value, found := test.environment[name]; return value, found })
		if err == nil || strings.Contains(err.Error(), canary) {
			t.Fatal("invalid job configuration was accepted or exposed credentials", err)
		}
	}
}
