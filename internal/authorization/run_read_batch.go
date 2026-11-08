package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type readStore struct {
	permissions.PermissionStore
	assignments map[permissions.PrincipalRef][]permissions.RoleAssignment
	grants      map[permissions.PrincipalRef][]permissions.Grant
	expansion   map[string][]string
}

func (s *readStore) RoleAssignmentsForPrincipal(ctx context.Context, p permissions.PrincipalRef) ([]permissions.RoleAssignment, error) {
	if value, ok := s.assignments[p]; ok {
		return value, nil
	}
	value, err := s.PermissionStore.RoleAssignmentsForPrincipal(ctx, p)
	if err == nil {
		s.assignments[p] = value
	}
	return value, err
}
func (s *readStore) GrantsForOwners(ctx context.Context, owners []permissions.PrincipalRef, _ permissions.Request) ([]permissions.Grant, error) {
	result := []permissions.Grant{}
	for _, owner := range owners {
		value, ok := s.grants[owner]
		if !ok {
			var err error
			value, err = s.PermissionStore.GrantsForPrincipal(ctx, owner)
			if err != nil {
				return nil, err
			}
			s.grants[owner] = value
		}
		result = append(result, value...)
	}
	return result, nil
}
func (s *readStore) ExpandRoles(ctx context.Context, ids []string) ([]string, error) {
	key := strings.Join(ids, "\x00")
	if value, ok := s.expansion[key]; ok {
		return value, nil
	}
	value, err := s.PermissionStore.ExpandRoles(ctx, ids)
	if err == nil {
		s.expansion[key] = value
	}
	return value, err
}

type readIdentity struct {
	permissions.IdentityProvider
	user          string
	groups, teams []string
}

func (s *readIdentity) GetUserGroups(ctx context.Context, user string) ([]string, error) {
	if user == s.user {
		return s.groups, nil
	}
	return s.IdentityProvider.GetUserGroups(ctx, user)
}
func (s *readIdentity) GetUserTeams(ctx context.Context, user string) ([]string, error) {
	if user == s.user {
		return s.teams, nil
	}
	return s.IdentityProvider.GetUserTeams(ctx, user)
}

type RunReadBatch struct {
	auth    *Service
	engine  *permissions.Service
	user    string
	version int64
	memo    *readStore
	uniform map[string]runReadDecision
}

type runReadDecision struct {
	allowed bool
	expires time.Time
	owners  int
}

func (s *Service) NewRunReadBatch(ctx context.Context, user string) (*RunReadBatch, error) {
	version, err := s.Directory.Revision(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := s.Directory.User(ctx, user)
	if err != nil {
		return nil, err
	}
	if profile.Disabled {
		return nil, ErrForbidden
	}
	groups, err := s.Directory.GetUserGroups(ctx, user)
	if err != nil {
		return nil, err
	}
	teams, err := s.Directory.GetUserTeams(ctx, user)
	if err != nil {
		return nil, err
	}
	disabled, err := s.Engine.DisabledBuiltInRoles(ctx)
	if err != nil {
		return nil, err
	}
	memo := &readStore{PermissionStore: s.Engine.GetStore(), assignments: map[permissions.PrincipalRef][]permissions.RoleAssignment{}, grants: map[permissions.PrincipalRef][]permissions.Grant{}, expansion: map[string][]string{}}
	engine, err := NewEngine(ctx, &builtInStateStore{PermissionStore: &applicationStore{PermissionStore: memo}, disabled: disabled}, &readIdentity{IdentityProvider: s.Directory, user: user, groups: groups, teams: teams})
	if err != nil {
		return nil, err
	}
	return &RunReadBatch{auth: s, engine: engine, user: user, version: version.Revision, memo: memo, uniform: map[string]runReadDecision{}}, nil
}
func (b *RunReadBatch) Check(ctx context.Context, project string, row storage.Record) error {
	if row.Bucket != "runs" || row.Project != project {
		return storage.ErrNotFound
	}
	var data scopedResource
	if err := json.Unmarshal(row.Data, &data); err != nil {
		return err
	}
	application := ""
	apps := runApplications(data)
	if len(apps) == 1 {
		application = apps[0]
	}
	key := project + "\x00" + application
	if decision, ok := b.uniform[key]; ok && decision.owners == len(b.memo.grants) && (decision.expires.IsZero() || time.Now().Before(decision.expires)) {
		if decision.allowed {
			return nil
		}
		return ErrForbidden
	}
	allowed, err := b.engine.HasPermission(WithApplication(ctx, project, application), RequestFor(b.user, project, "run/"+row.ID, "runs.read"))
	if err != nil {
		return err
	}
	independent := true
	var expiry time.Time
	for _, grants := range b.memo.grants {
		for _, grant := range grants {
			if grant.ObjectScope != nil && *grant.ObjectScope != "*" {
				independent = false
			}
			if grant.ExpiresAt != nil && time.Now().Before(*grant.ExpiresAt) && (expiry.IsZero() || grant.ExpiresAt.Before(expiry)) {
				expiry = *grant.ExpiresAt
			}
		}
	}
	if independent {
		b.uniform[key] = runReadDecision{allowed: allowed, expires: expiry, owners: len(b.memo.grants)}
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}
func (b *RunReadBatch) Current(ctx context.Context) error {
	version, err := b.auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	if version.Revision != b.version {
		return errors.New("access changed during status calculation; refresh before retrying")
	}
	return nil
}
func runApplications(data scopedResource) []string {
	app := data.Request.Application
	if data.Environment != nil {
		if app != "" && app != data.Environment.Application {
			return nil
		}
		app = data.Environment.Application
	}
	if app == "" && len(data.ApplicationIDs) == 1 {
		app = data.ApplicationIDs[0]
	}
	if app == "" {
		return nil
	}
	for _, test := range data.Tests {
		if test.Application != "" && test.Application != app {
			return nil
		}
	}
	for _, test := range data.Release.Catalog.Tests {
		if test.Application != "" && test.Application != app {
			return nil
		}
	}
	return []string{app}
}

func CompactRunReadSource(data json.RawMessage) (json.RawMessage, error) {
	var resource scopedResource
	if err := json.Unmarshal(data, &resource); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Applications []string `json:"applicationIds"`
	}{runApplications(resource)})
}

func (b *RunReadBatch) UniformSource(ctx context.Context, project string, source json.RawMessage) (bool, error) {
	err := b.Check(ctx, project, storage.Record{Bucket: "runs", ID: "reliability-scope", Project: project, Data: source})
	if err != nil && !errors.Is(err, ErrForbidden) {
		return false, err
	}
	var data scopedResource
	if err := json.Unmarshal(source, &data); err != nil {
		return false, err
	}
	app := ""
	if apps := runApplications(data); len(apps) == 1 {
		app = apps[0]
	}
	decision, ok := b.uniform[project+"\x00"+app]
	return ok && decision.owners == len(b.memo.grants) && (decision.expires.IsZero() || time.Now().Before(decision.expires)), nil
}
