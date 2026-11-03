package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

type bootstrapState struct {
	UserID  string `json:"userId"`
	Granted bool   `json:"granted"`
}

func bootstrapLocalAdministrator(ctx context.Context, auth *authorization.Service, recover bool) error {
	return seedAdministrator(ctx, auth, "local", recover, func(directory *identity.Directory) (identity.User, bool, error) {
		user, err := directory.User(ctx, "local-administrator")
		if !errors.Is(err, storage.ErrNotFound) {
			return user, false, err
		}
		user = identity.User{ID: "local-administrator", Name: "Local administrator"}
		err = directory.SaveUser(ctx, user, 0)
		return user, true, err
	})
}

func bootstrapExternalAdministrator(ctx context.Context, auth *authorization.Service, issuer, subject string) error {
	if issuer == "" || subject == "" {
		return errors.New("bootstrap issuer and subject are required together")
	}
	raw, _ := json.Marshal([]string{issuer, subject})
	sum := sha256.Sum256(raw)
	return seedAdministrator(ctx, auth, "external-"+hex.EncodeToString(sum[:]), false, func(directory *identity.Directory) (identity.User, bool, error) {
		link, err := directory.ExternalLink(ctx, issuer, subject)
		if err == nil {
			user, err := directory.User(ctx, link.UserID)
			return user, false, err
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return identity.User{}, false, err
		}
		user, err := directory.ResolveExternal(ctx, issuer, subject, "Installation administrator", "", false, true)
		return user, true, err
	})
}

func seedAdministrator(ctx context.Context, auth *authorization.Service, key string, recover bool, provision func(*identity.Directory) (identity.User, bool, error)) error {
	for range 8 {
		if !recover {
			if _, err := auth.Directory.Store.Get(ctx, "identity-bootstrap", key); err == nil {
				return nil
			} else if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
		}
		err := auth.WithAssignmentTransaction(ctx, func(bound *authorization.Service) error {
			marker, err := bound.Directory.Store.Get(ctx, "identity-bootstrap", key)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return err
			}
			if marker.Revision > 0 && !recover {
				return nil
			}
			user, created, err := provision(bound.Directory)
			if err != nil {
				return err
			}
			if recover {
				if err := recoverLocalAdministrator(ctx, bound.Directory, user.ID); err != nil {
					return err
				}
			}
			if created || recover {
				if err := bound.Engine.AssignRole(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: user.ID}, "builtin.teststudio.installation-admin", nil); err != nil {
					return err
				}
			}
			if marker.Revision > 0 {
				return nil
			}
			state := bootstrapState{UserID: user.ID, Granted: created || recover}
			c, err := storage.Put("identity-bootstrap", key, "", 0, state)
			if err != nil {
				return err
			}
			action := "identity.bootstrap-adopt"
			if state.Granted {
				action = "identity.bootstrap"
			}
			audit, err := domain.Audit("operator", "", action, user.ID, nil, state)
			if err != nil {
				return err
			}
			return bound.Directory.Store.Commit(ctx, c, audit)
		})
		if !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return storage.ErrConflict
}
