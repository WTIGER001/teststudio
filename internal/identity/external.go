package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type ExternalIdentity struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
	UserID  string `json:"userId"`
}

func externalID(issuer, subject string) string {
	data, _ := json.Marshal([]string{issuer, subject})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func (d *Directory) ResolveExternal(ctx context.Context, issuer, subject, name, email string, verified, provision bool) (User, error) {
	if issuer == "" || subject == "" {
		return User{}, errors.New("issuer and subject are required")
	}
	id := externalID(issuer, subject)
	if r, err := d.Store.Get(ctx, "external-identities", id); err == nil {
		i, err := storage.Decode[ExternalIdentity](r)
		if err != nil {
			return User{}, err
		}
		u, err := d.User(ctx, i.UserID)
		if err == nil && u.Disabled {
			err = errors.New("account is disabled")
		}
		return u, err
	} else if !errors.Is(err, storage.ErrNotFound) {
		return User{}, err
	}
	if !provision {
		return User{}, errors.New("account has not been provisioned")
	}
	if name == "" {
		name = subject
	}
	u := User{ID: domain.NewID(), Name: name, Email: email, EmailVerified: verified, CreatedAt: time.Now().UTC()}
	i := ExternalIdentity{Issuer: issuer, Subject: subject, UserID: u.ID}
	user, err := storage.Put("users", u.ID, "", 0, u)
	if err != nil {
		return User{}, err
	}
	link, err := storage.Put("external-identities", id, "", 0, i)
	if err != nil {
		return User{}, err
	}
	audit, err := domain.Audit(u.ID, "", "identity.provision", u.ID, nil, i)
	if err != nil {
		return User{}, err
	}
	if err := d.Store.Commit(ctx, user, link, audit); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return d.ResolveExternal(ctx, issuer, subject, name, email, verified, false)
		}
		return User{}, err
	}
	return u, nil
}

func (d *Directory) SyncExternalGroups(ctx context.Context, user, source string, groups []string, expires time.Time) error {
	if len(groups) > 40 {
		return errors.New("external group mapping exceeds 40 memberships")
	}
	rows, err := d.Store.All(ctx, "memberships", "")
	if err != nil {
		return err
	}
	changes := []storage.Change{}
	existing := map[string]storage.Record{}
	for _, r := range rows {
		m, err := storage.Decode[Membership](r)
		if err != nil {
			return err
		}
		if m.UserID == user && m.Source == source {
			existing[m.GroupID] = r
		}
	}
	seen := map[string]bool{}
	for _, id := range groups {
		if seen[id] {
			continue
		}
		seen[id] = true
		g, err := d.Group(ctx, id)
		if err != nil {
			return err
		}
		if g.ManagedBy != source {
			return errors.New("external group mapping does not own the destination group")
		}
		if g.Archived {
			continue
		}
		r := existing[id]
		m := Membership{UserID: user, GroupID: id, Source: source, ExpiresAt: &expires}
		c, err := storage.Put("memberships", membershipID(user, id, source), g.Project, r.Revision, m)
		if err != nil {
			return err
		}
		changes = append(changes, c)
		delete(existing, id)
	}
	for _, r := range existing {
		changes = append(changes, storage.Change{Record: r, Delete: true})
	}
	if len(changes) == 0 {
		return nil
	}
	audit, err := domain.Audit(user, "", "identity.groups.sync", source, nil, groups)
	if err != nil {
		return err
	}
	changes = append(changes, audit)
	return d.Store.Commit(ctx, changes...)
}
