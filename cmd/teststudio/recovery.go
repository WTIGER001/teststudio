package main

import (
	"context"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func recoverLocalAdministrator(ctx context.Context, directory *identity.Directory, user string) error {
	version, err := directory.Revision(ctx)
	if err != nil {
		return err
	}
	row, err := directory.Store.Get(ctx, "users", user)
	if err != nil {
		return err
	}
	v, err := storage.Decode[identity.User](row)
	if err != nil {
		return err
	}
	old := v
	v.Disabled = false
	v.SessionEpoch++
	profile, err := storage.Put("users", user, "", row.Revision, v)
	if err != nil {
		return err
	}
	audit, err := domain.Audit("operator", "", "identity.local-recovery", user, old, v)
	if err != nil {
		return err
	}
	return directory.CommitAt(ctx, version, profile, audit)
}
