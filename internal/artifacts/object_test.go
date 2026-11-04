package artifacts_test

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/artifacts"
)

func TestArtifactStagingBoundsBytesAndRemovesOwnedTemporaryFiles(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("TMPDIR", directory)
	staged, err := artifacts.Stage(t.Context(), strings.NewReader("private evidence"), 100)
	if err != nil {
		t.Fatal(err)
	}
	info, err := staged.Stat()
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("staging file is not private", err)
	}
	data, err := io.ReadAll(staged)
	if err != nil || string(data) != "private evidence" {
		t.Fatal("staging did not rewind the upload", err)
	}
	if err := staged.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := artifacts.Stage(t.Context(), strings.NewReader("too large"), 2); err == nil {
		t.Fatal("oversized source accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := artifacts.Stage(ctx, strings.NewReader("cancelled"), 100); !errors.Is(err, context.Canceled) {
		t.Fatal("staging ignored cancellation", err)
	}
	files, err := os.ReadDir(directory)
	if err != nil || len(files) != 0 {
		t.Fatal("owned upload files remain", len(files), err)
	}
}
