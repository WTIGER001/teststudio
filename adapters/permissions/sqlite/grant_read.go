package sqlite

import (
	"context"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Store) ReadGrant(ctx context.Context, id int64) (permissions.Grant, error) {
	if id < 1 {
		return permissions.Grant{}, storage.ErrNotFound
	}
	row, err := s.db.Get(ctx, "permission-grants", grantID(id))
	if err != nil {
		return permissions.Grant{}, err
	}
	return storage.Decode[permissions.Grant](row)
}
