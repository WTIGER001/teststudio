package jobsupervisor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/wtiger001/teststudio/internal/diagnostics"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func openRegular(root *os.Root, name string, limit int64) (*os.File, int64, error) {
	if !plainFile(name) || limit < 0 {
		return nil, 0, ErrRunner
	}
	entry, err := root.Lstat(name)
	if err != nil || !entry.Mode().IsRegular() {
		return nil, 0, ErrRunner
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, 0, ErrRunner
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		file.Close()
		return nil, 0, ErrRunner
	}
	return file, info.Size(), nil
}

func readRegular(root *os.Root, name string, limit int64) ([]byte, error) {
	file, size, err := openRegular(root, name, limit)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) != size || int64(len(data)) > limit {
		return nil, ErrRunner
	}
	return data, nil
}

func scrub(data []byte, values []string) []byte {
	for _, value := range values {
		if value != "" {
			data = bytes.ReplaceAll(data, []byte(value), []byte("[redacted]"))
		}
	}
	return data
}

func (p *Process) snapshot(root *os.Root, input execution.CodedInput, report execution.CodedReport, stdout, stderr *execution.Output) (Prepared, error) {
	directory, err := os.MkdirTemp(p.Temporary, "teststudio-job-evidence-")
	if err != nil {
		return Prepared{}, ErrRunner
	}
	prepared := Prepared{Cleanup: func() { _ = os.RemoveAll(directory) }}
	ok := false
	defer func() {
		if !ok {
			prepared.Cleanup()
		}
	}()
	encoded, err := json.Marshal(report)
	if err != nil {
		return Prepared{}, ErrRunner
	}
	if json.Unmarshal(scrub(encoded, p.PrivateValues), &report) != nil {
		return Prepared{}, ErrRunner
	}
	accepted := []execution.Attachment{}
	paths := map[string]bool{}
	var total int64
	add := func(file execution.Attachment, source io.Reader, limit int64) error {
		if len(prepared.Files) >= 32 || limit < 0 || !plainFile(file.Path) || !plainFile(file.Filename) || paths[file.Path] || len(file.Stream) > 128 || strings.ContainsAny(file.Stream+file.ContentType, "\r\n\x00") {
			return ErrRunner
		}
		if _, _, err := mime.ParseMediaType(file.ContentType); err != nil || len(file.ContentType) > 255 {
			return ErrRunner
		}
		if !execution.CaptureAttachment(input, report.Result, file) {
			return nil
		}
		if execution.APIDiagnostics(input) && !execution.APITextAttachment(file) {
			return ErrRunner
		}
		if execution.APIDiagnostics(input) {
			limit = min(limit, diagnostics.MaxBytes)
		}
		name := filepath.Join(directory, fmt.Sprintf("file-%02d", len(prepared.Files)))
		target, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		hash := sha256.New()
		n, copyErr := io.Copy(io.MultiWriter(target, hash), io.LimitReader(source, limit+1))
		closeErr := target.Close()
		if copyErr != nil || closeErr != nil || n > limit {
			os.Remove(name)
			return ErrRunner
		}
		if err := os.Chmod(name, 0400); err != nil {
			return err
		}
		descriptor := workerprotocol.File{Path: file.Path, Size: n, SHA256: hex.EncodeToString(hash.Sum(nil))}
		prepared.Files = append(prepared.Files, File{Descriptor: descriptor, Open: func() (io.ReadCloser, error) { return os.Open(name) }})
		prepared.Submission.Files = append(prepared.Submission.Files, descriptor)
		accepted = append(accepted, file)
		paths[file.Path] = true
		total += n
		return nil
	}
	for _, file := range report.Attachments {
		if !execution.CaptureAttachment(input, report.Result, file) {
			continue
		}
		limit := min(input.Capture.MaxFileBytes, input.Capture.MaxBytes-total, 64<<20)
		reader, _, err := openRegular(root, file.Path, limit)
		if err != nil {
			report.EvidenceErrors = appendProblem(report.EvidenceErrors, "A runner attachment was not an owned regular file within its capture limit.")
			continue
		}
		err = add(file, reader, limit)
		reader.Close()
		if err != nil {
			report.EvidenceErrors = appendProblem(report.EvidenceErrors, "A runner attachment could not enter bounded private evidence.")
		}
	}
	if input.Capture.Logs {
		for _, stream := range []struct {
			name   string
			output *execution.Output
		}{{"stdout", stdout}, {"stderr", stderr}} {
			if stream.output.Len() == 0 {
				continue
			}
			present := false
			for _, file := range accepted {
				if file.Kind == "log" && file.Stream == input.Test.Runner+"."+stream.name {
					present = true
				}
			}
			if present {
				continue
			}
			name := "supervisor-" + stream.name + ".log"
			for index := 1; paths[name]; index++ {
				name = fmt.Sprintf("supervisor-%s-%d.log", stream.name, index)
			}
			body := scrub(bytes.Clone(stream.output.Bytes()), p.PrivateValues)
			limit := max(min(input.Capture.MaxFileBytes, input.Capture.MaxBytes-total, 64<<20), 0)
			if execution.APIDiagnostics(input) {
				limit = min(limit, diagnostics.MaxBytes)
			}
			if int64(len(body)) > limit {
				body = body[:limit]
				report.EvidenceErrors = appendProblem(report.EvidenceErrors, "Worker process output exceeded its capture limit.")
			}
			if stream.output.Truncated {
				report.EvidenceErrors = appendProblem(report.EvidenceErrors, "Worker process output was truncated at 2 MiB.")
			}
			if len(body) == 0 {
				continue
			}
			file := execution.Attachment{Path: name, Filename: name, Kind: "log", Stream: input.Test.Runner + "." + stream.name, ContentType: "text/plain; charset=utf-8"}
			if err := add(file, bytes.NewReader(body), limit); err != nil {
				report.EvidenceErrors = appendProblem(report.EvidenceErrors, "Worker process output could not enter private evidence.")
			}
		}
	}
	report.Attachments = accepted
	prepared.Submission.Report = report
	ok = true
	return prepared, nil
}
