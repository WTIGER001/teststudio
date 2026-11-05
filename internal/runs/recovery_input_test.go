package runs

import (
	"context"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/projects"
)

func TestRecoveryInputRestoresFrozenEnvironmentAndCredentialReference(t *testing.T) {
	test := catalog.Test{ID: "API-1", Runner: "go", Type: "api", Selector: "./checks::TestStatus", RequiredBindings: []string{"reader"}, TimeoutSeconds: 5}
	policy := &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "reader", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/status", ReadOnly: true}}, MaxRequests: 2, MaxResponseBytes: 4096}
	run := Run{ID: "run", Tests: []catalog.Test{test}, Environment: &projects.Environment{Application: "app", Stage: "staging", Endpoints: map[string]string{"api": "https://application.test"}, Credentials: map[string]string{"reader": "saved-reader"}}, Release: catalog.Release{Draft: catalog.Draft{Artifact: catalog.Artifact{Browser: &browserpolicy.Spec{Endpoint: "api", Requests: policy}}}}}
	input, err := recoveryInput(run, execution.Reference{Test: test.ID, Attempt: 1})
	if err != nil || input.Browser == nil || input.Browser.Origin != "https://application.test" || !input.Browser.Authenticated || input.CredentialReference != "saved-reader" {
		t.Fatal("environment recovery lost its frozen target or binding", err)
	}
}

func TestRecoveryInputPreservesOriginalAttemptCaptureFlag(t *testing.T) {
	test := catalog.Test{ID: "TEST-1", Runner: "playwright", Selector: "@TEST-1", TimeoutSeconds: 1}
	run := Run{ID: "run", Tests: []catalog.Test{test}, TestCapture: map[string]capture.Policy{test.ID: capture.DefaultPolicy()}}
	for _, attempt := range []int{1, 2, 3} {
		input, err := recoveryInput(run, execution.Reference{Test: test.ID, Attempt: attempt})
		if err != nil || input.FailedBefore != (attempt > 1) {
			t.Fatalf("attempt %d changed its frozen capture flag: %v", attempt, err)
		}
	}
}

func TestPendingWorkerEvidenceRemainsEligibleForRecovery(t *testing.T) {
	run := Run{TestCapture: map[string]capture.Policy{"TEST-1": capture.DefaultPolicy()}, Capture: capture.DefaultPolicy()}
	report := execution.CodedReport{EvidencePending: true, EvidenceErrors: []string{"Hosted attachments are pending."}}
	var problems []string
	service := &Service{}
	if !service.captureCodedReport(context.Background(), run, "worker", execution.Result{TestID: "TEST-1", State: "passed"}, false, report, &problems) || len(problems) != 1 {
		t.Fatal("pending worker evidence was treated as a permanent omission", problems)
	}
}
