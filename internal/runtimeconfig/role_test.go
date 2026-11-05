package runtimeconfig_test

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

func TestServiceRolesKeepWorkerListenerSeparateFromApplicationRoles(t *testing.T) {
	for name, want := range map[string]runtimeconfig.ServiceRole{
		"all": {API: true, Artifacts: true}, "api": {API: true}, "artifacts": {Artifacts: true}, "worker": {Worker: true}, "controller": {Controller: true},
	} {
		got, err := runtimeconfig.ParseServiceRole(name)
		if err != nil || got != want {
			t.Fatal(name, got, err)
		}
	}
	for _, name := range []string{"", "workers", "Worker", "api,worker"} {
		if _, err := runtimeconfig.ParseServiceRole(name); err == nil {
			t.Fatal("invalid service role accepted")
		}
	}
}
