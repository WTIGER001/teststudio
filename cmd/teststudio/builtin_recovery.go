package main

import (
	"context"
	"errors"
	"slices"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func recoverBuiltinAdministrator(ctx context.Context, auth *authorization.Service) error {
	id := "builtin.teststudio.installation-admin"
	disabled, err := auth.Engine.DisabledBuiltInRoles(ctx)
	if err != nil {
		return err
	}
	row, err := auth.Directory.Store.Get(ctx, "access-builtin-roles", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	state := struct {
		ID       string `json:"id"`
		Disabled bool   `json:"disabled"`
	}{ID: id}
	if !slices.Contains(disabled, id) {
		if row.Revision == 0 {
			return nil
		}
		old, err := storage.Decode[struct {
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		}](row)
		if err == nil && old == state {
			return nil
		}
	}
	if err := auth.Engine.EnableBuiltInRole(ctx, id); err != nil {
		return err
	}
	change, err := storage.Put("access-builtin-roles", id, "", row.Revision, state)
	if err != nil {
		return err
	}
	audit, err := domain.Audit("operator", "", "role.builtin.recover", id, map[string]any{"metadata": row.Data, "nativeDisabled": slices.Contains(disabled, id)}, map[string]any{"value": state, "reason": "Explicit loopback administrator recovery"})
	if err != nil {
		return err
	}
	return auth.Directory.Store.Commit(ctx, change, audit)
}
