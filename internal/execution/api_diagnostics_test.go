package execution

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/diagnostics"
)

func TestAPIReportRedactsBeforeCheckpointWithoutChangingOutcomeOrTiming(t *testing.T) {
	input := CodedInput{Test: catalog.Test{Type: "api"}, Capture: capture.DefaultPolicy(), RedactionValues: []string{"bound-canary"}, Artifact: catalog.Artifact{Browser: &browserpolicy.Spec{Requests: &apipolicy.Spec{RedactFields: []string{"privateContact"}}}}}
	report := CodedReport{Result: Result{TestID: "API-1", State: "timed-out", FailureKind: "timeout", DurationMs: 123, Error: "Expected private-canary bound-canary", Steps: []Step{{Title: "GET https://example.test/?token=query-canary", Error: "actual response-canary"}}}, Attachments: []Attachment{
		{Kind: "log", Stream: "playwright.stdout", Filename: "private-canary.log", ContentType: "text/plain", Data: []byte("privateContact=private-canary\npassword=response-canary\n")},
		{Kind: "attachment", ContentType: "application/json", Data: []byte(`{"status":"up","token":"response-canary","authorization":"bound-canary"}`)},
		{Kind: "trace", ContentType: "application/zip", Data: []byte("binary-canary")},
		{Kind: "attachment", ContentType: "application/json", Data: []byte(`{"token":"invalid-canary`)},
		{Kind: "attachment", ContentType: "text/plain", Data: []byte{0xff}},
	}}
	clean := RedactAPIReport(input, report)
	if clean.Result.State != "timed-out" || clean.Result.FailureKind != "timeout" || clean.Result.DurationMs != 123 || clean.Result.DiagnosticsPolicy != diagnostics.Version {
		t.Fatal("observed result changed")
	}
	if len(clean.Attachments) != 2 || len(clean.EvidenceErrors) != 3 {
		t.Fatalf("omissions missing: %+v", clean)
	}
	all, _ := json.Marshal(clean)
	for _, file := range clean.Attachments {
		all = append(all, file.Data...)
	}
	if strings.Contains(string(all), "canary") {
		t.Fatalf("private canary remains: %s", all)
	}
	if !strings.Contains(string(clean.Attachments[1].Data), `"status":"up"`) {
		t.Fatal("response context lost")
	}
	if report.Result.Error != "Expected private-canary bound-canary" {
		t.Fatal("original observation mutated")
	}
}

func TestBrowserMediaAndOtherTestDiagnosticsStayAvailable(t *testing.T) {
	report := CodedReport{Result: Result{Error: "ordinary evidence"}, Attachments: []Attachment{{Kind: "screenshot", ContentType: "image/png", Data: []byte("image")}}}
	clean := RedactAPIReport(CodedInput{Test: catalog.Test{Type: "browser"}}, report)
	if len(clean.Attachments) != 1 || clean.Result.DiagnosticsPolicy != "" || clean.Result.Error != report.Result.Error {
		t.Fatal("browser evidence changed")
	}
}
