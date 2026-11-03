package postgres

import (
	"context"

	permissions "github.com/wtiger001/go-permissions"
)

func (s *Store) GrantsForPrincipal(ctx context.Context, p permissions.PrincipalRef) ([]permissions.Grant, error) {
	return s.GrantsForOwners(ctx, []permissions.PrincipalRef{p}, permissions.Request{})
}

func (s *Store) GrantsForOwners(ctx context.Context, owners []permissions.PrincipalRef, req permissions.Request) ([]permissions.Grant, error) {
	if len(owners) == 0 {
		return []permissions.Grant{}, nil
	}
	for _, p := range owners {
		if err := p.Validate(); err != nil {
			return nil, err
		}
	}
	q := permissions.GrantQuery{Principals: owners}
	if req.Perm != "" {
		q.Permissions = []string{req.Perm}
	}
	where, args := grantFilter(q)
	rows, err := s.pool.Query(ctx, `SELECT `+grantColumns+` FROM teststudio_permissions.permission_grants WHERE `+where+` AND (expires_at IS NULL OR expires_at>now()) ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []permissions.Grant{}
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}
