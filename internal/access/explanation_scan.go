package access

import (
	"context"
	"errors"
	"strings"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
)

func (s *Service) scanExplanation(ctx context.Context, engine *permissions.Service, user string, q ExplanationQuery, inventory explanationInventory, cursor *explanationCursor) ([]explanationProof, error) {
	result := []explanationProof{}
	if cursor.Path < 0 || cursor.Path > len(inventory.Paths) || (cursor.Phase != "direct" && cursor.Phase != "roles" && cursor.Phase != "done") {
		return nil, errors.New("invalid explanation cursor")
	}
	request := authorization.RequestFor(user, q.Project, q.Object, q.Permission)
	remaining := q.Limit
	for remaining > 0 && cursor.Phase != "done" {
		var path *explanationRolePath
		owners := inventory.Owners
		if cursor.Phase == "roles" {
			if cursor.Path == len(inventory.Paths) {
				cursor.Phase = "done"
				break
			}
			path = &inventory.Paths[cursor.Path]
			owners = []permissions.PrincipalRef{{Kind: permissions.PrincipalRole, ID: path.Role}}
		}
		var values []permissions.Grant
		next := ""
		if path != nil && strings.HasPrefix(path.Role, "builtin.") {
			if cursor.Native != "" {
				return nil, errors.New("invalid predefined source cursor")
			}
			var err error
			values, err = engine.GrantsForOwners(ctx, owners, request)
			if err != nil {
				return nil, err
			}
			if len(values) > 1 {
				return nil, errors.New("predefined permission source is ambiguous")
			}
		} else {
			page, err := s.Auth.Engine.GetStore().ListGrants(ctx, permissions.GrantQuery{Principals: owners, Permissions: []string{q.Permission}, Limit: remaining, Cursor: cursor.Native})
			if err != nil {
				return nil, err
			}
			values, next = page.Grants, page.NextCursor
			if next != "" && next == cursor.Native {
				return nil, errors.New("permission source pagination did not advance")
			}
		}
		remaining -= max(1, len(values))
		for _, grant := range values {
			if grant.ExpiresAt != nil && time.Now().Before(*grant.ExpiresAt) && (cursor.ExpiresAt == nil || grant.ExpiresAt.Before(*cursor.ExpiresAt)) {
				expiry := *grant.ExpiresAt
				cursor.ExpiresAt = &expiry
			}
			probe := authorization.GrantPath{Grant: grant, Principal: permissions.PrincipalRef{Kind: grant.OwnerKind, ID: grant.OwnerID}}
			if path != nil {
				probe.Principal = path.Principal
				assignment := path.Assignment
				probe.Assignment = &assignment
			}
			matched, err := s.Auth.ExplainGrantPath(ctx, engine, request, inventory.Groups, probe)
			if err != nil {
				return nil, err
			}
			status := "matching"
			if !matched && grant.IsExpiredAt(time.Now()) {
				probe.Grant.ExpiresAt = nil
				matched, err = s.Auth.ExplainGrantPath(ctx, engine, request, inventory.Groups, probe)
				if err != nil {
					return nil, err
				}
				status = "expired"
			}
			if matched {
				result = append(result, explanationProof{Path: path, Grant: grant, Status: status})
			}
		}
		cursor.Native = next
		if next == "" {
			if cursor.Phase == "direct" {
				cursor.Phase = "roles"
			} else {
				cursor.Path++
			}
			if cursor.Path == len(inventory.Paths) && cursor.Phase == "roles" {
				cursor.Phase = "done"
			}
		}
	}
	return result, nil
}
