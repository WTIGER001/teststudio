package capture_test

import (
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/capture"
)

func TestPolicyBoundsAndKindRetention(t *testing.T) {
	l := capture.DefaultLimits()
	p := capture.DefaultPolicy()
	p.KindRetention = map[string]int{"video": 7, "trace": 14}
	p.FailureRetentionDays = 90
	if err := capture.Validate(p, l); err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC()
	for _, test := range []struct {
		kind, outcome string
		days          int
	}{{"video", "passed", 7}, {"trace", "passed", 14}, {"log", "passed", 30}, {"video", "failed", 90}, {"trace", "flaky", 90}} {
		if expiry := capture.Expiry(created, p, test.kind, test.outcome); !expiry.Equal(created.Add(time.Duration(test.days) * 24 * time.Hour)) {
			t.Fatalf("retention for %s/%s: %v", test.kind, test.outcome, expiry)
		}
	}
	for _, change := range []func(*capture.Policy){func(p *capture.Policy) { p.Video = "unknown" }, func(p *capture.Policy) { p.MaxBytes = l.RunBytes + 1 }, func(p *capture.Policy) { p.MaxFileBytes = l.FileBytes + 1 }, func(p *capture.Policy) { p.RetentionDays = -1 }, func(p *capture.Policy) { p.KindRetention = map[string]int{"trace": l.RetentionDays + 1} }} {
		invalid := capture.DefaultPolicy()
		change(&invalid)
		if err := capture.Validate(invalid, l); err == nil {
			t.Fatal("invalid capture policy accepted")
		}
	}
}

func TestResolvedPolicyDoesNotShareMutableMaps(t *testing.T) {
	p := capture.DefaultPolicy()
	p.KindRetention = map[string]int{"log": 5}
	s := capture.DefaultPolicy()
	s.KindRetention = map[string]int{"log": 9}
	s.Logs = false
	effective := capture.Resolve(&p, &s)
	s.KindRetention["log"] = 100
	if effective.KindRetention["log"] != 9 || effective.Logs {
		t.Fatal("resolved policy did not freeze the suite override")
	}
}
