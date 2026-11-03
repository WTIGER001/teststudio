package sqlite

import (
	"context"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Store) AssignmentPage(ctx context.Context, cursor string, limit int) (authorization.AssignmentPage, error) {
	page, err := s.db.List(ctx, "permission-assignments", "", cursor, limit)
	if err != nil {
		return authorization.AssignmentPage{}, err
	}
	result := authorization.AssignmentPage{Items: []permissions.RoleAssignmentHit{}, Cursor: page.Cursor}
	for _, row := range page.Items {
		a, err := storage.Decode[assignment](row)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, permissions.RoleAssignmentHit{PrincipalRef: a.Principal, RoleID: a.Value.RoleID, BindingValues: a.Value.BindingValues})
	}
	return result, nil
}
