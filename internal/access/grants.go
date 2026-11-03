package access

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type GrantDefinition struct {
	PrincipalKind permissions.PrincipalKind `json:"principalKind"`
	PrincipalID   string                    `json:"principalId"`
	Permission    string                    `json:"permission"`
	Effect        permissions.Effect        `json:"effect"`
	Object        string                    `json:"object,omitempty"`
	ExpiresAt     *time.Time                `json:"expiresAt,omitempty"`
	Reason        string                    `json:"reason"`
}

type GrantUpdate struct {
	ExpiresAt *time.Time `json:"expiresAt"`
	Reason    string     `json:"reason"`
}

type grantIdentity struct {
	Project       string                    `json:"projectId"`
	PrincipalKind permissions.PrincipalKind `json:"principalKind"`
	PrincipalID   string                    `json:"principalId"`
	Permission    string                    `json:"permission"`
	Effect        permissions.Effect        `json:"effect"`
	Object        string                    `json:"object,omitempty"`
}

type grantState struct {
	grantIdentity
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	NativeID  int64      `json:"nativeId"`
	Present   bool       `json:"present"`
}

type GrantView struct {
	grantIdentity
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	Expired         bool       `json:"expired"`
	PrincipalName   string     `json:"principalName"`
	PrincipalActive bool       `json:"principalActive"`
	CanUpdate       bool       `json:"canUpdate"`
	CanRemove       bool       `json:"canRemove"`
}

type GrantRecord struct {
	ID       string    `json:"id"`
	Project  string    `json:"projectId"`
	Revision int64     `json:"revision"`
	Updated  time.Time `json:"updatedAt"`
	Data     GrantView `json:"data"`
}

type GrantsPage struct {
	Items     []GrantRecord `json:"items"`
	Cursor    string        `json:"cursor,omitempty"`
	Partial   bool          `json:"partial"`
	CanCreate bool          `json:"canCreate"`
}

func grantBucket(project string) string { return "access-grants/" + project }

func (g grantIdentity) id() string {
	raw, _ := json.Marshal(g)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (g grantIdentity) native(expiry *time.Time) permissions.Grant {
	value := roleGrant(RolePolicy{ID: g.PrincipalID, Project: g.Project}, RoleRule{Permission: g.Permission, Effect: g.Effect, Object: g.Object, ExpiresAt: expiry})
	value.OwnerKind = g.PrincipalKind
	return value
}

func normalizeGrant(project string, input GrantDefinition) (grantState, error) {
	if (input.PrincipalKind != permissions.PrincipalUser && input.PrincipalKind != permissions.PrincipalGroup) || !domain.ValidID(input.PrincipalID) {
		return grantState{}, errors.New("a valid user or group principal is required")
	}
	if project != "" && !domain.ValidID(project) {
		return grantState{}, errors.New("a valid project identifier is required")
	}
	value, err := normalizePolicyRule(project, RoleRule{Permission: input.Permission, Effect: input.Effect, Object: input.Object, ExpiresAt: input.ExpiresAt})
	if err != nil {
		return grantState{}, err
	}
	return grantState{grantIdentity: grantIdentity{Project: project, PrincipalKind: input.PrincipalKind, PrincipalID: input.PrincipalID, Permission: input.Permission, Effect: input.Effect, Object: input.Object}, ExpiresAt: value.ExpiresAt, Present: true}, nil
}

func (s *Service) loadGrant(ctx context.Context, project, id string) (grantState, storage.Record, error) {
	if !domain.ValidID(id) {
		return grantState{}, storage.Record{}, errors.New("a valid grant identifier is required")
	}
	row, err := s.Store.Get(ctx, grantBucket(project), id)
	if err != nil {
		return grantState{}, row, err
	}
	value, err := storage.Decode[grantState](row)
	if err != nil {
		return value, row, err
	}
	if row.Project != project || value.Project != project || value.id() != id || value.NativeID < 1 {
		return value, row, errors.New("grant metadata does not match its stored identity")
	}
	return value, row, nil
}

func (s *Service) verifyNativeGrant(ctx context.Context, value grantState) error {
	native, err := s.Auth.ReadPolicyGrant(ctx, value.NativeID)
	if errors.Is(err, storage.ErrNotFound) {
		return storage.ErrConflict
	}
	if err != nil {
		return err
	}
	if roleGrantKey(native) != roleGrantKey(value.native(value.ExpiresAt)) {
		return storage.ErrConflict
	}
	return nil
}

func (s *Service) matchingGrant(ctx context.Context, value grantState) (permissions.Grant, error) {
	wanted := value.native(nil)
	query := permissions.GrantQuery{Principals: []permissions.PrincipalRef{{Kind: value.PrincipalKind, ID: value.PrincipalID}}, TeamScopes: []string{wanted.TeamScope}, Permissions: []string{value.Permission}, Limit: 100}
	var found permissions.Grant
	for {
		page, err := s.Auth.Engine.GetStore().ListGrants(ctx, query)
		if err != nil {
			return found, err
		}
		for _, grant := range page.Grants {
			identity := grant
			identity.ExpiresAt = nil
			if roleGrantKey(identity) == roleGrantKey(wanted) {
				if found.ID > 0 {
					return found, storage.ErrConflict
				}
				found = grant
			}
		}
		if page.NextCursor == "" {
			return found, nil
		}
		if page.NextCursor == query.Cursor {
			return found, errors.New("grant pagination did not advance")
		}
		query.Cursor = page.NextCursor
	}
}
