package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Store struct {
	db *storage.Store
	mu sync.Mutex
}

var _ permissions.PermissionStore = (*Store)(nil)
var _ permissions.BulkGrantStore = (*Store)(nil)

func New(db *storage.Store) *Store { return &Store{db: db} }

func key(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func values[T any](ctx context.Context, db *storage.Store, bucket string) ([]T, error) {
	rows, err := db.All(ctx, bucket, "")
	if err != nil {
		return nil, err
	}
	result := make([]T, 0, len(rows))
	for _, row := range rows {
		value, err := storage.Decode[T](row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (s *Store) create(ctx context.Context, bucket, id string, value any) error {
	c, err := storage.Put(bucket, id, "", 0, value)
	if err != nil {
		return err
	}
	return s.db.Commit(ctx, c)
}

func (s *Store) delete(ctx context.Context, bucket, id string) error {
	r, err := s.db.Get(ctx, bucket, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.db.Commit(ctx, storage.Change{Record: r, Delete: true})
}

func (s *Store) DisableBuiltInRole(ctx context.Context, id string) error {
	err := s.create(ctx, "permission-disabled", id, id)
	if errors.Is(err, storage.ErrConflict) {
		return nil
	}
	return err
}

func (s *Store) EnableBuiltInRole(ctx context.Context, id string) error {
	return s.delete(ctx, "permission-disabled", id)
}

func (s *Store) DisabledBuiltInRoles(ctx context.Context) ([]string, error) {
	return values[string](ctx, s.db, "permission-disabled")
}

func validGrant(g permissions.Grant) error {
	if err := (permissions.PrincipalRef{Kind: g.OwnerKind, ID: g.OwnerID}).Validate(); err != nil {
		return err
	}
	if g.OwnerKind == permissions.PrincipalTeam {
		return errors.New("teams receive roles rather than direct grants")
	}
	if g.PermissionName == "" || g.TeamScope == "" {
		return errors.New("permission and scope are required")
	}
	if g.Effect != permissions.EffectAllow && g.Effect != permissions.EffectDeny {
		return fmt.Errorf("invalid grant effect %q", g.Effect)
	}
	return nil
}
