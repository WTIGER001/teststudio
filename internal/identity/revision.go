package identity

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (d *Directory) Revision(ctx context.Context) (storage.Record, error) {
	r, err := d.Store.Get(ctx, "identity-version", "directory")
	if errors.Is(err, storage.ErrNotFound) {
		return r, nil
	}
	return r, err
}

func (d *Directory) CommitAt(ctx context.Context, version storage.Record, changes ...storage.Change) error {
	c, err := storage.Put("identity-version", "directory", "", version.Revision, version.Revision+1)
	if err != nil {
		return err
	}
	return d.Store.Commit(ctx, append([]storage.Change{c}, changes...)...)
}

func (d *Directory) commit(ctx context.Context, changes ...storage.Change) error {
	for range 8 {
		version, err := d.Revision(ctx)
		if err != nil {
			return err
		}
		if err := d.CommitAt(ctx, version, changes...); !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return storage.ErrConflict
}
