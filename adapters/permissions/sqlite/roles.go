package sqlite

import (
	"context"
	"errors"
	"slices"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type edge struct {
	Parent string
	Child  string
}

func (s *Store) RoleDefinitions(ctx context.Context) ([]permissions.Role, error) {
	return values[permissions.Role](ctx, s.db, "permission-roles")
}

func (s *Store) RoleDefinition(ctx context.Context, id string) (permissions.Role, error) {
	r, err := s.db.Get(ctx, "permission-roles", id)
	if err != nil {
		return permissions.Role{}, err
	}
	return storage.Decode[permissions.Role](r)
}

func (s *Store) CreateRole(ctx context.Context, role permissions.Role) error {
	if err := role.Validate(); err != nil {
		return err
	}
	if strings.HasPrefix(role.ID, "builtin.") {
		return errors.New("built-in roles belong to the permission service")
	}
	return s.create(ctx, "permission-roles", role.ID, role)
}

func (s *Store) UpdateRole(ctx context.Context, role permissions.Role) error {
	if err := role.Validate(); err != nil {
		return err
	}
	if strings.HasPrefix(role.ID, "builtin.") {
		return errors.New("built-in roles are immutable")
	}
	r, err := s.db.Get(ctx, "permission-roles", role.ID)
	if err != nil {
		return err
	}
	c, err := storage.Put(r.Bucket, r.ID, "", r.Revision, role)
	if err != nil {
		return err
	}
	return s.db.Commit(ctx, c)
}

func (s *Store) DeleteRole(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	assignments, err := s.RoleAssignmentsForRoleID(ctx, id)
	if err != nil {
		return err
	}
	if len(assignments) > 0 {
		return errors.New("revoke role assignments before deleting the role")
	}
	edges, err := values[edge](ctx, s.db, "permission-inheritance")
	if err != nil {
		return err
	}
	for _, e := range edges {
		if e.Parent == id || e.Child == id {
			return errors.New("remove role inheritance before deleting the role")
		}
	}
	grants, err := s.GrantsForPrincipal(ctx, permissions.PrincipalRef{Kind: permissions.PrincipalRole, ID: id})
	if err != nil {
		return err
	}
	if len(grants) > 0 {
		return errors.New("remove role grants before deleting the role")
	}
	return s.delete(ctx, "permission-roles", id)
}

func (s *Store) AddRoleInheritance(ctx context.Context, parent, child string) error {
	if parent == "" || child == "" || parent == child {
		return errors.New("invalid role inheritance")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e := edge{Parent: parent, Child: child}
	for range 8 {
		graph, err := s.db.Get(ctx, "permission-graph", "inheritance")
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if _, err := s.db.Get(ctx, "permission-inheritance", key(e)); err == nil {
			return nil
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		expanded, err := s.ExpandRoles(ctx, []string{child})
		if err != nil {
			return err
		}
		if slices.Contains(expanded, parent) {
			return errors.New("role inheritance cycle")
		}
		entry, err := storage.Put("permission-inheritance", key(e), "", 0, e)
		if err != nil {
			return err
		}
		version, err := storage.Put("permission-graph", "inheritance", "", graph.Revision, graph.Revision+1)
		if err != nil {
			return err
		}
		if err := s.db.Commit(ctx, entry, version); err == nil {
			return nil
		} else if !errors.Is(err, storage.ErrConflict) {
			return err
		}
	}
	return storage.ErrConflict
}

func (s *Store) ExpandRoles(ctx context.Context, ids []string) ([]string, error) {
	edges, err := values[edge](ctx, s.db, "permission-inheritance")
	if err != nil {
		return nil, err
	}
	children := map[string][]string{}
	for _, e := range edges {
		children[e.Parent] = append(children[e.Parent], e.Child)
	}
	seen := map[string]bool{}
	queue := slices.Clone(ids)
	result := []string{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
		queue = append(queue, children[id]...)
	}
	slices.Sort(result)
	return result, nil
}
