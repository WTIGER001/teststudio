package access

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type ExplanationQuery struct {
	Application string
	Project     string
	Object      string
	Permission  string
	Cursor      string
	Limit       int
}
type PermissionSource struct {
	ScopeKnown            bool                      `json:"scopeKnown"`
	ID                    string                    `json:"id"`
	Kind                  string                    `json:"kind"`
	Effect                permissions.Effect        `json:"effect"`
	Status                string                    `json:"status"`
	Restricted            bool                      `json:"restricted"`
	PrincipalKind         permissions.PrincipalKind `json:"principalKind,omitempty"`
	PrincipalID           string                    `json:"principalId,omitempty"`
	PrincipalName         string                    `json:"principalName,omitempty"`
	RoleID                string                    `json:"roleId,omitempty"`
	RoleName              string                    `json:"roleName,omitempty"`
	GrantID               string                    `json:"grantId,omitempty"`
	AssignedRoleID        string                    `json:"assignedRoleId,omitempty"`
	AssignedRoleName      string                    `json:"assignedRoleName,omitempty"`
	Project               string                    `json:"projectId,omitempty"`
	TeamScope             string                    `json:"teamScope,omitempty"`
	ObjectScope           string                    `json:"objectScope,omitempty"`
	AssignmentProject     string                    `json:"assignmentProject,omitempty"`
	AssignmentApplication string                    `json:"assignmentApplication,omitempty"`
	ExpiresAt             *time.Time                `json:"expiresAt,omitempty"`
	RestrictedFields      []string                  `json:"restrictedFields,omitempty"`
}
type PermissionExplanation struct {
	Application      string                               `json:"applicationId,omitempty"`
	User             string                               `json:"userId"`
	UserName         string                               `json:"userName"`
	Project          string                               `json:"projectId"`
	Object           string                               `json:"object"`
	Permission       string                               `json:"permission"`
	Allowed          bool                                 `json:"allowed"`
	Reason           string                               `json:"reason"`
	UserDisabled     bool                                 `json:"userDisabled"`
	CheckedAt        time.Time                            `json:"checkedAt"`
	Items            []PermissionSource                   `json:"items"`
	Cursor           string                               `json:"cursor,omitempty"`
	Partial          bool                                 `json:"partial"`
	CanInspectOthers bool                                 `json:"canInspectOthers"`
	Permissions      []authorization.PermissionDefinition `json:"permissions"`
}

func (s *Service) explanationAuthority(ctx context.Context, actor, user string, q ExplanationQuery) error {
	if actor == "" {
		return authorization.ErrForbidden
	}
	current, err := s.Auth.Directory.User(ctx, actor)
	if err != nil {
		return err
	}
	if current.Disabled {
		return authorization.ErrForbidden
	}
	if actor != user {
		for _, action := range []string{"members.read", "members.manage", "permissions.read"} {
			if _, err := s.assignmentScope(ctx, actor, q.Project, action); err != nil {
				return err
			}
			if err := s.Auth.Check(ctx, actor, q.Project, "user/"+user, action); err != nil {
				return err
			}
		}
		if err := s.Auth.Check(ctx, actor, q.Project, q.Object, "permissions.read"); err != nil {
			return err
		}
	} else if q.Project != "" && q.Object == "" {
		if _, err := s.assignmentScope(ctx, actor, q.Project, "projects.read"); err != nil {
			return err
		}
	}
	if q.Object != "" && actor == user {
		err := s.Auth.Check(ctx, actor, q.Project, q.Object, "permissions.read")
		if errors.Is(err, authorization.ErrForbidden) {
			parts := splitPolicyObject(q.Object)
			action := map[string]string{"suite": "catalog.read", "test": "catalog.read", "run": "runs.read", "environment": "environments.read", "application": "projects.read", "group": "groups.read", "user": "users.read", "role": "roles.read", "grant": "permissions.read", "provider": "providers.read", "pending-access": "members.read"}[parts[0]]
			if !(q.Project == "" && q.Object == "user/"+actor) {
				err = s.Auth.Check(ctx, actor, q.Project, q.Object, action)
			} else {
				err = nil
			}
		}
		if err != nil {
			return err
		}
	}
	if q.Project != "" && q.Object != "" {
		row, err := s.Store.Get(ctx, "projects", q.Project)
		if err != nil {
			return err
		}
		value, err := storage.Decode[struct {
			ID string `json:"id"`
		}](row)
		if err != nil {
			return err
		}
		if row.Project != q.Project || value.ID != q.Project {
			return storage.ErrNotFound
		}
	}
	if q.Object != "" && splitPolicyObject(q.Object)[0] == "pending-access" {
		value, _, err := s.loadPendingAccess(ctx, q.Project, splitPolicyObject(q.Object)[1])
		if err != nil {
			return err
		}
		return s.pendingReadAuthority(ctx, actor, value)
	}
	return s.checkPolicyObject(ctx, q.Project, q.Object)
}

