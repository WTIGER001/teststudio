package identity

import (
	"context"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (d *Directory) providerAt(ctx context.Context, id string, revision int64) (storage.Record, error) {
	row, err := d.Store.Get(ctx, "providers", id)
	if err != nil {
		return row, err
	}
	state, err := storage.Decode[struct {
		Disabled bool `json:"disabled"`
	}](row)
	if err != nil {
		return row, err
	}
	if revision < 1 || row.Revision != revision || row.Project != "" || state.Disabled {
		return row, storage.ErrConflict
	}
	return row, nil
}

func (d *Directory) ResolveProviderIdentity(ctx context.Context, provider string, revision int64, issuer, subject, name, email string, verified, provision bool) (User, error) {
	row, err := d.providerAt(ctx, provider, revision)
	if err != nil {
		return User{}, err
	}
	state, err := storage.Decode[struct {
		Issuer string `json:"issuer"`
	}](row)
	if err != nil {
		return User{}, err
	}
	if issuer == "" || state.Issuer != issuer {
		return User{}, storage.ErrConflict
	}
	return d.resolveExternal(ctx, issuer, subject, name, email, verified, provision, []storage.Change{{Record: row, Assert: true}})
}
