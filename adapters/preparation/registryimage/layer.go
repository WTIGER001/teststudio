package registryimage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func sourceLayer(ctx context.Context, snapshot sourcebundle.Snapshot, origin suitespec.Origin) ([]byte, string, error) {
	if len(snapshot.Files) < 1 || len(snapshot.Files) > sourcebundle.EntryLimit {
		return nil, "", errors.New("source assembly requires a bounded source snapshot")
	}
	if snapshot.Manifest.Catalog == suitespec.ManifestFile || snapshot.Manifest.Catalog == suitespec.OriginFile {
		return nil, "", errors.New("source catalog cannot replace controlled image metadata")
	}
	files := map[string][]byte{}
	total := 0
	for name, data := range snapshot.Files {
		if !sourcebundle.SafePath(name) || len(data) > sourcebundle.FileLimit {
			return nil, "", errors.New("source assembly contains an unsafe or oversized file")
		}
		for _, part := range strings.Split(name, "/") {
			if strings.HasPrefix(part, ".wh.") {
				return nil, "", errors.New("source paths cannot contain OCI whiteout names")
			}
		}
		total += len(data)
		if total > sourcebundle.ExpandedLimit {
			return nil, "", errors.New("source assembly exceeds the expanded snapshot limit")
		}
		files[name] = data
	}
	for name := range files {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, exists := files[parent]; exists {
				return nil, "", errors.New("source assembly contains conflicting file and directory paths")
			}
		}
	}
	for name, value := range map[string]any{suitespec.ManifestFile: snapshot.Manifest, suitespec.OriginFile: origin} {
		data, err := json.Marshal(value)
		if err != nil {
			return nil, "", errors.New("source assembly metadata is invalid")
		}
		files[name] = data
	}
	for name := range files {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, exists := files[parent]; exists {
				return nil, "", errors.New("source metadata conflicts with a directory path")
			}
		}
	}
	files[".wh..wh..opq"] = nil
	entries := map[string]bool{"suite": true}
	for name := range files {
		name = "suite/" + name
		entries[name] = false
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			entries[parent] = true
			if len(entries) > 4*sourcebundle.EntryLimit {
				return nil, "", errors.New("source layer contains too many directory entries")
			}
		}
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)
	var output bytes.Buffer
	compression := gzip.NewWriter(&output)
	diff := sha256.New()
	archive := tar.NewWriter(io.MultiWriter(compression, diff))
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		source := strings.TrimPrefix(name, "suite/")
		data := files[source]
		header := &tar.Header{Name: name, Uid: 1000, Gid: 1000, Mode: 0600, Size: int64(len(data)), Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0), Format: tar.FormatPAX}
		if entries[name] {
			header.Typeflag, header.Mode, header.Size = tar.TypeDir, 0700, 0
		} else if source != suitespec.ManifestFile && source != suitespec.OriginFile && snapshot.Executable[source] {
			header.Mode = 0700
		}
		if err := archive.WriteHeader(header); err != nil {
			return nil, "", errors.New("source layer header could not be assembled")
		}
		if !entries[name] {
			if _, err := archive.Write(data); err != nil {
				return nil, "", errors.New("source layer could not be assembled")
			}
		}
	}
	if err := archive.Close(); err != nil {
		return nil, "", errors.New("source archive could not be completed")
	}
	if err := compression.Close(); err != nil {
		return nil, "", errors.New("source compression could not be completed")
	}
	return output.Bytes(), "sha256:" + hex.EncodeToString(diff.Sum(nil)), nil
}
