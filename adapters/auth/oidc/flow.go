package oidc

import (
	"context"
	"crypto/subtle"
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

var ErrSignInCancelled = errors.New("sign-in was cancelled or denied")
var ErrSignInExpired = errors.New("sign-in transaction is missing or expired")

type SignInError struct {
	Cause    error
	ReturnTo string
}

func (e *SignInError) Error() string { return e.Cause.Error() }
func (e *SignInError) Unwrap() error { return e.Cause }

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
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") || strings.Contains(u.Path, "\\") || strings.HasPrefix(u.Path, "//") || len(value) > 4096 || u.Path == "/auth" || strings.HasPrefix(u.Path, "/auth/") || u.Path == "/api" || strings.HasPrefix(u.Path, "/api/") {
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

func (s *Service) Callback(w http.ResponseWriter, r *http.Request, id string) (outcome error) {
	cookie, err := r.Cookie("teststudio_login")
	if err != nil {
		return ErrSignInExpired
	}
	record, err := s.Store.Get(r.Context(), "login-transactions", sessions.Digest(cookie.Value))
	if err != nil {
		return ErrSignInExpired
	}
	v, err := storage.Decode[transaction](record)
	if err != nil {
		return err
	}
	if v.Provider != id || subtle.ConstantTimeCompare([]byte(v.State), []byte(r.URL.Query().Get("state"))) != 1 {
		return errors.New("sign-in state is invalid")
	}
	defer func() {
		if outcome != nil && v.Account == nil {
			outcome = &SignInError{Cause: outcome, ReturnTo: returnPath(v.ReturnTo)}
		}
	}()
	if !time.Now().Before(v.ExpiresAt) {
		return ErrSignInExpired
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
		return ErrSignInCancelled
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
	ctx = coreoidc.ClientContext(ctx, s.httpClient())
	identity, err := s.exchangeIdentity(ctx, p, remote, cfg, v, r.URL.Query().Get("code"))
	if err != nil {
		return err
	}
	verified, claims, text := identity.Token, identity.Claims, identity.text
	emailVerified := providerEmailVerified(claims)
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
		groups, err := mappedGroups(p, claims)
		if err != nil {
			return rejectGroups(err.Error())
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
	if s.ProjectAccess != nil {
		if err := s.ProjectAccess.ApplyPendingAccess(ctx, SignInIdentity{UserID: user.ID, ProviderID: p.ID, ProviderRevision: v.ProviderRevision, Issuer: verified.Issuer, Subject: verified.Subject, Email: text("email"), EmailVerified: emailVerified, ExpiresAt: verified.Expiry}); err != nil {
			return err
		}
	}
	previous := ""
	if cookie, err := r.Cookie("teststudio_session"); err == nil && len(cookie.Value) == 43 {
		previous = sessions.Digest(cookie.Value)
	}
	if _, err := s.Sessions.CreateOIDCReplacing(ctx, w, user.ID, p.ID, v.ProviderRevision, authUntil, previous); err != nil {
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
