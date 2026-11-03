package identity

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/storage"
	"time"
)

type LocalCredential struct {
	UserID     string    `json:"userId"`
	Hash       string    `json:"hash"`
	Disabled   bool      `json:"disabled"`
	MustChange bool      `json:"mustChangePassword"`
	CreatedAt  time.Time `json:"createdAt"`
}

type LocalAuthMode struct {
	Enabled bool `json:"enabled"`
}

func (d *Directory) LocalAccountsEnabled(ctx context.Context) (bool, error) {
	row, err := d.Store.Get(ctx, "local-auth-mode", "installation")
	if errors.Is(err, storage.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	state, err := storage.Decode[LocalAuthMode](row)
	return state.Enabled, err
}
