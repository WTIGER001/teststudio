package access

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

const pendingAccessBucket = "pending-project-access"

type PendingAccessInput struct {
	ProviderID       string `json:"providerId"`
	ProviderRevision int64  `json:"providerRevision"`
	Email            string `json:"email"`
	RoleID           string `json:"roleId"`
	Reason           string `json:"reason"`
}

type PendingAccessView struct {
	ID                   string     `json:"id"`
	Project              string     `json:"projectId"`
	ProviderID           string     `json:"providerId"`
	ProviderName         string     `json:"providerName"`
	ProviderRevision     int64      `json:"providerRevision"`
	Issuer               string     `json:"issuer"`
	Email                string     `json:"email"`
	RequireVerifiedEmail bool       `json:"requireVerifiedEmail"`
	RoleID               string     `json:"roleId"`
	RoleName             string     `json:"roleName"`
	ApprovedBy           string     `json:"approvedBy"`
	Reason               string     `json:"reason"`
	State                string     `json:"state"`
	CreatedAt            time.Time  `json:"createdAt"`
	ActivatedAt          *time.Time `json:"activatedAt,omitempty"`
	ActivatedUser        string     `json:"activatedUserId,omitempty"`
	AssignmentID         string     `json:"assignmentId,omitempty"`
	BlockReason          string     `json:"blockReason,omitempty"`
	CanCancel            bool       `json:"canCancel"`
}

type pendingAccessState struct {
	PendingAccessView
	RoleDigest string `json:"roleDigest"`
}

type PendingAccessRecord struct {
	ID       string            `json:"id"`
	Revision int64             `json:"revision"`
	Data     PendingAccessView `json:"data"`
}

type PendingAccessPage struct {
	Items   []PendingAccessRecord `json:"items"`
	Cursor  string                `json:"cursor,omitempty"`
	Partial bool                  `json:"partial"`
}

type PendingProviderChoice struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Issuer        string `json:"issuer"`
	Revision      int64  `json:"revision"`
	AutoProvision bool   `json:"autoProvision"`
}

type pendingAccessMatch struct {
	ID      string `json:"id"`
	Project string `json:"projectId"`
}

func pendingEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := mail.ParseAddress(value)
	if err != nil || len(value) > 254 || parsed.Name != "" || parsed.Address != value || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("one email address without a display name is required")
	}
	at := strings.LastIndexByte(value, '@')
	if at < 1 || at == len(value)-1 {
		return "", errors.New("one complete email address is required")
	}
	return value[:at+1] + strings.ToLower(value[at+1:]), nil
}

func pendingDigest(values ...string) string {
	data, _ := json.Marshal(values)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func pendingMatchBucket(provider, email string) string {
	return "pending-access-matches/" + pendingDigest(provider, email)
}
func pendingMatchID(value pendingAccessState) string {
	return pendingDigest(value.Project, value.RoleID)
}

func (s *Service) pendingAuthority(ctx context.Context, actor, project, id, role string, manage bool) (storage.Record, error) {
	if project == "" {
		return storage.Record{}, errors.New("pending access requires an explicit project")
	}
	membership, grant := "members.read", "permissions.read"
	if manage {
		membership, grant = "members.manage", "permissions.grant"
	}
	row, err := s.assignmentScope(ctx, actor, project, membership)
	if err != nil {
		return row, err
	}
	for _, object := range []string{"", "pending-access/" + id, "role/" + role} {
		if err := s.Auth.Check(ctx, actor, project, object, membership); err != nil {
			return row, err
		}
		if err := s.Auth.Check(ctx, actor, project, object, grant); err != nil {
			return row, err
		}
	}

	return row, nil
}

func (s *Service) pendingRoleDigest(ctx context.Context, project, id string) (string, string, error) {
	role, err := s.roleForScope(ctx, project, id)
	if err != nil {
		return "", "", err
	}
	if role.IsDisabled {
		return "", "", errors.New("enable the disabled role before adding pending access")
	}
	versions := map[string]int64{}
	if definition, ok := builtinRole(project, id); ok {
		row, err := s.Store.Get(ctx, "access-builtin-roles", id)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return "", "", err
		}
		versions[id] = row.Revision
		raw, _ := json.Marshal(definition)
		return pendingDigest(string(raw), pendingVersions(versions)), role.Name, nil
	}
	visited := map[string]bool{}
	var visit func(string, int) error
	visit = func(current string, depth int) error {
		if depth > 16 || len(visited) > 256 {
			return errors.New("role inheritance exceeds its bound")
		}
		if visited[current] {
			return nil
		}
		policy, row, err := s.loadRolePolicy(ctx, current)
		if err != nil {
			return err
		}
		if policy.Project != project {
			return storage.ErrNotFound
		}
		visited[current] = true
		versions[current] = row.Revision
		for _, child := range policy.Inherits {
			if err := visit(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(id, 0); err != nil {
		return "", "", err
	}
	return pendingDigest(pendingVersions(versions)), role.Name, nil
}

func pendingVersions(versions map[string]int64) string {
	raw, _ := json.Marshal(versions)
	return string(raw)
}

func (s *Service) loadPendingAccess(ctx context.Context, project, id string) (pendingAccessState, storage.Record, error) {
	row, err := s.Store.Get(ctx, pendingAccessBucket, id)
	if err != nil {
		return pendingAccessState{}, row, err
	}
	value, err := storage.Decode[pendingAccessState](row)
	if err != nil {
		return value, row, err
	}
	if !domain.ValidID(project) || row.Project != project || value.Project != project || value.ID != id || !value.RequireVerifiedEmail {
		return pendingAccessState{}, storage.Record{}, storage.ErrNotFound
	}
	return value, row, nil
}

func (s *Service) pendingRecord(ctx context.Context, actor string, value pendingAccessState, row storage.Record) (PendingAccessRecord, error) {
	value.CanCancel = false
	if value.State == "pending" || value.State == "blocked" {
		_, err := s.pendingAuthority(ctx, actor, value.Project, value.ID, value.RoleID, true)
		if err != nil && !errors.Is(err, authorization.ErrForbidden) && !errors.Is(err, storage.ErrNotFound) {
			return PendingAccessRecord{}, err
		}
		value.CanCancel = err == nil
	}
	return PendingAccessRecord{ID: row.ID, Revision: row.Revision, Data: value.PendingAccessView}, nil
}

func (s *Service) pendingReadAuthority(ctx context.Context, actor string, value pendingAccessState) error {
	if _, err := s.pendingAuthority(ctx, actor, value.Project, value.ID, value.RoleID, false); err != nil {
		return err
	}
	if value.ActivatedUser != "" {
		return s.assignmentAuthority(ctx, actor, value.Project, AssignmentDefinition{PrincipalKind: permissions.PrincipalUser, PrincipalID: value.ActivatedUser, RoleID: value.RoleID}, false)
	}
	return nil
}