func (s *Service) ExplainPermission(ctx context.Context, actor, user string, q ExplanationQuery) (PermissionExplanation, error) {
	if q.Application != "" {
		if err := s.Auth.CheckApplication(ctx, actor, q.Project, q.Application, "", "projects.read"); err != nil {
			return PermissionExplanation{}, err
		}
		ctx = authorization.WithApplication(ctx, q.Project, q.Application)
	} else if q.Object != "" {
		contexts, err := s.Auth.PermissionContexts(ctx, q.Project, q.Object, q.Permission)
		if err != nil {
			return PermissionExplanation{}, err
		}
		if len(contexts) == 1 {
			ctx = contexts[0]
		}
	}
	if !domain.ValidID(user) || (q.Project != "" && !domain.ValidID(q.Project)) || !slices.Contains(authorization.PermissionNames(), q.Permission) || (q.Project != "" && authorization.InstallationPermission(q.Permission)) || (q.Object != "" && !validPolicyObject(q.Object)) {
		return PermissionExplanation{}, errors.New("a registered user, permission, and exact supported context are required")
	}
	if q.Limit < 1 || q.Limit > 1000 {
		return PermissionExplanation{}, errors.New("the explanation scan limit must be from 1 through 1000")
	}
	if err := s.explanationAuthority(ctx, actor, user, q); err != nil {
		return PermissionExplanation{}, err
	}
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return PermissionExplanation{}, err
	}
	u, err := s.Auth.Directory.User(ctx, user)
	if err != nil {
		return PermissionExplanation{}, err
	}
	result := PermissionExplanation{User: user, UserName: u.Name, UserDisabled: u.Disabled, Project: q.Project, Object: q.Object, Permission: q.Permission, CheckedAt: time.Now().UTC(), Items: []PermissionSource{}, Permissions: []authorization.PermissionDefinition{}}
	result.Application = authorization.ScopedApplication(ctx, q.Project)
	for _, definition := range authorization.PermissionDefinitions() {
		if q.Project == "" || !definition.InstallationOnly {
			result.Permissions = append(result.Permissions, definition)
		}
	}
	result.CanInspectOthers, err = roleCapability(func() error {
		for _, action := range []string{"members.read", "members.manage", "permissions.read"} {
			if err := s.Auth.Check(ctx, actor, q.Project, "", action); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	result.Reason = "user-disabled"
	if u.Disabled {
		if q.Cursor != "" {
			return result, storage.ErrConflict
		}
		if err := s.explanationAuthority(ctx, actor, user, q); err != nil {
			return result, err
		}
		latest, err := s.Auth.Directory.Revision(ctx)
		if err != nil {
			return result, err
		}
		if latest.Revision != version.Revision {
			return result, storage.ErrConflict
		}
		return result, nil
	}
	engine, err := s.Auth.CurrentEngine(ctx)
	if err != nil {
		return result, err
	}
	inventory, err := s.explanationInventory(ctx, engine, user)
	if err != nil {
		return result, err
	}
	allowed, err := previewDecision(ctx, s, user, q.Project, q.Object, q.Permission)
	if err != nil {
		return result, err
	}
	result.Allowed = allowed
	result.Reason = "allowed"
	if !allowed {
		denied, err := s.Auth.ExplainMatchingDeny(ctx, engine, authorization.RequestFor(user, q.Project, q.Object, q.Permission), inventory.Groups)
		if err != nil {
			return result, err
		}
		result.Reason = "no-matching-allow"
		if denied {
			result.Reason = "explicit-deny"
		}
	}
	stamp, err := explanationStamp(version.Revision, inventory, result)
	if err != nil {
		return result, err
	}
	scope := queryScope(actor, "effective-permissions", explanationContext(user, q), false)
	raw, err := s.openCursor(ctx, scope, q.Cursor)
	if err != nil {
		return result, err
	}
	cursor := explanationCursor{Stamp: stamp, Phase: "direct"}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &cursor); err != nil {
			return result, errors.New("invalid explanation cursor")
		}
		if cursor.Stamp != stamp {
			return result, storage.ErrConflict
		}
	}
	if cursor.ExpiresAt != nil && !time.Now().Before(*cursor.ExpiresAt) {
		return result, storage.ErrConflict
	}
	proofs, err := s.scanExplanation(ctx, engine, user, q, inventory, &cursor)
	if err != nil {
		return result, err
	}
	latest, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return result, err
	}
	if latest.Revision != version.Revision {
		return result, storage.ErrConflict
	}
	groups, err := s.Auth.Directory.GetUserGroups(ctx, user)
	if err != nil {
		return result, err
	}
	if !slices.Equal(groups, inventory.Groups) {
		return result, storage.ErrConflict
	}
	if err := s.explanationAuthority(ctx, actor, user, q); err != nil {
		return result, err
	}
	key, err := s.explanationKey(ctx)
	if err != nil {
		return result, err
	}
	for _, proof := range proofs {
		value, err := s.explanationSource(ctx, actor, user, q, inventory, proof, key)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, value)
	}
	if cursor.ExpiresAt != nil && !time.Now().Before(*cursor.ExpiresAt) {
		return result, storage.ErrConflict
	}
	if err := s.explanationAuthority(ctx, actor, user, q); err != nil {
		return result, err
	}
	latest, err = s.Auth.Directory.Revision(ctx)
	if err != nil {
		return result, err
	}
	if latest.Revision != version.Revision {
		return result, storage.ErrConflict
	}
	if cursor.Phase != "done" {
		encoded, err := json.Marshal(cursor)
		if err != nil {
			return result, err
		}
		result.Cursor, err = s.sealCursor(ctx, scope, string(encoded))
		if err != nil {
			return result, err
		}
		result.Partial = true
	}
	return result, nil
}

func splitPolicyObject(object string) []string {
	for i, c := range object {
		if c == '/' {
			return []string{object[:i], object[i+1:]}
		}
	}
	return []string{object, ""}
}
