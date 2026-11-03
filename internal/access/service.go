package access

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store              *storage.Store
	Auth               *authorization.Service
	LocalUser          string
	LocalAccounts      bool
	SessionIdleTimeout time.Duration
}

type Page struct {
	Items   []storage.Record `json:"items"`
	Cursor  string           `json:"cursor,omitempty"`
	Partial bool             `json:"partial"`
}

func reason(value string) error {
	if strings.TrimSpace(value) == "" || len(value) > 1024 {
		return errors.New("a reason of 1 through 1024 bytes is required")
	}
	return nil
}

func (s *Service) commit(ctx context.Context, actor, project, action, resource, why string, before, after any, version storage.Record, changes ...storage.Change) error {
	audit, err := domain.Audit(actor, project, action, resource, before, map[string]any{"value": after, "reason": why})
	if err != nil {
		return err
	}
	return s.Auth.Directory.CommitAt(ctx, version, append(changes, audit)...)
}

func (s *Service) group(ctx context.Context, actor, id, permission string) (identity.Group, storage.Record, error) {
	if !domain.ValidID(id) {
		return identity.Group{}, storage.Record{}, errors.New("valid group identifier is required")
	}
	r, err := s.Store.Get(ctx, "groups", id)
	if err != nil {
		return identity.Group{}, r, err
	}
	g, err := storage.Decode[identity.Group](r)
	if err != nil {
		return g, r, err
	}
	if g.Project != r.Project {
		return g, r, storage.ErrNotFound
	}
	if err := s.Auth.Check(ctx, actor, g.Project, "group/"+id, permission); err != nil {
		return identity.Group{}, storage.Record{}, err
	}
	return g, r, nil
}
