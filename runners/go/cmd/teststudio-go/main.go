package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/wtiger001/teststudio/runners/go/runner"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 65537))
	var input runner.Input
	if err != nil || len(data) > 65536 || json.Unmarshal(data, &input) != nil {
		fmt.Fprintln(os.Stderr, "Go worker input is invalid or exceeds 64 KiB")
		os.Exit(2)
	}
	report := runner.Execute(ctx, input, "/suite", "/output")
	encoded, err := json.Marshal(report)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.WriteFile("/output/report.tmp", encoded, 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.Rename("/output/report.tmp", "/output/report.json"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if report.Result.State != "passed" && report.Result.State != "skipped" {
		os.Exit(1)
	}
}
