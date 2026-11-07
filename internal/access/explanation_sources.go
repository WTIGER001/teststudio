package access

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) explanationKey(ctx context.Context) ([]byte, error) {
	if _, err := s.cursorCipher(ctx); err != nil {
		return nil, err
	}
	row, err := s.Store.Get(ctx, "access-settings", "cursor-key")
	if err != nil {
		return nil, err
	}
	key, err := storage.Decode[[]byte](row)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("permission source key is unavailable")
	}
	return key, nil
}

func explanationID(key []byte, actor, user string, q ExplanationQuery, proof explanationProof) (string, error) {
	value := struct {
		Context string
		Actor   string
		Proof   explanationProof
	}{explanationContext(user, q), actor, proof}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("permission-source\x00"))
	mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) explanationRoleScope(ctx context.Context, id string, path explanationRolePath) (string, bool, error) {
	if strings.HasPrefix(id, "builtin.") {
		if _, err := predefinedRole(id); err != nil {
			return "", false, err
		}
		if id == "builtin.teststudio.installation-admin" {
			return "", true, nil
		}
		project, ok := path.Assignment.BindingValues["team"].(string)
		return project, ok && domain.ValidID(project), nil
	}
	value, _, err := s.loadRolePolicy(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value.Project, true, nil
}

func (s *Service) explanationSource(ctx context.Context, actor, user string, q ExplanationQuery, inventory explanationInventory, proof explanationProof, key []byte) (PermissionSource, error) {
	value := PermissionSource{Kind: "direct", Effect: proof.Grant.Effect, Status: proof.Status, Restricted: true}
	var err error
	value.ID, err = explanationID(key, actor, user, q, proof)
	if err != nil {
		return value, err
	}
	principal := permissions.PrincipalRef{Kind: proof.Grant.OwnerKind, ID: proof.Grant.OwnerID}
	sourceProject := proof.Grant.TeamScope
	scopeKnown := sourceProject == "*" || domain.ValidID(sourceProject)
	if sourceProject == "*" {
		sourceProject = ""
	}
	checks := func(project, object string, names ...string) error {
		for _, name := range names {
			if err := s.Auth.Check(ctx, actor, project, object, name); err != nil {
				return err
			}
		}
		return nil
	}
	var managed storage.Record
	if proof.Path == nil {
		object := ""
		if proof.Grant.ObjectScope != nil {
			object = *proof.Grant.ObjectScope
			if sourceProject != "" {
				object = strings.TrimPrefix(object, sourceProject+"/")
			}
		}
		identity := grantIdentity{Project: sourceProject, PrincipalKind: principal.Kind, PrincipalID: principal.ID, Permission: proof.Grant.PermissionName, Effect: proof.Grant.Effect, Object: object}
		state, row, err := s.loadGrant(ctx, sourceProject, identity.id())
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return value, err
		}
		if err == nil && state.Present && state.NativeID == proof.Grant.ID {
			if err := s.verifyNativeGrant(ctx, state); err != nil {
				return value, err
			}
			managed = row
		}
	}
	visible, err := roleCapability(func() error {
		if proof.Path != nil {
			value.Kind = "role"
			scopeKnown = true
			principal = proof.Path.Principal
			for _, id := range []string{proof.Path.Assignment.RoleID, proof.Path.Role} {
				project, known, err := s.explanationRoleScope(ctx, id, *proof.Path)
				if err != nil {
					return err
				}
				if !known {
					project = ""
					scopeKnown = false
				}
				if err := checks(project, "", "permissions.read"); err != nil {
					return err
				}
				if err := checks(project, "role/"+id, "roles.read"); err != nil {
					return err
				}
				if err := checks(project, "role/"+id, "permissions.read", "members.read"); err != nil {
					return err
				}
				if id == proof.Path.Assignment.RoleID {
					if err := checks(project, string(principal.Kind)+"/"+principal.ID, "permissions.read", "members.read"); err != nil {
						return err
					}
				}
				sourceProject = project
			}
			if proof.Grant.TeamScope == "*" {
				if err := checks("", "", "permissions.read"); err != nil {
					return err
				}
			}
			if project, ok := proof.Path.Assignment.BindingValues["team"].(string); ok && domain.ValidID(project) {
				if err := checks(project, string(principal.Kind)+"/"+principal.ID, "permissions.read", "members.read"); err != nil {
					return err
				}
			}
		} else if !(actor == user && principal.Kind == permissions.PrincipalUser) {
			if err := checks(sourceProject, "", "permissions.read"); err != nil {
				return err
			}
		}
		if managed.Revision > 0 {
			if err := checks(sourceProject, "grant/"+managed.ID, "permissions.read", "members.read"); err != nil {
				return err
			}
		}
		if principal.Kind == permissions.PrincipalGroup {
			if err := checks(sourceProject, "group/"+principal.ID, "permissions.read", "members.read"); err != nil {
				return err
			}
		}
		if principal.Kind == permissions.PrincipalGroup {
			if _, _, err := s.group(ctx, actor, principal.ID, "groups.read"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return value, err
	}
	if !visible {
		return value, nil
	}
	value.Restricted = false
	if managed.Revision > 0 {
		value.GrantID = managed.ID
	}
	value.PrincipalKind = principal.Kind
	value.PrincipalID = principal.ID
	value.Project = sourceProject
	value.ScopeKnown = scopeKnown
	value.TeamScope = proof.Grant.TeamScope
	value.ExpiresAt = proof.Grant.ExpiresAt
	value.RestrictedFields = proof.Grant.RestrictedFields
	if proof.Grant.ObjectScope != nil {
		value.ObjectScope = *proof.Grant.ObjectScope
	}
	if principal.Kind == permissions.PrincipalUser {
		u, err := s.Auth.Directory.User(ctx, principal.ID)
		if err != nil {
			return value, err
		}
		value.PrincipalName = u.Name
	} else {
		g, err := s.Auth.Directory.Group(ctx, principal.ID)
		if err != nil {
			return value, err
		}
		value.PrincipalName = g.Name
	}
	if proof.Path != nil {
		value.RoleID = proof.Path.Role
		value.RoleName = inventory.Roles[proof.Path.Role].Name
		value.AssignedRoleID = proof.Path.Assignment.RoleID
		value.AssignedRoleName = inventory.Roles[value.AssignedRoleID].Name
		if project, ok := proof.Path.Assignment.BindingValues["team"].(string); ok && domain.ValidID(project) {
			value.AssignmentProject = project
		}
		value.AssignmentApplication, _ = proof.Path.Assignment.BindingValues["application"].(string)
	}
	return value, nil
}
