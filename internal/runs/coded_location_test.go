package runs_test

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/runs"
)

type locatedGoExecutor struct {
	goExecutor
	location string
}

func (e *locatedGoExecutor) Profile() string { return "go-" + e.location + "-isolated-v1" }

func TestCodedAdmissionMatchesTheConfiguredExecutionLocation(t *testing.T) {
	for _, scenario := range []struct {
		service, driver string
		allowed         bool
	}{{"local", "local", true}, {"kubernetes", "kubernetes", true}, {"kubernetes", "local", false}, {"local", "kubernetes", false}} {
		t.Run(scenario.service+"/"+scenario.driver, func(t *testing.T) {
			service, project, _ := setup(t, nil)
			configureGo(t, service, project)
			service.Location = scenario.service
			service.Go = &locatedGoExecutor{location: scenario.driver}
			run, err := service.Create(t.Context(), "admin", "project", runs.Request{Suite: "health"}, "location-check", "manual")
			if scenario.allowed {
				if err != nil || run.Request.Location != scenario.service {
					t.Fatalf("matching execution location was rejected: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "configured coded profile") {
				t.Fatalf("a driver for another location was admitted: %v", err)
			}
		})
	}
}
