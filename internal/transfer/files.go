package transfer

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/wtiger001/teststudio/internal/storage"
)

func ExportFile(ctx context.Context, source *storage.Store, filename string) (Stats, error) {
	if filename == "" {
		return Stats{}, errors.New("an archive output path is required")
	}
	if _, err := os.Lstat(filename); err == nil {
		return Stats{}, errors.New("archive output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Stats{}, err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return Stats{}, err
	}
	file, err := os.CreateTemp(filepath.Dir(filename), ".teststudio-installation-*")
	if err != nil {
		return Stats{}, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	stats, err := Export(ctx, source, file)
	if err != nil {
		return Stats{}, err
	}
	if err := file.Sync(); err != nil {
		return Stats{}, err
	}
	if err := file.Close(); err != nil {
		return Stats{}, err
	}
	if err := os.Link(file.Name(), filename); err != nil {
		return Stats{}, errors.New("archive output already exists or cannot be published")
	}
	return stats, nil
}
