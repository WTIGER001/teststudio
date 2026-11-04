package execution

import (
	"encoding/json"
	"mime"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/diagnostics"
)

func APIDiagnostics(input CodedInput) bool { return input.Test.Type == "api" }

func APITextAttachment(file Attachment) bool {
	typeName, _, err := mime.ParseMediaType(file.ContentType)
	return err == nil && (file.Kind == "log" || file.Kind == "attachment") && (typeName == "text/plain" || typeName == "application/json" || strings.HasPrefix(typeName, "application/") && strings.HasSuffix(typeName, "+json"))
}

func RedactAPIReport(input CodedInput, report CodedReport) CodedReport {
	if !APIDiagnostics(input) {
		return report
	}
	fields := []string{}
	if input.Artifact.Browser != nil && input.Artifact.Browser.Requests != nil {
		fields = input.Artifact.Browser.Requests.RedactFields
	}
	r := diagnostics.New(fields, input.RedactionValues)
	r.Learn(report.Result.Error)
	for _, step := range report.Result.Steps {
		r.Learn(step.Title)
		r.Learn(step.Error)
	}
	for _, message := range report.EvidenceErrors {
		r.Learn(message)
	}
	for _, file := range report.Attachments {
		typeName, _, _ := mime.ParseMediaType(file.ContentType)
		isJSON := typeName == "application/json" || strings.HasSuffix(typeName, "+json")
		if APITextAttachment(file) && utf8.Valid(file.Data) && len(file.Data) <= diagnostics.MaxBytes && (!isJSON || json.Valid(file.Data)) {
			r.Learn(string(file.Data))
		}
	}
	report.Result.DiagnosticsPolicy = diagnostics.Version
	report.Result.Error = r.Text(report.Result.Error, 16384)
	report.Result.RunnerStatus = r.Text(report.Result.RunnerStatus, 128)
	report.Result.ExpectedStatus = r.Text(report.Result.ExpectedStatus, 128)
	report.Result.Steps = slices.Clone(report.Result.Steps)
	for index := range report.Result.Steps {
		step := &report.Result.Steps[index]
		step.Title = r.Text(step.Title, 2048)
		step.Error = r.Text(step.Error, 16384)
		step.Category = r.Text(step.Category, 128)
	}
	problems := []string{}
	for _, message := range report.EvidenceErrors {
		problems = append(problems, r.Text(message, 2048))
	}
	files := []Attachment{}
	for index, file := range report.Attachments {
		if !APITextAttachment(file) || !utf8.Valid(file.Data) || len(file.Data) > diagnostics.MaxBytes {
			problems = append(problems, "API attachment omitted: only bounded UTF-8 text and JSON support diagnostic redaction.")
			continue
		}
		typeName, _, _ := mime.ParseMediaType(file.ContentType)
		if typeName == "application/json" || strings.HasSuffix(typeName, "+json") {
			body, err := r.JSON(file.Data)
			if err != nil {
				problems = append(problems, err.Error())
				continue
			}
			if int64(len(body)) > input.Capture.MaxFileBytes {
				problems = append(problems, "Redacted API JSON exceeds the attachment limit.")
				continue
			}
			file.Data = body
		} else {
			file.Data = []byte(r.Text(string(file.Data), int(min(input.Capture.MaxFileBytes, diagnostics.MaxBytes))))
		}
		file.Filename = "api-diagnostic-" + strconv.Itoa(index) + ".txt"
		if typeName != "text/plain" {
			file.Filename = "api-diagnostic-" + strconv.Itoa(index) + ".json"
		}
		file.Stream = r.Text(file.Stream, 128)
		files = append(files, file)
	}
	report.Attachments = files
	if r.Limited() {
		problems = append(problems, "API diagnostic content omitted because redaction limits are exceeded.")
	}
	report.EvidenceErrors = problems[:min(len(problems), 20)]
	return report
}
