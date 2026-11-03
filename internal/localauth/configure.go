package localauth

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func Configure(ctx context.Context, directory *identity.Directory, enabled bool) error {
	for range 8 {
		version, err := directory.Revision(ctx)
		if err != nil {
			return err
		}
		row, err := directory.Store.Get(ctx, "local-auth-mode", "installation")
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		old := identity.LocalAuthMode{}
		if row.Revision > 0 {
			old, err = storage.Decode[identity.LocalAuthMode](row)
			if err != nil {
				return err
			}
			if old.Enabled == enabled {
				return nil
			}
		}
		state := identity.LocalAuthMode{Enabled: enabled}
		c, err := storage.Put("local-auth-mode", "installation", "", row.Revision, state)
		if err != nil {
			return err
		}
		audit, err := domain.Audit("operator", "", "identity.local-auth-mode", "installation", old, state)
		if err != nil {
			return err
		}
		if err := directory.CommitAt(ctx, version, c, audit); errors.Is(err, storage.ErrConflict) {
			continue
		} else {
			return err
		}
	}
	return storage.ErrConflict
}
