package access

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type SessionView struct {
	ID                  string    `json:"id"`
	Provider            string    `json:"provider"`
	CreatedAt           time.Time `json:"createdAt"`
	LastSeen            time.Time `json:"lastSeen"`
	ExpiresAt           time.Time `json:"expiresAt"`
	AuthenticationUntil time.Time `json:"authenticationUntil"`
	Current             bool      `json:"current"`
}

type Profile struct {
	User         identity.User `json:"user"`
	Identities   Page          `json:"identities"`
	LocalAccount bool          `json:"localAccount"`
}

func (s *Service) CurrentProfile(ctx context.Context, user, cursor string, limit int) (Profile, error) {
	u, err := s.Auth.Directory.User(ctx, user)
	if err != nil {
		return Profile{}, err
	}
	if u.Disabled {
		return Profile{}, authorization.ErrForbidden
	}
	identities, err := s.identities(ctx, user, user, cursor, limit)
	return Profile{User: u, Identities: identities, LocalAccount: user == s.LocalUser}, err
}

func (s *Service) UserIdentities(ctx context.Context, actor, user, cursor string, limit int) (Page, error) {
	if _, err := s.ReadUser(ctx, actor, user); err != nil {
		return Page{}, err
	}
	return s.identities(ctx, actor, user, cursor, limit)
}

func (s *Service) identities(ctx context.Context, actor, user, cursor string, limit int) (Page, error) {
	page, err := s.page(ctx, "external-identities", "", queryScope(actor, "identities", user, false), cursor, limit)
	if err != nil {
		return Page{}, err
	}
	result := Page{Items: []storage.Record{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	for _, row := range page.Items {
		link, err := storage.Decode[identity.ExternalIdentity](row)
		if err != nil {
			return result, err
		}
		if link.UserID == user {
			result.Items = append(result.Items, row)
		}
	}
	return result, nil
}

func (s *Service) UserMemberships(ctx context.Context, actor, user, cursor string, limit int) (Page, error) {
	r, err := s.ReadUser(ctx, actor, user)
	if err != nil {
		return Page{}, err
	}
	u, err := storage.Decode[identity.User](r)
	if err != nil {
		return Page{}, err
	}
	page, err := s.page(ctx, "memberships", "", queryScope(actor, "user-memberships", user, false), cursor, limit)
	if err != nil {
		return Page{}, err
	}
	result := Page{Items: []storage.Record{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	for _, row := range page.Items {
		m, err := storage.Decode[identity.Membership](row)
		if err != nil {
			return result, err
		}
		if m.UserID != user {
			continue
		}
		g, _, err := s.group(ctx, actor, m.GroupID, "groups.read")
		if errors.Is(err, authorization.ErrForbidden) {
			continue
		}
		if err != nil {
			return result, err
		}
		if g.Project != row.Project {
			continue
		}
		if g.Project != "" {
			err := s.Auth.Check(ctx, actor, g.Project, "group/"+g.ID, "members.read")
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			if err != nil {
				return result, err
			}
		}
		view := struct {
			MembershipView
			GroupName string `json:"groupName"`
			Project   string `json:"projectId"`
		}{MembershipView{Membership: m, Name: u.Name, Active: !u.Disabled && !g.Archived && (m.ExpiresAt == nil || time.Now().Before(*m.ExpiresAt)), ReadOnly: g.ManagedBy != "" || m.Source != "local"}, g.Name, g.Project}
		row.Data, err = json.Marshal(view)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, row)
	}
	return result, nil
}

func (s *Service) UserSessions(ctx context.Context, actor, user, current, cursor string, limit int) (Page, error) {
	r, err := s.ReadUser(ctx, actor, user)
	if err != nil {
		return Page{}, err
	}
	u, err := storage.Decode[identity.User](r)
	if err != nil {
		return Page{}, err
	}
	page, err := s.page(ctx, "sessions", "", queryScope(actor, "sessions", user, false), cursor, limit)
	if err != nil {
		return Page{}, err
	}
	result := Page{Items: []storage.Record{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	for _, row := range page.Items {
		v, err := storage.Decode[sessions.Session](row)
		if err != nil {
			return result, err
		}
		now := time.Now()
		idle := s.SessionIdleTimeout
		if idle == 0 {
			idle = time.Hour
		}
		if v.UserID != user || u.Disabled || u.SessionEpoch != v.UserEpoch || !now.Before(v.ExpiresAt) || !now.Before(v.AuthenticationUntil) || now.Sub(v.LastSeen) >= idle {
			continue
		}
		view := SessionView{ID: row.ID, Provider: v.Provider, CreatedAt: v.CreatedAt, LastSeen: v.LastSeen, ExpiresAt: v.ExpiresAt, AuthenticationUntil: v.AuthenticationUntil, Current: row.ID == current}
		row.Data, err = json.Marshal(view)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, row)
	}
	return result, nil
}

func (s *Service) RevokeSession(ctx context.Context, actor, user, id, why string, revision int64) error {
	if err := s.Auth.Check(ctx, actor, "", "user/"+user, "users.manage"); err != nil {
		return err
	}
	if err := reason(why); err != nil {
		return err
	}
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	r, err := s.Store.Get(ctx, "sessions", id)
	if err != nil {
		return err
	}
	v, err := storage.Decode[sessions.Session](r)
	if err != nil {
		return err
	}
	if v.UserID != user {
		return storage.ErrNotFound
	}
	if revision < 1 || revision != r.Revision {
		return storage.ErrConflict
	}
	before := SessionView{ID: id, Provider: v.Provider, CreatedAt: v.CreatedAt, LastSeen: v.LastSeen, ExpiresAt: v.ExpiresAt, AuthenticationUntil: v.AuthenticationUntil}
	return s.commit(ctx, actor, "", "user.session.revoke", user+"/"+id, why, before, nil, version, storage.Change{Record: r, Delete: true})
}
