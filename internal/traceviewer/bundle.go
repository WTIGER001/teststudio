package traceviewer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const Version = "1.64.1"

type Bundle struct{ files map[string][]byte }

func Load(directory string) (*Bundle, error) {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("private trace viewer requires a regular trusted directory")
	}
	var manifest struct {
		Schema  int               `json:"schemaVersion"`
		Version string            `json:"playwrightVersion"`
		Files   map[string]string `json:"files"`
	}
	data, err := os.ReadFile(filepath.Join(directory, "teststudio-viewer.json"))
	if err != nil {
		return nil, errors.New("the private trace viewer manifest is unavailable")
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.Schema != 1 || manifest.Version != Version || len(manifest.Files) < 5 || len(manifest.Files) > 100 {
		return nil, errors.New("the private trace viewer must match its pinned runtime and manifest")
	}
	bundle := &Bundle{files: map[string][]byte{}}
	total := 0
	for name, expected := range manifest.Files {
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\:\x00") || path.Clean(name) != name {
			return nil, errors.New("invalid private trace viewer asset path")
		}
		filename := filepath.Join(directory, filepath.FromSlash(name))
		for parent := filename; parent != filepath.Clean(directory); parent = filepath.Dir(parent) {
			info, err := os.Lstat(parent)
			if err != nil || info.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("private trace viewer assets must not use symbolic links")
			}
		}
		info, err := os.Stat(filename)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 32<<20 {
			return nil, errors.New("invalid or oversized private trace viewer asset")
		}
		body, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != expected {
			return nil, errors.New("private trace viewer asset checksum differs from its build manifest")
		}
		total += len(body)
		if total > 32<<20 {
			return nil, errors.New("private trace viewer bundle exceeds 32 MiB")
		}
		bundle.files[name] = body
	}
	for _, name := range []string{"index.html", "snapshot.html", "sw.bundle.js", "sw.vendor.js", "teststudio-session.js", "LICENSE"} {
		if len(bundle.files[name]) == 0 {
			return nil, errors.New("private trace viewer bundle is incomplete")
		}
	}
	return bundle, nil
}

func (b *Bundle) Asset(name string) ([]byte, bool) {
	if b == nil {
		return nil, false
	}
	data, ok := b.files[name]
	return data, ok
}
