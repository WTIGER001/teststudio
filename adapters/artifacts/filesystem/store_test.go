package filesystem_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
)

func TestArtifactsAreBoundedAndConfined(t *testing.T) {
	root := t.TempDir()
	s, err := filesystem.New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	object, err := s.Put(t.Context(), "project/run/log.txt", strings.NewReader("test evidence"), 100)
	if err != nil || object.Size != 13 || object.Checksum == "" {
		t.Fatalf("artifact: %+v %v", object, err)
	}
	r, err := s.Open(t.Context(), "project/run/log.txt")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil || string(data) != "test evidence" {
		t.Fatalf("artifact body: %s %v", data, err)
	}
	if _, err := s.Put(t.Context(), "project/run/large.txt", strings.NewReader("too much evidence"), 3); err == nil {
		t.Fatal("accepted oversized evidence")
	}
	if _, err := s.Open(t.Context(), "project/run/large.txt"); err == nil {
		t.Fatal("failed upload became available")
	}
	for _, key := range []string{"../outside", "/absolute", "project/../../outside", "project\\outside"} {
		if _, err := s.Put(t.Context(), key, strings.NewReader("bad"), 10); err == nil {
			t.Errorf("accepted key %s", key)
		}
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(t.Context(), "outside/escape.txt", strings.NewReader("bad"), 10); err == nil {
		t.Fatal("followed an escaping symlink")
	}
}
