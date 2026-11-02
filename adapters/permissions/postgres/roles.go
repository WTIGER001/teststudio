package postgres

import (
	"context"
	"errors"
)

func (s *Store) AddRoleInheritance(ctx context.Context, parent, child string) error {
	if parent == "" || child == "" || parent == child {
		return errors.New("invalid role inheritance")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(884190213)"); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM roles WHERE id=$1 OR id=$2", parent, child).Scan(&count); err != nil {
		return err
	}
	if count != 2 {
		return errors.New("both inheritance roles must exist in the custom role store")
	}
	var cycle bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM role_closure WHERE ancestor_role_id=$1 AND descendant_role_id=$2)", child, parent).Scan(&cycle); err != nil {
		return err
	}
	if cycle {
		return errors.New("role inheritance cycle")
	}
	if _, err := tx.Exec(ctx, "INSERT INTO role_inheritance (parent_role_id,child_role_id) VALUES ($1,$2) ON CONFLICT DO NOTHING", parent, child); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_closure (ancestor_role_id,descendant_role_id,depth)
SELECT a.ancestor_role_id,d.descendant_role_id,a.depth+d.depth+1
FROM role_closure a CROSS JOIN role_closure d
WHERE a.descendant_role_id=$1 AND d.ancestor_role_id=$2 ON CONFLICT DO NOTHING`, parent, child); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
