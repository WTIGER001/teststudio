package oidc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/wtiger001/teststudio/internal/accountlinks"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"golang.org/x/oauth2"
)

type transaction struct {
	Account          *accountlinks.ProofRequest `json:"account,omitempty"`
	ProviderRevision int64                      `json:"providerRevision"`
	Provider         string                     `json:"provider"`
	State            string                     `json:"state"`
	Nonce            string                     `json:"nonce"`
	Verifier         string                     `json:"verifier"`
	ReturnTo         string                     `json:"returnTo"`
	ExpiresAt        time.Time                  `json:"expiresAt"`
}

func returnPath(value string) string {
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") {
		return "/"
	}
	return value
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request, id string) error {
	p, _, cfg, err := s.config(r.Context(), id)
	if err != nil {
		return err
	}
	v := transaction{ProviderRevision: p.revision, Provider: id, State: sessions.Random(), Nonce: sessions.Random(), Verifier: oauth2.GenerateVerifier(), ReturnTo: returnPath(r.URL.Query().Get("returnTo")), ExpiresAt: time.Now().UTC().Add(10 * time.Minute)}
	browser := sessions.Random()
	c, err := storage.Put("login-transactions", sessions.Digest(browser), "", 0, v)
	if err != nil {
		return err
	}
	guard := storage.Change{Record: storage.Record{Bucket: "providers", ID: id, Revision: p.revision}, Assert: true}
	if err := s.Store.Commit(r.Context(), guard, c); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "teststudio_login", Value: browser, Path: "/auth", HttpOnly: true, Secure: s.Sessions.Secure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	http.Redirect(w, r, cfg.AuthCodeURL(v.State, oauth2.S256ChallengeOption(v.Verifier), coreoidc.Nonce(v.Nonce)), http.StatusFound)
	return nil
}

func (s *Service) Callback(w http.ResponseWriter, r *http.Request, id string) error {
	cookie, err := r.Cookie("teststudio_login")
	if err != nil {
		return errors.New("sign-in transaction is missing")
	}
	record, err := s.Store.Get(r.Context(), "login-transactions", sessions.Digest(cookie.Value))
	if err != nil {
		return errors.New("sign-in transaction is unavailable")
	}
	v, err := storage.Decode[transaction](record)
	if err != nil {
		return err
	}
	if v.Provider != id || !time.Now().Before(v.ExpiresAt) || subtle.ConstantTimeCompare([]byte(v.State), []byte(r.URL.Query().Get("state"))) != 1 {
		return errors.New("sign-in state is invalid or expired")
	}
	if v.Account != nil && (v.Account.ProviderID != id || v.Account.TransactionID != record.ID) {
		return errors.New("identity proof transaction is invalid")
	}
	if err := s.Store.Commit(r.Context(), storage.Change{Record: record, Delete: true}); err != nil {
		return errors.New("sign-in transaction was already used")
	}
	http.SetCookie(w, &http.Cookie{Name: "teststudio_login", Value: "", Path: "/auth", HttpOnly: true, Secure: s.Sessions.Secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	if v.Account != nil {
		return s.accountCallback(w, r, v)
	}
	if r.URL.Query().Get("error") != "" {
		return errors.New("sign-in was cancelled or denied")
	}
	if err := s.currentProvider(r.Context(), id, v.ProviderRevision); err != nil {
		return err
	}
	p, remote, cfg, err := s.config(r.Context(), id)
	if err != nil {
		return err
	}
	if p.revision != v.ProviderRevision {
		return errors.New("sign-in provider configuration changed")
	}
	if err := s.currentProvider(r.Context(), id, v.ProviderRevision); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if s.Client != nil {
		ctx = coreoidc.ClientContext(ctx, s.Client)
	}
	identity, err := s.exchangeIdentity(ctx, p, remote, cfg, v, r.URL.Query().Get("code"))
	if err != nil {
		return err
	}
	verified, claims, text := identity.Token, identity.Claims, identity.text
	var emailVerified bool
	_ = json.Unmarshal(claims["email_verified"], &emailVerified)
	if err := s.currentProvider(ctx, id, v.ProviderRevision); err != nil {
		return err
	}
	user, err := s.Directory.ResolveProviderIdentity(ctx, p.ID, v.ProviderRevision, verified.Issuer, verified.Subject, text("name"), text("email"), emailVerified, p.AutoProvision)
	if err != nil {
		return err
	}
	authUntil := verified.Expiry
	if len(p.GroupMappings) > 0 {
		rejectGroups := func(message string) error {
			if err := s.Directory.SyncProviderGroups(ctx, user.ID, p.ID, v.ProviderRevision, nil, time.Now()); err != nil {
				return err
			}
			return errors.New(message)
		}
		var external []string
		if _, ok := claims["hasgroups"]; ok {
			return rejectGroups("provider requires an explicit group resolver")
		}
		var distributed map[string]any
		if data := claims["_claim_names"]; len(data) > 0 {
			_ = json.Unmarshal(data, &distributed)
			if distributed[p.GroupClaim] != nil {
				return rejectGroups("provider returned incomplete group membership")
			}
		}
		if err := json.Unmarshal(claims[p.GroupClaim], &external); err != nil {
			return rejectGroups("complete group membership is required")
		}
		groups := []string{}
		for _, name := range external {
			if group := p.GroupMappings[name]; group != "" {
				groups = append(groups, group)
			}
		}
		ttl := p.MembershipTTLSeconds
		if ttl == 0 {
			ttl = 1800
		}
		if until := time.Now().Add(time.Duration(ttl) * time.Second); until.Before(authUntil) {
			authUntil = until
		}
		if err := s.Directory.SyncProviderGroups(ctx, user.ID, p.ID, v.ProviderRevision, groups, authUntil); err != nil {
			return err
		}
	}
	if _, err := s.Sessions.CreateOIDC(ctx, w, user.ID, p.ID, v.ProviderRevision, authUntil); err != nil {
		return err
	}
	http.Redirect(w, r, v.ReturnTo, http.StatusFound)
	return nil
}

func (s *Service) currentProvider(ctx context.Context, id string, revision int64) error {
	row, err := s.Store.Get(ctx, "providers", id)
	if err != nil {
		return err
	}
	if revision < 1 || row.Revision != revision {
		return errors.New("sign-in provider configuration changed")
	}
	return nil
}
