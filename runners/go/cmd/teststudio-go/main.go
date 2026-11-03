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
	if len(os.Args) > 1 {
		if len(os.Args) != 2 || os.Args[1] != "--discover" {
			fmt.Fprintln(os.Stderr, "unknown Go worker operation")
			os.Exit(2)
		}
		discovery, err := runner.Discover(ctx, "/suite")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		encoded, err := json.Marshal(discovery)
		if err != nil || len(encoded) > 8<<20 {
			fmt.Fprintln(os.Stderr, "discovery report is invalid or too large")
			os.Exit(2)
		}
		if err := os.WriteFile("/output/discovery.tmp", encoded, 0600); err != nil {
			fmt.Fprintln(os.Stderr, "discovery report could not be stored")
			os.Exit(2)
		}
		if err := os.Rename("/output/discovery.tmp", "/output/discovery.json"); err != nil {
			fmt.Fprintln(os.Stderr, "discovery report could not be committed")
			os.Exit(2)
		}
		return
	}
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
