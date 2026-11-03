package access

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

type MembershipView struct {
	identity.Membership
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	ReadOnly bool   `json:"readOnly"`
}

func (s *Service) ReadMember(ctx context.Context, actor, group, user, source string) (storage.Record, error) {
	g, _, err := s.group(ctx, actor, group, "groups.read")
	if err != nil {
		return storage.Record{}, err
	}
	if g.Project != "" {
		if err := s.Auth.Check(ctx, actor, g.Project, "group/"+group, "members.read"); err != nil {
			return storage.Record{}, err
		}
	}
	if source == "" {
		source = "local"
	}
	if !domain.ValidID(user) || len(source) > 256 {
		return storage.Record{}, errors.New("valid user and membership source are required")
	}
	row, err := s.Store.Get(ctx, "memberships", identity.MembershipID(user, group, source))
	if err != nil {
		return storage.Record{}, err
	}
	return s.memberView(ctx, row, g)
}

func (s *Service) memberView(ctx context.Context, row storage.Record, g identity.Group) (storage.Record, error) {
	m, err := storage.Decode[identity.Membership](row)
	if err != nil {
		return row, err
	}
	if m.GroupID != g.ID || row.Project != g.Project {
		return storage.Record{}, storage.ErrNotFound
	}
	u, err := s.Auth.Directory.User(ctx, m.UserID)
	if err != nil {
		return row, err
	}
	active, err := s.Auth.Directory.MembershipCurrent(ctx, m)
	if err != nil {
		return row, err
	}
	view := MembershipView{Membership: m, Name: u.Name, Active: !g.Archived && !u.Disabled && active, ReadOnly: g.ManagedBy != "" || m.Source != "local"}
	row.Data, err = json.Marshal(view)
	return row, err
}

func (s *Service) ListMembers(ctx context.Context, actor, group, cursor string, limit int) (Page, error) {
	g, _, err := s.group(ctx, actor, group, "groups.read")
	if err != nil {
		return Page{}, err
	}
	if g.Project != "" {
		if err := s.Auth.Check(ctx, actor, g.Project, "group/"+group, "members.read"); err != nil {
			return Page{}, err
		}
	}
	page, err := s.page(ctx, "memberships", g.Project, queryScope(actor, "members", group, false), cursor, limit)
	if err != nil {
		return Page{}, err
	}
	result := Page{Items: []storage.Record{}, Cursor: page.Cursor, Partial: page.Cursor != ""}
	for _, row := range page.Items {
		m, err := storage.Decode[identity.Membership](row)
		if err != nil {
			return result, err
		}
		if m.GroupID != group || row.Project != g.Project {
			continue
		}
		row, err = s.memberView(ctx, row, g)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, row)
	}
	return result, nil
}

func (s *Service) SaveMember(ctx context.Context, actor, group, user string, expires *time.Time, revision int64) error {
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	g, gr, err := s.group(ctx, actor, group, "groups.manage")
	if err != nil {
		return err
	}
	if g.Project != "" {
		if err := s.Auth.Check(ctx, actor, g.Project, "group/"+group, "members.manage"); err != nil {
			return err
		}
	}
	if g.ManagedBy != "" {
		return errors.New("external membership is read-only; update its identity provider")
	}
	if g.Archived {
		return errors.New("restore the group before changing its membership")
	}
	if !domain.ValidID(user) || (expires != nil && !expires.After(time.Now())) {
		return errors.New("a valid user and a future membership expiry are required")
	}
	ur, err := s.Store.Get(ctx, "users", user)
	if err != nil {
		return err
	}
	u, err := storage.Decode[identity.User](ur)
	if err != nil {
		return err
	}
	if u.Disabled {
		return errors.New("restore the user before changing membership")
	}
	id := identity.MembershipID(user, group, "local")
	r, err := s.Store.Get(ctx, "memberships", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if revision != r.Revision {
		return storage.ErrConflict
	}
	var old *identity.Membership
	if r.Revision > 0 {
		value, err := storage.Decode[identity.Membership](r)
		if err != nil {
			return err
		}
		old = &value
	}
	v := identity.Membership{UserID: user, GroupID: group, Source: "local", ExpiresAt: expires}
	c, err := storage.Put("memberships", id, g.Project, revision, v)
	if err != nil {
		return err
	}
	if expires != nil {
		if err := s.preserveAdministrator(ctx, c); err != nil {
			return err
		}
	}
	return s.commit(ctx, actor, g.Project, "group.member.save", group+"/"+user, "", old, v, version, c, storage.Change{Record: gr, Assert: true}, storage.Change{Record: ur, Assert: true})
}

func (s *Service) RemoveMember(ctx context.Context, actor, group, user, why string, revision int64) error {
	if err := reason(why); err != nil {
		return err
	}
	version, err := s.Auth.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	g, gr, err := s.group(ctx, actor, group, "groups.manage")
	if err != nil {
		return err
	}
	if g.Project != "" {
		if err := s.Auth.Check(ctx, actor, g.Project, "group/"+group, "members.manage"); err != nil {
			return err
		}
	}
	if g.ManagedBy != "" {
		return errors.New("external membership is read-only; update its identity provider")
	}
	r, err := s.Store.Get(ctx, "memberships", identity.MembershipID(user, group, "local"))
	if err != nil {
		return err
	}
	if revision < 1 || r.Revision != revision {
		return storage.ErrConflict
	}
	old, err := storage.Decode[identity.Membership](r)
	if err != nil {
		return err
	}
	c := storage.Change{Record: r, Delete: true}
	if err := s.preserveAdministrator(ctx, c); err != nil {
		return err
	}
	return s.commit(ctx, actor, g.Project, "group.member.remove", group+"/"+user, why, old, nil, version, c, storage.Change{Record: gr, Assert: true})
}
