package localcontainer

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/diagnostics"
	"github.com/wtiger001/teststudio/internal/execution"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (e *Executor) copyFile(ctx context.Context, container, file, directory string, limit int64) error {
	if file == "" || filepath.Base(file) != file || strings.ContainsAny(file, "\\\x00") || limit < 0 {
		return errors.New("worker output path or byte limit is invalid")
	}
	command := exec.CommandContext(ctx, e.binary(), "cp", container+":/output/"+file, "-")
	reader, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	defer func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	archive := tar.NewReader(reader)
	header, err := archive.Next()
	if err != nil {
		return err
	}
	if header.Typeflag != tar.TypeReg || header.Size < 0 || header.Size > limit {
		return errors.New("worker output is not a regular file within its byte limit")
	}
	data, err := io.ReadAll(io.LimitReader(archive, limit+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > limit {
		return errors.New("worker output exceeds its byte limit")
	}
	if _, err := archive.Next(); !errors.Is(err, io.EOF) {
		return errors.New("worker output archive contains unexpected entries")
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	if err := command.Wait(); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, file), data, 0600)
}

func (e *Executor) collect(ctx context.Context, container, directory string, input execution.CodedInput) error {
	if err := e.copyFile(ctx, container, "report.json", directory, 2<<20); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(directory, "report.json"))
	if err != nil {
		return err
	}
	var report execution.CodedReport
	if err := json.Unmarshal(data, &report); err != nil {
		return err
	}
	if len(report.Attachments) > 32 {
		return errors.New("worker output exceeds the attachment limit")
	}
	var bytes int64
	problems := []string{}
	for _, file := range report.Attachments {
		if execution.APIDiagnostics(input) && !execution.APITextAttachment(file) {
			continue
		}
		limit := min(input.Capture.MaxFileBytes, input.Capture.MaxBytes-bytes, 64<<20)
		if execution.APIDiagnostics(input) {
			limit = min(limit, diagnostics.MaxBytes)
		}
		if err := e.copyFile(ctx, container, file.Path, directory, limit); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", file.Filename, err))
			continue
		}
		info, err := os.Stat(filepath.Join(directory, file.Path))
		if err != nil {
			return err
		}
		bytes += info.Size()
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
