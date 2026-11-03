package access

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

type UserDefinition struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *Service) ListUsers(ctx context.Context, actor, cursor string, limit int) (storage.Page, error) {
	if err := s.Auth.Check(ctx, actor, "", "", "users.read"); err != nil {
		return storage.Page{}, err
	}
	page, err := s.page(ctx, "users", "", queryScope(actor, "users", "", false), cursor, limit)
	if err != nil {
		return page, err
	}
	allowed := []storage.Record{}
	for _, row := range page.Items {
		if err := s.Auth.Check(ctx, actor, "", "user/"+row.ID, "users.read"); err != nil {
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			return page, err
		}
		allowed = append(allowed, row)
	}
	page.Items = allowed
	return page, nil
}

func (s *Service) ReadUser(ctx context.Context, actor, id string) (storage.Record, error) {
	if err := s.Auth.Check(ctx, actor, "", "user/"+id, "users.read"); err != nil {
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, "users", id)
}

func (s *Service) SaveUser(ctx context.Context, actor string, d UserDefinition, revision int64) error {
	if err := s.Auth.Check(ctx, actor, "", "user/"+d.ID, "users.manage"); err != nil {
		return err
	}
	d.Name, d.Email = strings.TrimSpace(d.Name), strings.TrimSpace(d.Email)
	if !domain.ValidID(d.ID) || d.Name == "" || len(d.Name) > 256 || len(d.Email) > 254 {
		return errors.New("valid user identifier, name, and optional email are required")
	}
	if d.Email != "" {
		address, err := mail.ParseAddress(d.Email)
		if err != nil || address.Address != d.Email {
			return errors.New("email must be one valid address")
		}
	}
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	r, err := s.Store.Get(ctx, "users", d.ID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if revision != r.Revision {
		return storage.ErrConflict
	}
	var old identity.User
	if r.Revision > 0 {
		old, err = storage.Decode[identity.User](r)
		if err != nil {
			return err
		}
	}
	v := old
	v.ID, v.Name, v.Email = d.ID, d.Name, d.Email
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	if old.Email != d.Email {
		v.EmailVerified = false
	}
	c, err := storage.Put("users", d.ID, "", revision, v)
	if err != nil {
		return err
	}
	return s.commit(ctx, actor, "", "user.save", d.ID, "", old, v, version, c)
}

func (s *Service) UserState(ctx context.Context, actor, id, why string, revision int64, disabled bool) error {
	if err := s.Auth.Check(ctx, actor, "", "user/"+id, "users.manage"); err != nil {
		return err
	}
	if err := reason(why); err != nil {
		return err
	}
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	r, err := s.Store.Get(ctx, "users", id)
	if err != nil {
		return err
	}
	if revision < 1 || r.Revision != revision {
		return storage.ErrConflict
	}
	v, err := storage.Decode[identity.User](r)
	if err != nil {
		return err
	}
	old := v
	v.Disabled = disabled
	if disabled {
		v.SessionEpoch++
	}
	c, err := storage.Put("users", id, "", revision, v)
	if err != nil {
		return err
	}
	if disabled {
		if err := s.preserveAdministrator(ctx, c); err != nil {
			return err
		}
	}
	action := "user.restore"
	if disabled {
		action = "user.disable"
	}
	return s.commit(ctx, actor, "", action, id, why, old, v, version, c)
}
