package localplaywright

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/execution"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
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
		return execution.CodedReport{}, fmt.Errorf("Playwright result is unavailable: %w", err)
	}
	var report execution.CodedReport
	if err := json.Unmarshal(data, &report); err != nil {
		return report, errors.New("Playwright result is invalid")
	}
	result := report.Result
	if report.Schema != "teststudio.playwright/v1" || report.TestID != input.Test.ID || report.Selector != input.Test.Selector || result.TestID != input.Test.ID {
		return report, errors.New("Playwright result does not match the selected test")
	}
	if result.State != "passed" && result.State != "failed" && result.State != "skipped" {
		return report, errors.New("Playwright result has no supported terminal state")
	}
	if math.IsNaN(result.DurationMs) || math.IsInf(result.DurationMs, 0) || result.DurationMs < 0 || result.DurationMs > 3600000 || result.StartedAt.IsZero() || result.FinishedAt.Before(result.StartedAt) || result.FinishedAt.After(time.Now().Add(time.Minute)) {
		return report, errors.New("Playwright timing is invalid")
	}
	if exitError != nil && (result.State != "failed" || result.FailureKind == "") {
		return report, errors.New("Playwright process failed without a matching failed result")
	}
	if len(result.Steps) > 500 || len(report.Attachments) > 32 || len(report.EvidenceErrors) > 20 || len(result.Error) > 16384 {
		return report, errors.New("Playwright result exceeds its contract limits")
	}
	for _, step := range result.Steps {
		if len(step.Title) > 2048 || len(step.Error) > 16384 || step.DurationMs < 0 || math.IsNaN(step.DurationMs) || math.IsInf(step.DurationMs, 0) {
			return report, errors.New("Playwright step is invalid")
		}
	}
	var bytes int64
	accepted := []execution.Attachment{}
	for _, file := range report.Attachments {
		if file.Kind != "screenshot" && file.Kind != "trace" && file.Kind != "video" && file.Kind != "attachment" && file.Kind != "log" {
			report.EvidenceErrors = append(report.EvidenceErrors, "Unsupported Playwright attachment kind.")
			continue
		}
		limit := min(input.Capture.MaxFileBytes, input.Capture.MaxBytes-bytes, 64<<20)
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
