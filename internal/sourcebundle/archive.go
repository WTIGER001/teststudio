package sourcebundle

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

const CompressedLimit = 32 << 20
const ExpandedLimit = 128 << 20
const FileLimit = 16 << 20
const EntryLimit = 10000

type Snapshot struct {
	Digest        string             `json:"digest"`
	ArchiveRoot   string             `json:"archiveRoot,omitempty"`
	ManifestPath  string             `json:"manifestPath"`
	Manifest      suitespec.Manifest `json:"manifest"`
	Catalog       catalog.Document   `json:"catalog"`
	Files         map[string][]byte  `json:"-"`
	Executable    map[string]bool    `json:"-"`
	ExpandedBytes int                `json:"expandedBytes"`
}

func SafePath(name string) bool {
	return len(name) > 0 && len(name) <= 512 && utf8.ValidString(name) && fs.ValidPath(name) && name != "." && !strings.ContainsAny(name, "\\:\x00") && !strings.ContainsFunc(name, unicode.IsControl)
}

func Validate(data []byte, manifestPath, suite string) (Snapshot, error) {
	return ValidateRoot(data, manifestPath, suite, "")
}

func ReadRoot(data []byte, root string) (snapshot Snapshot, err error) {
	if root != "" && (!SafePath(root) || strings.Contains(root, "/")) {
		return snapshot, errors.New("repository archive root must identify one safe wrapper directory")
	}
	if len(data) < 1 || len(data) > CompressedLimit {
		return snapshot, errors.New("provide a ZIP of at most 32 MiB and a relative manifest path")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return snapshot, errors.New("source bundle is not a valid ZIP")
	}
	if len(archive.File) < 1 || len(archive.File) > EntryLimit {
		return snapshot, errors.New("ZIP must contain 1 through 10000 entries")
	}
	snapshot.Files = map[string][]byte{}
	snapshot.Executable = map[string]bool{}
	names := map[string]bool{}
	directories := map[string]bool{}
	for _, entry := range archive.File {
		name := strings.TrimSuffix(entry.Name, "/")
		mode := entry.Mode()
		if !SafePath(name) || (!mode.IsRegular() && !mode.IsDir()) || (strings.HasSuffix(entry.Name, "/") != mode.IsDir()) {
			return Snapshot{}, errors.New("ZIP entries must be relative regular files or directories without links")
		}
		if root != "" {
			if name == root {
				if !mode.IsDir() {
					return Snapshot{}, errors.New("repository archive wrapper must be a directory")
				}
				continue
			}
			if !strings.HasPrefix(name, root+"/") {
				return Snapshot{}, errors.New("repository archive contains files outside its wrapper directory")
			}
			name = strings.TrimPrefix(name, root+"/")
		}
		key := strings.ToLower(name)
		if names[key] {
			return Snapshot{}, fmt.Errorf("duplicate ZIP path: %s", name)
		}
		names[key] = true
		if mode.IsDir() {
			directories[key] = true
			continue
		}
		if entry.UncompressedSize64 > FileLimit || entry.UncompressedSize64 > uint64(ExpandedLimit-snapshot.ExpandedBytes) {
			return Snapshot{}, errors.New("ZIP exceeds the 16 MiB file or 128 MiB expanded limit")
		}
		stream, err := entry.Open()
		if err != nil {
			return Snapshot{}, errors.New("ZIP file cannot be opened")
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, FileLimit+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || len(content) > FileLimit || uint64(len(content)) != entry.UncompressedSize64 {
			return Snapshot{}, errors.New("ZIP file is corrupt or exceeds its declared size")
		}
		snapshot.ExpandedBytes += len(content)
		snapshot.Files[name] = content
		snapshot.Executable[name] = mode.Perm()&0111 != 0
	}
	for name := range names {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if names[parent] && !directories[parent] {
				return Snapshot{}, errors.New("ZIP contains conflicting file and directory paths")
			}
		}
	}
	if len(snapshot.Files) == 0 {
		return Snapshot{}, errors.New("ZIP contains no source files")
	}
	sum := sha256.Sum256(data)
	snapshot.Digest = "sha256:" + hex.EncodeToString(sum[:])
	snapshot.ArchiveRoot = root
	return snapshot, nil
}

