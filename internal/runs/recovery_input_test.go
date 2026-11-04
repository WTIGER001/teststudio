package runs

import (
	"context"
	"testing"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
)

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
