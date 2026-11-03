package postgres

import (
	"context"
	"errors"
	"strconv"

	permissions "github.com/wtiger001/go-permissions"
)

func (s *policyTransaction) ListGrants(ctx context.Context, q permissions.GrantQuery) (permissions.GrantQueryResult, error) {
	result := permissions.GrantQueryResult{Grants: []permissions.Grant{}}
	after := int64(0)
	if q.Cursor != "" {
		var err error
		after, err = strconv.ParseInt(q.Cursor, 10, 64)
		if err != nil || after < 0 {
			return result, errors.New("invalid grant cursor")
		}
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	where, args := grantFilter(q)
	if err := s.tx.QueryRowContext(ctx, `SELECT count(*) FROM teststudio_permissions.permission_grants WHERE `+where, args...).Scan(&result.TotalCount); err != nil {
		return result, err
	}
	args = append(args, after, limit+1)
	rows, err := s.tx.QueryContext(ctx, `SELECT `+grantColumns+` FROM teststudio_permissions.permission_grants WHERE `+where+` AND id>`+placeholder(len(args)-1)+` ORDER BY id LIMIT `+placeholder(len(args)), args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return result, err
		}
		result.Grants = append(result.Grants, g)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if len(result.Grants) > limit {
		result.Grants = result.Grants[:limit]
		result.NextCursor = strconv.FormatInt(result.Grants[limit-1].ID, 10)
	}
	return result, nil
}
