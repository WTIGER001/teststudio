package sourcebundle

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"path"
	"slices"
	"sort"
)

func Overlay(data []byte, root string, files map[string]string) ([]byte, error) {
	if len(files) == 0 {
		return data, nil
	}
	snapshot, err := ReadRoot(data, root)
	if err != nil {
		return nil, err
	}
	for name, value := range files {
		if !SafePath(name) || !slices.Contains([]string{".json", ".yaml", ".yml"}, path.Ext(name)) || len(value) > 1<<20 {
			return nil, errors.New("source overlay requires bounded relative JSON metadata")
		}
		snapshot.Files[name] = []byte(value)
		snapshot.Executable[name] = false
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	names := make([]string, 0, len(snapshot.Files))
	for name := range snapshot.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		archived := name
		if root != "" {
			archived = root + "/" + name
		}
		header := &zip.FileHeader{Name: archived, Method: zip.Deflate}
		mode := fs.FileMode(0600)
		if snapshot.Executable[name] {
			mode = 0700
		}
		header.SetMode(mode)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err = entry.Write(snapshot.Files[name]); err != nil {
			return nil, err
		}
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	if output.Len() > CompressedLimit {
		return nil, errors.New("source with metadata exceeds archive limit")
	}
	return output.Bytes(), nil
}
