package sqlite

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func grantID(id int64) string { return fmt.Sprintf("%020d", id) }

func (s *Store) CreateGrant(ctx context.Context, grant permissions.Grant) error {
	return s.CreateGrants(ctx, []permissions.Grant{grant})
}

func (s *Store) CreateGrants(ctx context.Context, grants []permissions.Grant) error {
	if len(grants) == 0 {
		return nil
	}
	if len(grants) > 100 {
		return errors.New("maximum 100 grants per transaction")
	}
	for _, g := range grants {
		if err := validGrant(g); err != nil {
			return err
		}
	}
	changes := []storage.Change{}
	for _, g := range grants {
		id, err := s.db.Next(ctx, "permission-grants")
		if err != nil {
			return err
		}
		g.ID = id
		c, err := storage.Put("permission-grants", grantID(id), "", 0, g)
		if err != nil {
			return err
		}
		changes = append(changes, c)
	}
	return s.db.Commit(ctx, changes...)
}

func (s *Store) GrantsForPrincipal(ctx context.Context, p permissions.PrincipalRef) ([]permissions.Grant, error) {
	return s.GrantsForOwners(ctx, []permissions.PrincipalRef{p}, permissions.Request{})
}

func (s *Store) GrantsForOwners(ctx context.Context, owners []permissions.PrincipalRef, req permissions.Request) ([]permissions.Grant, error) {
	all, err := values[permissions.Grant](ctx, s.db, "permission-grants")
	if err != nil {
		return nil, err
	}
	result := []permissions.Grant{}
	for _, g := range all {
		if g.IsExpiredAt(time.Now()) {
			continue
		}
		if !slices.Contains(owners, permissions.PrincipalRef{Kind: g.OwnerKind, ID: g.OwnerID}) {
			continue
		}
		if req.Perm != "" && g.PermissionName != req.Perm {
			continue
		}
		result = append(result, g)
	}
	return result, nil
}

func (s *Store) PrincipalsWithGrant(ctx context.Context, req permissions.Request) ([]permissions.PrincipalHit, error) {
	if req.Perm == "" {
		return nil, errors.New("permission is required")
	}
	all, err := values[permissions.Grant](ctx, s.db, "permission-grants")
	if err != nil {
		return nil, err
	}
	matching := []permissions.Grant{}
	denied := map[permissions.PrincipalRef]bool{}
	now := time.Now()
	for _, g := range all {
		if g.IsExpiredAt(now) || g.PermissionName != req.Perm || (g.TeamScope != "*" && g.TeamScope != req.TeamID) || (g.ObjectScope != nil && *g.ObjectScope != "*" && *g.ObjectScope != req.Object) {
			continue
		}
		matching = append(matching, g)
		if g.Effect == permissions.EffectDeny {
			denied[permissions.PrincipalRef{Kind: g.OwnerKind, ID: g.OwnerID}] = true
		}
	}
	result := []permissions.PrincipalHit{}
	seen := map[string]bool{}
	for _, g := range matching {
		if g.Effect != permissions.EffectAllow || denied[permissions.PrincipalRef{Kind: g.OwnerKind, ID: g.OwnerID}] {
			continue
		}
		hit := permissions.PrincipalHit{Kind: g.OwnerKind, ID: g.OwnerID, TeamScope: g.TeamScope, ObjectScope: g.ObjectScope, PermissionName: g.PermissionName}
		identity := key(hit)
		if !seen[identity] {
			result = append(result, hit)
			seen[identity] = true
		}
	}
	return result, nil
}

func (s *Store) DeleteGrantsForOwner(ctx context.Context, kind permissions.PrincipalKind, id string) error {
	all, err := s.db.All(ctx, "permission-grants", "")
	if err != nil {
		return err
	}
	changes := []storage.Change{}
	for _, r := range all {
		g, err := storage.Decode[permissions.Grant](r)
		if err != nil {
			return err
		}
		if g.OwnerKind == kind && g.OwnerID == id {
			changes = append(changes, storage.Change{Record: r, Delete: true})
		}
	}
	if len(changes) == 0 {
		return nil
	}
	return s.db.Commit(ctx, changes...)
}

func (s *Store) RemoveGrant(ctx context.Context, id int64) error {
	return s.delete(ctx, "permission-grants", grantID(id))
}

func (s *Store) ListGrants(ctx context.Context, q permissions.GrantQuery) (permissions.GrantQueryResult, error) {
	all, err := values[permissions.Grant](ctx, s.db, "permission-grants")
	if err != nil {
		return permissions.GrantQueryResult{}, err
	}
	after := int64(0)
	if q.Cursor != "" {
		after, err = strconv.ParseInt(q.Cursor, 10, 64)
		if err != nil || after < 0 {
			return permissions.GrantQueryResult{}, errors.New("invalid grant cursor")
		}
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	result := permissions.GrantQueryResult{Grants: []permissions.Grant{}}
	matched := []permissions.Grant{}
	for _, g := range all {
		if len(q.Principals) > 0 && !slices.Contains(q.Principals, permissions.PrincipalRef{Kind: g.OwnerKind, ID: g.OwnerID}) {
			continue
		}
		if len(q.TeamScopes) > 0 && !slices.Contains(q.TeamScopes, g.TeamScope) {
			continue
		}
		if len(q.ObjectScopes) > 0 && (g.ObjectScope == nil || !slices.Contains(q.ObjectScopes, *g.ObjectScope)) {
			continue
		}
		if len(q.Permissions) > 0 && !slices.Contains(q.Permissions, g.PermissionName) {
			continue
		}
		if !strings.HasPrefix(g.PermissionName, q.PermissionPrefix) {
			continue
		}
		result.TotalCount++
		if g.ID > after {
			matched = append(matched, g)
		}
	}
	slices.SortFunc(matched, func(a, b permissions.Grant) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	if len(matched) > limit {
		result.NextCursor = strconv.FormatInt(matched[limit-1].ID, 10)
		matched = matched[:limit]
	}
	result.Grants = matched
	return result, nil
}
