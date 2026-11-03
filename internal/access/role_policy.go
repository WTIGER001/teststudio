package access

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RoleRule struct {
	Permission string             `json:"permission"`
	Effect     permissions.Effect `json:"effect"`
	Object     string             `json:"object,omitempty"`
	ExpiresAt  *time.Time         `json:"expiresAt,omitempty"`
}

type RolePolicy struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Project     string     `json:"projectId"`
	Disabled    bool       `json:"disabled"`
	Rules       []RoleRule `json:"rules"`
	Inherits    []string   `json:"inherits"`
}

type RolePolicyMutation struct {
	RolePolicy
	Reason string `json:"reason"`
}

func rolePolicyBucket(project string) string { return "access-roles/" + project }

func validCustomRoleID(id string) bool {
	return domain.ValidID(id) && id != "custom" && id != "choices" && !strings.HasPrefix(id, "builtin.")
}

func (s *Service) loadRolePolicy(ctx context.Context, id string) (RolePolicy, storage.Record, error) {
	if !validCustomRoleID(id) {
		return RolePolicy{}, storage.Record{}, errors.New("a valid custom role identifier is required")
	}
	index, err := s.Store.Get(ctx, "access-role-index", id)
	if err != nil {
		return RolePolicy{}, index, err
	}
	row, err := s.Store.Get(ctx, rolePolicyBucket(index.Project), id)
	if err != nil {
		return RolePolicy{}, row, err
	}
	value, err := storage.Decode[RolePolicy](row)
	if err != nil {
		return value, row, err
	}
	if value.ID != id || value.Project != index.Project || row.Project != index.Project {
		return RolePolicy{}, row, errors.New("role metadata does not match its stored scope")
	}
	return value, row, nil
}

func (s *Service) rolePolicyAuthority(ctx context.Context, actor string, value RolePolicy, manage bool) error {
	actions := []string{"roles.read"}
	if manage {
		actions = []string{"roles.manage", "permissions.grant"}
	}
	for _, action := range actions {
		if _, err := s.assignmentScope(ctx, actor, value.Project, action); err != nil {
			return err
		}
		if err := s.Auth.Check(ctx, actor, value.Project, "role/"+value.ID, action); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ReadRolePolicy(ctx context.Context, actor, id string) (storage.Record, error) {
	value, row, err := s.loadRolePolicy(ctx, id)
	if err != nil {
		return storage.Record{}, err
	}
	if err := s.rolePolicyAuthority(ctx, actor, value, false); err != nil {
		return storage.Record{}, err
	}
	return row, nil
}

func (s *Service) ListRolePolicies(ctx context.Context, actor, project, cursor string, limit int) (Page, error) {
	return s.listRolePolicies(ctx, actor, project, cursor, limit, "custom-roles")
}

func (s *Service) listRolePolicies(ctx context.Context, actor, project, cursor string, limit int, collection string) (Page, error) {
	if _, err := s.assignmentScope(ctx, actor, project, "roles.read"); err != nil {
		return Page{}, err
	}
	page, err := s.page(ctx, rolePolicyBucket(project), project, queryScope(actor, collection, project, false), cursor, limit)
	if err != nil {
		return Page{}, err
	}
	result := Page{Items: []storage.Record{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	for _, row := range page.Items {
		value, err := storage.Decode[RolePolicy](row)
		if err != nil {
			return result, err
		}
		if value.Project != project || row.Project != project || value.ID != row.ID {
			return result, errors.New("role metadata does not match its stored scope")
		}
		if err := s.rolePolicyAuthority(ctx, actor, value, false); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return result, err
		}
		result.Items = append(result.Items, row)
	}
	return result, nil
}

func normalizeRolePolicy(value RolePolicy) (RolePolicy, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Description = strings.TrimSpace(value.Description)
	if !validCustomRoleID(value.ID) || value.Name == "" || len(value.Name) > 200 || len(value.Description) > 4096 {
		return value, errors.New("a valid custom role identifier, name up to 200 bytes, and description up to 4096 bytes are required")
	}
	if value.Project != "" && !domain.ValidID(value.Project) {
		return value, errors.New("a valid project identifier is required")
	}
	if len(value.Rules) > 100 || len(value.Inherits) > 16 {
		return value, errors.New("a role permits at most 100 rules and 16 directly inherited roles")
	}
	value.Rules = slices.Clone(value.Rules)
	if value.Rules == nil {
		value.Rules = []RoleRule{}
	}
	value.Inherits = slices.Clone(value.Inherits)
	if value.Inherits == nil {
		value.Inherits = []string{}
	}
	slices.Sort(value.Inherits)
	for i, id := range value.Inherits {
		if !validCustomRoleID(id) || id == value.ID || (i > 0 && value.Inherits[i-1] == id) {
			return value, errors.New("inheritance requires different, unique custom role identifiers")
		}
	}
	registered := authorization.PermissionNames()
	seen := map[string]bool{}
	for i, rule := range value.Rules {
		if !slices.Contains(registered, rule.Permission) || (rule.Effect != permissions.EffectAllow && rule.Effect != permissions.EffectDeny) {
			return value, errors.New("each rule requires a registered permission and allow or deny effect")
		}
		if value.Project != "" && authorization.InstallationPermission(rule.Permission) {
			return value, errors.New("project roles cannot contain installation actions")
		}
		if rule.Object != "" && !validPolicyObject(rule.Object) {
			return value, errors.New("rule objects require an exact supported resource identifier")
		}
		key := rule.Permission + "\x00" + string(rule.Effect) + "\x00" + rule.Object
		if seen[key] {
			return value, errors.New("duplicate role rule")
		}
		seen[key] = true
		if rule.ExpiresAt != nil {
			expiry := rule.ExpiresAt.UTC().Truncate(time.Microsecond)
			rule.ExpiresAt = &expiry
			value.Rules[i] = rule
		}
	}
	return value, nil
}
