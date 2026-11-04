package localcontainer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/diagnostics"
	"github.com/wtiger001/teststudio/internal/execution"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func regularFile(root *os.Root, path string, limit int64) ([]byte, error) {
	if path == "" || filepath.Base(path) != path || strings.ContainsAny(path, "\\\x00") {
		return nil, errors.New("worker artifact path must be a plain filename")
	}
	info, err := root.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("worker artifact is not a regular file within its size limit")
	}
	f, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(data)) > limit {
		return nil, errors.New("worker artifact exceeds its size limit")
	}
	return data, err
}

func readReport(directory string, input execution.CodedInput, exitError error) (execution.CodedReport, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return execution.CodedReport{}, err
	}
	defer root.Close()
	data, err := regularFile(root, "report.json", 2<<20)
	if err != nil {
		return execution.CodedReport{}, fmt.Errorf("Coded worker result is unavailable: %w", err)
	}
	var report execution.CodedReport
	if err := json.Unmarshal(data, &report); err != nil {
		return report, errors.New("Coded worker result is invalid")
	}
	report.Result = execution.NormalizeTimeout(report.Result)
	if err := execution.ValidateCodedReport(input, report, exitError); err != nil {
		return report, err
	}
	var bytes int64
	accepted := []execution.Attachment{}
	for _, file := range report.Attachments {
		if execution.APIDiagnostics(input) && !execution.APITextAttachment(file) {
			report.EvidenceErrors = append(report.EvidenceErrors, "API attachment omitted: only bounded UTF-8 text and JSON support diagnostic redaction.")
			continue
		}
		if file.Kind != "screenshot" && file.Kind != "trace" && file.Kind != "video" && file.Kind != "attachment" && file.Kind != "log" {
			report.EvidenceErrors = append(report.EvidenceErrors, "Unsupported coded worker attachment kind.")
			continue
		}
		limit := min(input.Capture.MaxFileBytes, input.Capture.MaxBytes-bytes, 64<<20)
		if execution.APIDiagnostics(input) {
			limit = min(limit, diagnostics.MaxBytes)
		}
		body, err := regularFile(root, file.Path, limit)
		if err != nil {
			report.EvidenceErrors = append(report.EvidenceErrors, file.Filename+": "+err.Error())
			continue
		}
		bytes += int64(len(body))
		file.Data = body
		accepted = append(accepted, file)
	}
	report.Attachments = accepted
	if len(report.EvidenceErrors) > 20 {
		report.EvidenceErrors = report.EvidenceErrors[:20]
	}
	return report, nil
}
