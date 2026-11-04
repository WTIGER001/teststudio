package jobsupervisor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/wtiger001/teststudio/internal/suitespec"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func (p *Process) Discover(ctx context.Context, input workerprotocol.DiscoveryInput, onStart func(time.Time)) (Prepared, error) {
	if workerprotocol.ValidateDiscoveryInput(input) != nil || input.Runner != p.Runner || p.Directory == "" || p.Output == "" {
		return Prepared{}, ErrRunner
	}
	root, err := os.OpenRoot(p.Output)
	if err != nil {
		return Prepared{}, ErrRunner
	}
	defer root.Close()
	directory, err := root.Open(".")
	if err != nil {
		return Prepared{}, ErrRunner
	}
	entries, err := directory.ReadDir(2)
	directory.Close()
	if len(entries) != 0 || err != nil && !errors.Is(err, io.EOF) {
		return Prepared{}, ErrRunner
	}
	wall, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if onStart != nil {
		onStart(time.Now().UTC())
	}
	stdout, stderr, exit, runErr := p.runCommand(wall, nil, true, nil)
	log := append(append([]byte{}, stdout.Bytes()...), stderr.Bytes()...)
	log = scrub(log, p.PrivateValues)
	report := &workerprotocol.DiscoveryReport{Log: string(log[:min(len(log), 128<<10)]), LogTruncated: stdout.Truncated || stderr.Truncated || len(log) > 128<<10}
	prepared := Prepared{Submission: workerprotocol.Submission{ProcessExitCode: exit, Discovery: report}}
	report.Raw, err = readRegular(root, "discovery.json", suitespec.DiscoveryLimit)
	directory, openErr := root.Open(".")
	if openErr == nil {
		entries, openErr = directory.ReadDir(3)
		directory.Close()
	}
	if runErr != nil || exit != 0 || err != nil || openErr != nil && !errors.Is(openErr, io.EOF) || len(entries) != 1 || entries[0].Name() != "discovery.json" || !json.Valid(report.Raw) {
		report.Raw, report.Failure = nil, "worker"
		if errors.Is(wall.Err(), context.DeadlineExceeded) {
			report.Failure = "timeout"
		} else if errors.Is(wall.Err(), context.Canceled) {
			report.Failure = "cancelled"
		}
		return prepared, nil
	}
	report.Raw = scrub(report.Raw, p.PrivateValues)
	if !json.Valid(report.Raw) {
		report.Raw, report.Failure = nil, "worker"
	}
	return prepared, nil
}
