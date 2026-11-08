package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedRecordingRequiresLocalAPIAndExclusiveRecorder(t *testing.T) {
	for name, options := range map[string][]string{
		"hosted":              {"--recording-embedded-cli=private-runtime-canary"},
		"artifact":            {"--serve-role=artifacts", "--recording-embedded-cli=private-runtime-canary"},
		"two local recorders": {"--local", "--recording-cli=private-runtime-canary", "--recording-embedded-cli=private-runtime-canary"},
		"companion":           {"--local", "--recording-companion", "--recording-embedded-cli=private-runtime-canary"},
	} {
		t.Run(name, func(t *testing.T) {
			originalFlags, originalArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
			flag.CommandLine = flag.NewFlagSet("embedded-recording-check", flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)
			database := filepath.Join(t.TempDir(), "must-not-exist.db")
			os.Args = append([]string{"teststudio", "--database=" + database}, options...)
			err := run()
			if err == nil || strings.Contains(err.Error(), "private-runtime-canary") {
				t.Fatal("invalid embedded recorder configuration accepted or disclosed", err)
			}
			if _, err := os.Stat(database); !os.IsNotExist(err) {
				t.Fatal("invalid embedded recorder configuration created state", err)
			}
		})
	}
}
