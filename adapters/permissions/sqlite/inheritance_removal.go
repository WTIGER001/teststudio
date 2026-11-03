package sqlite

import (
	"context"
	"errors"
	"strings"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Store) RemoveRoleInheritance(ctx context.Context, parent, child string) error {
	if parent == "" || child == "" || strings.HasPrefix(parent, "builtin.") || strings.HasPrefix(child, "builtin.") {
		return errors.New("two custom roles are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range []string{parent, child} {
		if _, err := s.RoleDefinition(ctx, id); err != nil {
			return err
		}
	}
	for range 8 {
		graph, err := s.db.Get(ctx, "permission-graph", "inheritance")
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		row, err := s.db.Get(ctx, "permission-inheritance", key(edge{Parent: parent, Child: child}))
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		version, err := storage.Put("permission-graph", "inheritance", "", graph.Revision, graph.Revision+1)
		if err != nil {
			return err
		}
		if err := s.db.Commit(ctx, storage.Change{Record: row, Delete: true}, version); err == nil {
			return nil
		} else if !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return storage.ErrConflict
}
