package access

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"time"

	permissions "github.com/wtiger001/go-permissions"
)

type explanationRolePath struct {
	Principal  permissions.PrincipalRef
	Assignment permissions.RoleAssignment
	Role       string
}
type explanationInventory struct {
	Groups   []string
	Owners   []permissions.PrincipalRef
	Paths    []explanationRolePath
	Roles    map[string]permissions.Role
	Disabled []string
}
type explanationCursor struct {
	Stamp     string     `json:"stamp"`
	Phase     string     `json:"phase"`
	Path      int        `json:"path"`
	Native    string     `json:"native"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}
type explanationProof struct {
	Path   *explanationRolePath
	Grant  permissions.Grant
	Status string
}

func (s *Service) explanationInventory(ctx context.Context, engine *permissions.Service, user string) (explanationInventory, error) {
	result := explanationInventory{Owners: []permissions.PrincipalRef{{Kind: permissions.PrincipalUser, ID: user}}, Paths: []explanationRolePath{}, Roles: map[string]permissions.Role{}}
	var err error
	result.Groups, err = s.Auth.Directory.GetUserGroups(ctx, user)
	if err != nil {
		return result, err
	}
	result.Disabled, err = engine.DisabledBuiltInRoles(ctx)
	if err != nil {
		return result, err
	}
	slices.Sort(result.Disabled)
	for _, group := range result.Groups {
		result.Owners = append(result.Owners, permissions.PrincipalRef{Kind: permissions.PrincipalGroup, ID: group})
	}
	readRole := func(id string) (permissions.Role, error) {
		if value, ok := result.Roles[id]; ok {
			return value, nil
		}
		value, err := engine.RoleDefinition(ctx, id)
		if err == nil {
			result.Roles[id] = value
		}
		return value, err
	}
	seen := map[string]bool{}
	for _, owner := range result.Owners {
		assignments, err := engine.RoleAssignmentsForPrincipal(ctx, owner)
		if err != nil {
			return result, err
		}
		for _, assignment := range assignments {
			role, err := readRole(assignment.RoleID)
			if err != nil {
				return result, err
			}
			if role.IsDisabled {
				continue
			}
			if assignment.BindingValues == nil {
				assignment.BindingValues = map[string]any{}
			}
			roles, err := engine.ExpandRoles(ctx, []string{assignment.RoleID})
			if err != nil {
				return result, err
			}
			for _, id := range roles {
				if _, err := readRole(id); err != nil {
					return result, err
				}
				path := explanationRolePath{Principal: owner, Assignment: assignment, Role: id}
				raw, err := json.Marshal(path)
				if err != nil {
					return result, err
				}
				if seen[string(raw)] {
					continue
				}
				seen[string(raw)] = true
				result.Paths = append(result.Paths, path)
			}
		}
	}
	slices.SortFunc(result.Paths, func(a, b explanationRolePath) int {
		left, _ := json.Marshal(a)
		right, _ := json.Marshal(b)
		return slices.Compare(left, right)
	})
	return result, nil
}

func explanationContext(user string, q ExplanationQuery) string {
	value, _ := json.Marshal([]string{user, q.Project, q.Object, q.Permission})
	return string(value)
}
func explanationStamp(revision int64, inventory explanationInventory, result PermissionExplanation) (string, error) {
	value := struct {
		Revision   int64
		Inventory  explanationInventory
		User       string
		Project    string
		Object     string
		Permission string
		Allowed    bool
		Reason     string
	}{revision, inventory, result.User, result.Project, result.Object, result.Permission, result.Allowed, result.Reason}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