func ValidateRoot(data []byte, manifestPath, suite, root string) (snapshot Snapshot, err error) {
	if !SafePath(manifestPath) {
		return snapshot, errors.New("invalid manifest path")
	}
	snapshot, err = ReadRoot(data, root)
	if err != nil {
		return snapshot, err
	}
	if err := decode(snapshot.Files, manifestPath, &snapshot.Manifest); err != nil {
		return Snapshot{}, fmt.Errorf("suite manifest: %w", err)
	}
	if err := suitespec.Validate(snapshot.Manifest); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Manifest.Suite != suite {
		return Snapshot{}, errors.New("bundle manifest does not match the destination suite")
	}
	if err := decode(snapshot.Files, snapshot.Manifest.Catalog, &snapshot.Catalog); err != nil {
		return Snapshot{}, fmt.Errorf("suite catalog: %w", err)
	}
	if err := suitespec.ValidateCatalog(snapshot.Manifest, snapshot.Catalog); err != nil {
		return Snapshot{}, err
	}
	if err := validateContext(snapshot); err != nil {
		return Snapshot{}, err
	}
	sum := sha256.Sum256(data)
	snapshot.Digest = "sha256:" + hex.EncodeToString(sum[:])
	snapshot.ManifestPath = manifestPath
	snapshot.ArchiveRoot = root
	return snapshot, nil
}

func decode(files map[string][]byte, name string, value any) error {
	data, ok := files[name]
	if !ok {
		return errors.New("declared file is absent from the source bundle")
	}
	format := "json"
	if extension := path.Ext(name); extension == ".yaml" || extension == ".yml" {
		format = "yaml"
	}
	return definitions.Decode(format, string(data), value)
}

func validateContext(snapshot Snapshot) error {
	root := snapshot.Manifest.ModuleRoot
	required := []string{}
	switch snapshot.Manifest.Runner {
	case "go":
		required = []string{path.Join(root, "go.mod")}
	case "playwright":
		config := snapshot.Manifest.Tests.Config
		if config == "" {
			config = "playwright.config.ts"
		}
		required = []string{path.Join(root, "package.json"), path.Join(root, "package-lock.json"), path.Join(root, config)}
	}
	for _, name := range required {
		if _, ok := snapshot.Files[name]; !ok {
			return fmt.Errorf("source bundle requires %s", name)
		}
	}
	for _, test := range snapshot.Catalog.Tests {
		if _, ok := snapshot.Files[test.Source]; !ok {
			return fmt.Errorf("declared test source is absent: %s", test.Source)
		}
	}
	return nil
}

func (s Snapshot) Paths() []string {
	result := make([]string, 0, len(s.Files))
	for name := range s.Files {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

func WrappedRoot(data []byte) (string, error) {
	if len(data) < 1 || len(data) > CompressedLimit {
		return "", errors.New("repository archive exceeds its compressed limit")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", errors.New("repository archive is not a valid ZIP")
	}
	if len(archive.File) < 1 || len(archive.File) > EntryLimit {
		return "", errors.New("repository archive entry count exceeds its limit")
	}
	root := ""
	for _, entry := range archive.File {
		name := strings.TrimSuffix(entry.Name, "/")
		if !SafePath(name) {
			return "", errors.New("repository archive contains an unsafe path")
		}
		first, _, nested := strings.Cut(name, "/")
		if !nested && !entry.Mode().IsDir() {
			return "", errors.New("repository archive requires one wrapper directory")
		}
		if root == "" {
			root = first
		} else if root != first {
			return "", errors.New("repository archive contains multiple wrapper directories")
		}
	}
	return root, nil
}
