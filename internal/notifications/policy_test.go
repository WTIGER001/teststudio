package notifications

import (
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"testing"
	"time"
)

func TestNotificationScopesKeepSuiteTargetStageAndTriggerIndependent(t *testing.T) {
	run := runs.Run{Request: runs.Request{Suite: "api", Environment: "dev"}, Environment: &projects.Environment{Stage: "development", Protected: true}, Trigger: "schedule"}
	d := Definition{Suites: []string{"api"}, Environments: []string{"dev"}, Stages: []string{"development"}, Triggers: []string{"schedule"}}
	if !scopeMatches(d, run) {
		t.Fatal("matching development target was suppressed because it was protected")
	}
	d.Stages = []string{"production"}
	if scopeMatches(d, run) {
		t.Fatal("target protection became production-stage notification selection")
	}
	d.Stages = nil
	d.Triggers = []string{"manual"}
	if scopeMatches(d, run) {
		t.Fatal("scheduled run matched manual-only notification policy")
	}
	d.Triggers = nil
	d.Environments = []string{"none"}
	run.Request.Environment = ""
	run.Environment = nil
	if scopeMatches(d, run) {
		t.Fatal("literal environment identifier became an implicit targetless scope")
	}
	d.Environments = nil
	if !scopeMatches(d, run) {
		t.Fatal("unfiltered targetless suite did not match")
	}
}
func TestNotificationDefinitionsRejectUnboundedOrAmbiguousSelections(t *testing.T) {
	valid := normalize(Definition{ID: "alerts", Name: "Alerts", Kind: "webhook", Reference: "receiver"})
	if err := validate(valid); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Definition){func(d *Definition) { d.Outcomes = []string{} }, func(d *Definition) { d.Outcomes = []string{"timeout"} }, func(d *Definition) { d.Triggers = []string{"manual", "manual"} }, func(d *Definition) { d.MaxAttempts = 11 }, func(d *Definition) { d.Kind = "shell" }, func(d *Definition) { d.Reference = "https://not-a-reference" }} {
		d := valid
		change(&d)
		if err := validate(d); err == nil {
			t.Fatalf("invalid definition accepted: %+v", d)
		}
	}
	if delay("delivery", 1, 30*time.Second) < 30*time.Second || delay("delivery", 100, time.Hour*2) != time.Hour {
		t.Fatal("retry timing ignored bounded server delays")
	}
}
