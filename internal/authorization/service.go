package authorization

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/identity"
)

var ErrForbidden = errors.New("permission denied")

type Service struct {
	Engine            *permissions.Service
	Directory         *identity.Directory
	policyTransaction bool
	evaluationMu      sync.Mutex
	evaluationOrigin  *permissions.Service
	evaluationState   string
	evaluationEngine  *permissions.Service
}

type RoleDefinition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func PermissionNames() []string {
	areas := map[string][]string{
		"catalog":      {"read", "create", "update", "import", "export", "publish", "archive", "restore", "delete"},
		"runs":         {"read", "create", "cancel", "execute-protected"},
		"evidence":     {"read", "download", "retain", "delete", "override-capture"},
		"schedules":    {"read", "create", "update", "archive", "delete"},
		"applications": {"read", "manage"},
		"statuspages":  {"read", "manage", "publish", "delete"},
		"policies":     {"manage"},
		"environments": {"read", "manage"}, "sources": {"read", "manage"}, "suites": {"read", "manage"},
		"credentials": {"read", "manage"}, "notifications": {"read", "manage"}, "projects": {"read", "create", "manage"},
		"members": {"read", "manage", "owners"}, "roles": {"read", "manage"}, "permissions": {"read", "grant"},
		"users": {"read", "manage"}, "groups": {"read", "manage"}, "providers": {"read", "manage"},
		"workers": {"read", "manage"}, "system": {"read", "manage"},
	}
	result := []string{}
	for area, actions := range areas {
		for _, action := range actions {
			result = append(result, "teststudio."+area+"."+action)
		}
	}
	slices.Sort(result)
	return result
}

func BuiltInRoles() []RoleDefinition {
	read := []string{"catalog.read", "runs.read", "evidence.read", "evidence.download", "schedules.read", "environments.read", "sources.read", "suites.read", "projects.read"}
	with := func(extra ...string) []string { return append(slices.Clone(read), extra...) }
	roles := []RoleDefinition{
		{ID: "builtin.teststudio.viewer", Name: "Viewer", Permissions: read},
		{ID: "builtin.teststudio.operator", Name: "Operator", Permissions: with("runs.create", "runs.cancel", "schedules.create", "schedules.update", "schedules.delete")},
		{ID: "builtin.teststudio.editor", Name: "Catalog editor", Permissions: with("catalog.create", "catalog.update", "catalog.import", "catalog.export", "catalog.archive", "catalog.restore")},
		{ID: "builtin.teststudio.publisher", Name: "Publisher", Permissions: with("catalog.publish", "sources.manage", "suites.manage")},
		{ID: "builtin.teststudio.project-admin", Name: "Project administrator", Permissions: with("projects.manage", "members.read", "members.manage", "groups.read", "groups.manage", "roles.read", "permissions.read", "permissions.grant", "environments.manage", "credentials.read", "credentials.manage", "notifications.read", "notifications.manage")},
	}
	roles = append(roles, standardRoles()...)
	for i := range roles {
		for j := range roles[i].Permissions {
			roles[i].Permissions[j] = "teststudio." + roles[i].Permissions[j]
		}
	}
	roles = append(roles, RoleDefinition{ID: "builtin.teststudio.installation-admin", Name: "System Admin", Permissions: PermissionNames()})
	return roles
}

func New(ctx context.Context, store permissions.PermissionStore, directory *identity.Directory) (*Service, error) {
	engine, err := NewEngine(ctx, store, directory)
	if err != nil {
		return nil, err
	}
	return &Service{Engine: engine, Directory: directory}, nil
}

func NewEngine(ctx context.Context, store permissions.PermissionStore, directory permissions.IdentityProvider) (*permissions.Service, error) {
	engine, err := roleEngine(ctx, store, directory)
	if err != nil {
		return nil, err
	}
	for _, role := range BuiltInRoles() {
		for _, name := range role.Permissions {
			g := permissions.Grant{OwnerKind: permissions.PrincipalRole, OwnerID: role.ID, PermissionName: name, Effect: permissions.EffectAllow, TeamScope: "?team", VariableSpec: map[string]any{"team": "required"}}
			if role.ID == "builtin.teststudio.installation-admin" {
				g.TeamScope = "*"
				g.VariableSpec = nil
			}
			if err := engine.CreateGrant(ctx, g); err != nil {
				return nil, err
			}
		}
	}
	return engine, nil
}

func roleEngine(ctx context.Context, store permissions.PermissionStore, directory permissions.IdentityProvider) (*permissions.Service, error) {
	engine := permissions.NewServiceWithIdentity(directory, store)
	engine.SetSyntheticRoleIDs("", "", "")
	for _, role := range BuiltInRoles() {
		if err := engine.AddBuiltInRole(ctx, permissions.Role{ID: role.ID, Name: role.Name}); err != nil {
			return nil, err
		}
	}
	return engine, nil
}

func RequestFor(user, project, object, name string) permissions.Request {
	if !strings.HasPrefix(name, "teststudio.") {
		name = "teststudio." + name
	}
	if project != "" && object != "" {
		object = project + "/" + object
	}
	return permissions.Request{UserID: user, TeamID: project, Object: object, Perm: name}
}

func (s *Service) Check(ctx context.Context, user, project, object, name string) error {
	if user == "" {
		return ErrForbidden
	}
	u, err := s.Directory.User(ctx, user)
	if err != nil {
		return err
	}
	if u.Disabled {
		return ErrForbidden
	}
	contexts, err := s.PermissionContexts(ctx, project, object, name)
	if err != nil {
		return err
	}
	engine, err := s.CurrentEngine(ctx)
	if err != nil {
		return err
	}
	for _, scope := range contexts {
		ok, err := engine.HasPermission(scope, RequestFor(user, project, object, name))
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return ErrForbidden
}

func (s *Service) Capabilities(ctx context.Context, user, project string) (map[string]bool, error) {
	result := map[string]bool{}
	for _, name := range PermissionNames() {
		err := s.Check(ctx, user, project, "", name)
		if err != nil && !errors.Is(err, ErrForbidden) {
			return nil, err
		}
		result[name] = err == nil
	}
	return result, nil
}
