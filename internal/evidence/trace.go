package evidence

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
)

func (s *Service) ValidateTrace(ctx context.Context, project, run, id string) error {
	a, reader, err := s.Open(ctx, project, run, id)
	if err != nil {
		return err
	}
	defer reader.Close()
	if a.Kind != "trace" || a.Size < 1 || a.Size > 64<<20 {
		return errors.New("the trace exceeds the private viewer limits")
	}
	var source io.ReaderAt
	if value, ok := reader.(io.ReaderAt); ok {
		source = value
	} else {
		data, err := io.ReadAll(io.LimitReader(reader, 64<<20+1))
		if err != nil {
			return err
		}
		if int64(len(data)) != a.Size {
			return errors.New("the trace size does not match its retained metadata")
		}
		source = bytes.NewReader(data)
	}
	archive, err := zip.NewReader(source, a.Size)
	if err != nil || len(archive.File) > 10000 {
		return errors.New("the file is not a supported Playwright trace archive")
	}
	total := uint64(0)
	found := false
	for _, entry := range archive.File {
		if !fs.ValidPath(strings.TrimSuffix(entry.Name, "/")) || entry.Mode()&fs.ModeSymlink != 0 || entry.Flags&1 != 0 {
			return errors.New("the trace contains unsupported archive entries")
		}
		if entry.UncompressedSize64 > 256<<20 || total > 256<<20-entry.UncompressedSize64 {
			return errors.New("the expanded trace exceeds 256 MiB")
		}
		total += entry.UncompressedSize64
		found = found || strings.HasSuffix(entry.Name, ".trace")
	}
	if !found {
		return errors.New("the archive has no Playwright trace records")
	}
	return nil
}
