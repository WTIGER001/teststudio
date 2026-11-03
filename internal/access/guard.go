package access

import (
	"context"
	"errors"
	"slices"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/storage"
)

type directoryView struct {
	localAccounts bool
	local         map[string]bool
	users         map[string]identity.User
	groups        map[string]identity.Group
	memberships   map[string]identity.Membership
}

func (v *directoryView) GetUserGroups(_ context.Context, user string) ([]string, error) {
	if v.users[user].Disabled {
		return []string{}, nil
	}
	groups := []string{}
	for _, m := range v.memberships {
		g, ok := v.groups[m.GroupID]
		if m.UserID == user && ok && !g.Archived && (m.ExpiresAt == nil || time.Now().Before(*m.ExpiresAt)) && !slices.Contains(groups, g.ID) {
			groups = append(groups, g.ID)
		}
	}
	slices.Sort(groups)
	return groups, nil
}
func (v *directoryView) GetGroupMembers(ctx context.Context, group string) ([]string, error) {
	users := []string{}
	for id := range v.users {
		groups, _ := v.GetUserGroups(ctx, id)
		if slices.Contains(groups, group) {
			users = append(users, id)
		}
	}
	return users, nil
}
func (v *directoryView) IsUserInGroup(ctx context.Context, user, group string) (bool, error) {
	groups, err := v.GetUserGroups(ctx, user)
	return slices.Contains(groups, group), err
}
func (*directoryView) GetUserTeams(context.Context, string) ([]string, error)     { return []string{}, nil }
func (*directoryView) IsUserInTeam(context.Context, string, string) (bool, error) { return false, nil }

func (s *Service) hasUsableAdministrator(ctx context.Context) (bool, error) {
	engine, view, logins, err := s.administratorState(ctx)
	if err != nil {
		return false, err
	}
	return usableAdministrator(ctx, engine, view, logins)
}

func (s *Service) administratorState(ctx context.Context) (*permissions.Service, *directoryView, map[string]bool, error) {
	v := &directoryView{localAccounts: s.LocalAccounts, local: map[string]bool{}, users: map[string]identity.User{}, groups: map[string]identity.Group{}, memberships: map[string]identity.Membership{}}
	if v.localAccounts {
		mode, err := s.Store.Get(ctx, "local-auth-mode", "installation")
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, nil, nil, err
		}
		state := identity.LocalAuthMode{}
		if mode.Revision > 0 {
			state, err = storage.Decode[identity.LocalAuthMode](mode)
			if err != nil {
				return nil, nil, nil, err
			}
		}
		v.localAccounts = state.Enabled
	}
	for _, bucket := range []string{"users", "groups", "memberships", "local-credentials"} {
		rows, err := s.Store.All(ctx, bucket, "")
		if err != nil {
			return nil, nil, nil, err
		}
		for _, row := range rows {
			if err := v.apply(row, false); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	engine, err := authorization.NewEngine(ctx, s.Auth.Engine.GetStore(), v)
	if err != nil {
		return nil, nil, nil, err
	}
	logins := map[string]bool{}
	if s.LocalUser != "" {
		logins[s.LocalUser] = true
	}
	providers, err := s.Store.All(ctx, "providers", "")
	if err != nil {
		return nil, nil, nil, err
	}
	issuers := map[string]bool{}
	for _, row := range providers {
		p, err := storage.Decode[struct {
			Issuer   string `json:"issuer"`
			ClientID string `json:"clientId"`
			Disabled bool   `json:"disabled"`
		}](row)
		if err != nil {
			return nil, nil, nil, err
		}
		if !p.Disabled && p.Issuer != "" && p.ClientID != "" {
			issuers[p.Issuer] = true
		}
	}
	links, err := s.Store.All(ctx, "external-identities", "")
	if err != nil {
		return nil, nil, nil, err
	}
	for _, row := range links {
		link, err := storage.Decode[identity.ExternalIdentity](row)
		if err != nil {
			return nil, nil, nil, err
		}
		if issuers[link.Issuer] && link.Subject != "" {
			logins[link.UserID] = true
		}
	}
	return engine, v, logins, nil
}

func (s *Service) preserveAdministrator(ctx context.Context, changes ...storage.Change) error {
	engine, v, logins, err := s.administratorState(ctx)
	if err != nil {
		return err
	}
	before, err := usableAdministrator(ctx, engine, v, logins)
	if err != nil || !before {
		return err
	}
	for _, change := range changes {
		if err := v.apply(change.Record, change.Delete); err != nil {
			return err
		}
	}
	after, err := usableAdministrator(ctx, engine, v, logins)
	if err != nil {
		return err
	}
	if after {
		return nil
	}
	return errors.New("the change would remove the last usable installation administrator")
}

func usableAdministrator(ctx context.Context, engine *permissions.Service, v *directoryView, logins map[string]bool) (bool, error) {
	for id, user := range v.users {
		if user.Disabled || (!logins[id] && !v.local[id]) {
			continue
		}
		usable := true
		for _, permission := range []string{"teststudio.system.manage", "teststudio.users.manage", "teststudio.permissions.grant"} {
			ok, err := engine.HasPermission(ctx, permissions.Request{UserID: id, Perm: permission})
			if err != nil {
				return false, err
			}
			if !ok {
				usable = false
				break
			}
		}
		if usable {
			return true, nil
		}
	}
	return false, nil
}

func (v *directoryView) apply(row storage.Record, remove bool) error {
	switch row.Bucket {
	case "local-credentials":
		if !v.localAccounts {
			return nil
		}
		c, err := storage.Decode[identity.LocalCredential](row)
		if err != nil {
			return err
		}
		v.local[row.ID] = !remove && !c.Disabled && c.UserID == row.ID && passwords.ValidHash(c.Hash)

	case "users":
		value, err := storage.Decode[identity.User](row)
		if err != nil {
			return err
		}
		v.users[row.ID] = value
	case "groups":
		value, err := storage.Decode[identity.Group](row)
		if err != nil {
			return err
		}
		v.groups[row.ID] = value
	case "memberships":
		if remove {
			delete(v.memberships, row.ID)
			return nil
		}
		value, err := storage.Decode[identity.Membership](row)
		if err != nil {
			return err
		}
		v.memberships[row.ID] = value
	}
	return nil
}
