package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
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

func ExternalIdentityID(issuer, subject string) string { return externalID(issuer, subject) }

func (d *Directory) ExternalLink(ctx context.Context, issuer, subject string) (ExternalIdentity, error) {
	if issuer == "" || subject == "" {
		return ExternalIdentity{}, errors.New("issuer and subject are required")
	}
	row, err := d.Store.Get(ctx, "external-identities", externalID(issuer, subject))
	if err != nil {
		return ExternalIdentity{}, err
	}
	link, err := storage.Decode[ExternalIdentity](row)
	if err != nil {
		return ExternalIdentity{}, err
	}
	if link.Issuer != issuer || link.Subject != subject {
		return ExternalIdentity{}, storage.ErrNotFound
	}
	return link, nil
}

func (d *Directory) ResolveExternal(ctx context.Context, issuer, subject, name, email string, verified, provision bool) (User, error) {
	return d.resolveExternal(ctx, issuer, subject, name, email, verified, provision, nil)
}

func (d *Directory) resolveExternal(ctx context.Context, issuer, subject, name, email string, verified, provision bool, guards []storage.Change) (User, error) {
	if issuer == "" || subject == "" {
		return User{}, errors.New("issuer and subject are required")
	}
	id := externalID(issuer, subject)
	if i, err := d.ExternalLink(ctx, issuer, subject); err == nil {
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
	changes := append([]storage.Change{user, link, audit}, guards...)
	if err := d.commit(ctx, changes...); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return d.resolveExternal(ctx, issuer, subject, name, email, verified, false, guards)
		}
		return User{}, err
	}
	return u, nil
}

func (d *Directory) SyncExternalGroups(ctx context.Context, user, source string, groups []string, expires time.Time) error {
	if strings.HasPrefix(source, "oidc:") {
		return errors.New("OIDC membership synchronization requires a current provider revision")
	}
	return d.syncExternalGroups(ctx, user, source, groups, expires, 0, nil)
}

func (d *Directory) SyncProviderGroups(ctx context.Context, user, provider string, revision int64, groups []string, expires time.Time) error {
	row, err := d.providerAt(ctx, provider, revision)
	if err != nil {
		return err
	}
	return d.syncExternalGroups(ctx, user, "oidc:"+provider, groups, expires, revision, []storage.Change{{Record: row, Assert: true}})
}

func (d *Directory) syncExternalGroups(ctx context.Context, user, source string, groups []string, expires time.Time, providerRevision int64, guards []storage.Change) error {
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
		m := Membership{ProviderRevision: providerRevision, UserID: user, GroupID: id, Source: source, ExpiresAt: &expires}
		c, err := storage.Put("memberships", MembershipID(user, id, source), g.Project, r.Revision, m)
		if err != nil {
			return err
		}
		changes = append(changes, c)
		delete(existing, id)
	}
	for _, r := range existing {
		changes = append(changes, storage.Change{Record: r, Delete: true})
	}
	if len(changes) == 0 && len(guards) == 0 {
		return nil
	}
	audit, err := domain.Audit(user, "", "identity.groups.sync", source, nil, groups)
	if err != nil {
		return err
	}
	changes = append(changes, guards...)
	changes = append(changes, audit)
	return d.commit(ctx, changes...)
}
