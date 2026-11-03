package access

import (
	"context"
	"errors"
	"strings"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func validPolicyObject(object string) bool {
	parts := strings.Split(object, "/")
	if len(parts) != 2 && len(parts) != 3 {
		return false
	}
	if len(parts) == 3 && parts[0] != "test" {
		return false
	}
	if len(parts) == 2 && !strings.Contains("|suite|test|run|environment|application|source|group|user|role|grant|provider|pending-access|", "|"+parts[0]+"|") {
		return false
	}
	if parts[0] == "test" && len(parts) != 3 {
		return false
	}
	for _, id := range parts[1:] {
		if !domain.ValidID(id) {
			return false
		}
	}
	return true
}

func (s *Service) checkPolicyObject(ctx context.Context, project, object string) error {
	if object == "" {
		return nil
	}
	if !validPolicyObject(object) {
		return errors.New("an exact supported resource identifier is required")
	}
	parts := strings.Split(object, "/")
	kind, id := parts[0], parts[1]
	var row storage.Record
	var err error
	switch kind {
	case "pending-access":
		if project == "" {
			return errors.New("pending access requires project scope")
		}
		_, row, err = s.loadPendingAccess(ctx, project, id)
	case "provider":
		if project != "" {
			return errors.New("provider objects require installation scope")
		}
		row, err = s.Store.Get(ctx, "providers", id)
	case "grant":
		value, stored, loadErr := s.loadGrant(ctx, project, id)
		if loadErr != nil {
			return loadErr
		}
		if !value.Present {
			return storage.ErrNotFound
		}
		row = stored
	case "role":
		if strings.HasPrefix(id, "builtin.") {
			if project == "" {
				_, err := predefinedRole(id)
				return err
			}
			if _, ok := builtinRole(project, id); !ok {
				return storage.ErrNotFound
			}
			return nil
		}
		_, row, err = s.loadRolePolicy(ctx, id)
	case "group":
		row, err = s.Store.Get(ctx, "groups", id)
	case "user":
		if project != "" {
			return errors.New("user directory objects require installation scope")
		}
		row, err = s.Store.Get(ctx, "users", id)
	case "run":
		row, err = s.Store.Get(ctx, "runs", id)
	case "source":
		if project == "" {
			return errors.New("repository sources require an explicit project scope")
		}
		row, err = s.Store.Get(ctx, "repository-sources/"+project, id)
	case "suite", "environment", "application":
		if project == "" {
			return errors.New("project resources require an explicit project scope")
		}
		row, err = s.Store.Get(ctx, projects.Bucket(kind+"s", project), id)
	case "test":
		if project == "" {
			return errors.New("test resources require an explicit project scope")
		}
		row, err = s.Store.Get(ctx, catalog.Bucket("tests", project, id), parts[2])
	default:
		return errors.New("unsupported rule object")
	}
	if err != nil {
		return err
	}
	if row.Project != project {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Service) roleRulesAuthority(ctx context.Context, actor string, value RolePolicy, checkResources bool) error {
	for _, rule := range value.Rules {
		if checkResources {
			if err := s.checkPolicyObject(ctx, value.Project, rule.Object); err != nil {
				return err
			}
		}
		for _, action := range []string{"permissions.grant", rule.Permission} {
			if err := s.Auth.Check(ctx, actor, value.Project, rule.Object, action); err != nil {
				return err
			}
		}
	}
	return nil
}
