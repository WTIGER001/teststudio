package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
)

func (s *Store) AssignmentPage(ctx context.Context, cursor string, limit int) (authorization.AssignmentPage, error) {
	if limit < 1 || limit > 1000 {
		return authorization.AssignmentPage{}, errors.New("limit must be between 1 and 1000")
	}
	var after int64
	if cursor != "" {
		var err error
		after, err = strconv.ParseInt(cursor, 10, 64)
		if err != nil || after < 1 {
			return authorization.AssignmentPage{}, errors.New("invalid assignment cursor")
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT id,role_id,binding_values,principal_kind,principal_id FROM principal_roles WHERE id>$1 ORDER BY id LIMIT $2`, after, limit+1)
	if err != nil {
		return authorization.AssignmentPage{}, err
	}
	defer rows.Close()
	result := authorization.AssignmentPage{Items: []permissions.RoleAssignmentHit{}}
	for rows.Next() {
		var id int64
		var item permissions.RoleAssignmentHit
		var raw []byte
		if err := rows.Scan(&id, &item.RoleID, &raw, &item.Kind, &item.ID); err != nil {
			return result, err
		}
		if len(result.Items) == limit {
			result.Cursor = strconv.FormatInt(after, 10)
			break
		}
		if err := json.Unmarshal(raw, &item.BindingValues); err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
		after = id
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	return result, nil
}
