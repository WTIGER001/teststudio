package postgres

import (
	"context"
	"errors"
	"strings"
)

func (s *policyTransaction) AddRoleInheritance(ctx context.Context, parent, child string) error {
	if parent == "" || child == "" || parent == child || strings.HasPrefix(parent, "builtin.") || strings.HasPrefix(child, "builtin.") {
		return errors.New("two different custom roles are required")
	}
	if _, err := s.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(884190213)`); err != nil {
		return err
	}
	for _, id := range []string{parent, child} {
		if _, err := s.RoleDefinition(ctx, id); err != nil {
			return err
		}
	}
	var cycle bool
	if err := s.tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teststudio_permissions.role_closure WHERE ancestor_role_id=$1 AND descendant_role_id=$2)`, child, parent).Scan(&cycle); err != nil {
		return err
	}
	if cycle {
		return errors.New("role inheritance cycle")
	}
	if _, err := s.tx.ExecContext(ctx, `INSERT INTO teststudio_permissions.role_inheritance (parent_role_id,child_role_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, parent, child); err != nil {
		return err
	}
	_, err := s.tx.ExecContext(ctx, `INSERT INTO teststudio_permissions.role_closure (ancestor_role_id,descendant_role_id,depth)
SELECT a.ancestor_role_id,d.descendant_role_id,a.depth+d.depth+1 FROM teststudio_permissions.role_closure a CROSS JOIN teststudio_permissions.role_closure d
WHERE a.descendant_role_id=$1 AND d.ancestor_role_id=$2 ON CONFLICT DO NOTHING`, parent, child)
	return err
}

func (s *policyTransaction) ExpandRoles(ctx context.Context, ids []string) ([]string, error) {
	result := []string{}
	if len(ids) == 0 {
		return result, nil
	}
	args := make([]any, len(ids))
	marks := make([]string, len(ids))
	for i, id := range ids {
		args[i] = id
		marks[i] = placeholder(i + 1)
	}
	rows, err := s.tx.QueryContext(ctx, `SELECT DISTINCT descendant_role_id FROM teststudio_permissions.role_closure WHERE ancestor_role_id IN (`+strings.Join(marks, ",")+`) ORDER BY descendant_role_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (s *policyTransaction) RemoveRoleInheritance(ctx context.Context, parent, child string) error {
	if parent == "" || child == "" || strings.HasPrefix(parent, "builtin.") || strings.HasPrefix(child, "builtin.") {
		return errors.New("two custom roles are required")
	}
	if _, err := s.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(884190213)`); err != nil {
		return err
	}
	for _, id := range []string{parent, child} {
		if _, err := s.RoleDefinition(ctx, id); err != nil {
			return err
		}
	}
	if _, err := s.tx.ExecContext(ctx, `DELETE FROM teststudio_permissions.role_inheritance WHERE parent_role_id=$1 AND child_role_id=$2`, parent, child); err != nil {
		return err
	}
	if _, err := s.tx.ExecContext(ctx, `DELETE FROM teststudio_permissions.role_closure`); err != nil {
		return err
	}
	_, err := s.tx.ExecContext(ctx, `WITH RECURSIVE reach(ancestor,descendant,depth) AS (
SELECT id,id,0 FROM teststudio_permissions.roles
UNION SELECT r.ancestor,i.child_role_id,r.depth+1 FROM reach r JOIN teststudio_permissions.role_inheritance i ON r.descendant=i.parent_role_id
) INSERT INTO teststudio_permissions.role_closure (ancestor_role_id,descendant_role_id,depth) SELECT ancestor,descendant,min(depth) FROM reach GROUP BY ancestor,descendant`)
	return err
}
