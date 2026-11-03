package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

type User struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email,omitempty"`
	EmailVerified bool       `json:"emailVerified"`
	Disabled      bool       `json:"disabled"`
	SessionEpoch  int64      `json:"sessionEpoch"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastSignIn    *time.Time `json:"lastSignIn,omitempty"`
}

type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Project     string `json:"projectId,omitempty"`
	Archived    bool   `json:"archived"`
	ManagedBy   string `json:"managedBy,omitempty"`
}

type Membership struct {
	ProviderRevision int64      `json:"providerRevision,omitempty"`
	UserID           string     `json:"userId"`
	GroupID          string     `json:"groupId"`
	Source           string     `json:"source"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
}

type Directory struct{ Store *storage.Store }

func (d *Directory) User(ctx context.Context, id string) (User, error) {
	r, err := d.Store.Get(ctx, "users", id)
	if err != nil {
		return User{}, err
	}
	return storage.Decode[User](r)
}

func (d *Directory) SaveUser(ctx context.Context, u User, revision int64) error {
	if u.ID == "" || u.Name == "" {
		return errors.New("user identifier and name are required")
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}
	c, err := storage.Put("users", u.ID, "", revision, u)
	if err != nil {
		return err
	}
	return d.commit(ctx, c)
}

func (d *Directory) Group(ctx context.Context, id string) (Group, error) {
	r, err := d.Store.Get(ctx, "groups", id)
	if err != nil {
		return Group{}, err
	}
	return storage.Decode[Group](r)
}

func (d *Directory) SaveGroup(ctx context.Context, g Group, revision int64) error {
	if g.ID == "" || g.Name == "" {
		return errors.New("group identifier and name are required")
	}
	c, err := storage.Put("groups", g.ID, g.Project, revision, g)
	if err != nil {
		return err
	}
	return d.commit(ctx, c)
}

func MembershipID(user, group, source string) string {
	h := sha256.Sum256([]byte(user + "\x00" + group + "\x00" + source))
	return hex.EncodeToString(h[:])
}

func (d *Directory) AddMember(ctx context.Context, m Membership) error {
	u, err := d.User(ctx, m.UserID)
	if err != nil {
		return err
	}
	g, err := d.Group(ctx, m.GroupID)
	if err != nil {
		return err
	}
	if u.Disabled || g.Archived {
		return errors.New("user or group is inactive")
	}
	if m.Source == "" {
		m.Source = "local"
	}
	if g.ManagedBy != "" && g.ManagedBy != m.Source {
		return errors.New("membership belongs to an external provider")
	}
	id := MembershipID(m.UserID, m.GroupID, m.Source)
	r, err := d.Store.Get(ctx, "memberships", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	c, err := storage.Put("memberships", id, g.Project, r.Revision, m)
	if err != nil {
		return err
	}
	return d.commit(ctx, c)
}

func (d *Directory) RemoveMember(ctx context.Context, user, group, source string) error {
	if source == "" {
		source = "local"
	}
	r, err := d.Store.Get(ctx, "memberships", MembershipID(user, group, source))
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return d.commit(ctx, storage.Change{Record: r, Delete: true})
}

func (d *Directory) memberships(ctx context.Context) ([]Membership, error) {
	rows, err := d.Store.All(ctx, "memberships", "")
	if err != nil {
		return nil, err
	}
	result := []Membership{}
	for _, r := range rows {
		m, err := storage.Decode[Membership](r)
		if err != nil {
			return nil, err
		}
		active, err := d.MembershipCurrent(ctx, m)
		if err != nil {
			return nil, err
		}
		if active {
			result = append(result, m)
		}
	}
	return result, nil
}

func (d *Directory) GetUserGroups(ctx context.Context, user string) ([]string, error) {
	u, err := d.User(ctx, user)
	if err != nil {
		return nil, err
	}
	if u.Disabled {
		return []string{}, nil
	}
	all, err := d.memberships(ctx)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, m := range all {
		if m.UserID != user {
			continue
		}
		g, err := d.Group(ctx, m.GroupID)
		if err != nil {
			return nil, err
		}
		if !g.Archived && !slices.Contains(result, g.ID) {
			result = append(result, g.ID)
		}
	}
	slices.Sort(result)
	return result, nil
}

func (d *Directory) GetGroupMembers(ctx context.Context, group string) ([]string, error) {
	g, err := d.Group(ctx, group)
	if err != nil {
		return nil, err
	}
	if g.Archived {
		return []string{}, nil
	}
	all, err := d.memberships(ctx)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, m := range all {
		if m.GroupID != group {
			continue
		}
		u, err := d.User(ctx, m.UserID)
		if err != nil {
			return nil, err
		}
		if !u.Disabled && !slices.Contains(result, u.ID) {
			result = append(result, u.ID)
		}
	}
	slices.Sort(result)
	return result, nil
}

func (d *Directory) IsUserInGroup(ctx context.Context, user, group string) (bool, error) {
	groups, err := d.GetUserGroups(ctx, user)
	return slices.Contains(groups, group), err
}

func (d *Directory) GetUserTeams(context.Context, string) ([]string, error)     { return []string{}, nil }
func (d *Directory) IsUserInTeam(context.Context, string, string) (bool, error) { return false, nil }
