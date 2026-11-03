package oidc

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/wtiger001/teststudio/internal/accountlinks"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"golang.org/x/oauth2"
)

func (s *Service) AuthorizeIdentityLink(w http.ResponseWriter, r *http.Request, id string) error {
	if s.Links == nil {
		return accountlinks.ErrInvalid
	}
	active, err := s.Sessions.Read(r.Context(), r)
	if err != nil {
		return err
	}
	browser := sessions.Random()
	proof, err := s.Links.PrepareProof(r.Context(), active.ID, id, sessions.Digest(browser))
	if err != nil {
		return err
	}
	p, _, cfg, err := s.config(r.Context(), proof.ProviderID)
	if err != nil {
		return errors.New("identity provider is unavailable")
	}
	if p.revision != proof.ProviderRevision {
		return accountlinks.ErrInvalid
	}
	v := transaction{Account: &proof, ProviderRevision: p.revision, Provider: p.ID, State: sessions.Random(), Nonce: sessions.Random(), Verifier: oauth2.GenerateVerifier(), ExpiresAt: time.Now().UTC().Add(5 * time.Minute)}
	change, err := storage.Put("login-transactions", proof.TransactionID, "", 0, v)
	if err != nil {
		return err
	}
	if err := s.Links.BindProof(r.Context(), active.ID, proof, change); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "teststudio_login", Value: browser, Path: "/auth", HttpOnly: true, Secure: s.Sessions.Secure, SameSite: http.SameSiteLaxMode, MaxAge: 300})
	http.Redirect(w, r, cfg.AuthCodeURL(v.State, oauth2.S256ChallengeOption(v.Verifier), coreoidc.Nonce(v.Nonce), oauth2.SetAuthURLParam("max_age", "0"), oauth2.SetAuthURLParam("prompt", "login")), http.StatusFound)
	return nil
}

func (s *Service) accountCallback(w http.ResponseWriter, r *http.Request, v transaction) error {
	if s.Links == nil {
		return accountlinks.ErrInvalid
	}
	active, err := s.Sessions.Read(r.Context(), r)
	if err != nil {
		return err
	}
	complete := func() error {
		if err := s.Links.CheckProof(r.Context(), active.ID, *v.Account); err != nil {
			return err
		}
		if r.URL.Query().Get("error") != "" {
			return accountlinks.ErrInvalid
		}
		p, remote, cfg, err := s.config(r.Context(), v.Provider)
		if err != nil {
			return err
		}
		if p.revision != v.ProviderRevision || p.revision != v.Account.ProviderRevision {
			return accountlinks.ErrInvalid
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		ctx = coreoidc.ClientContext(ctx, s.httpClient())
		proof, err := s.exchangeIdentity(ctx, p, remote, cfg, v, r.URL.Query().Get("code"))
		if err != nil {
			return err
		}
		_, err = s.Links.AcceptProof(ctx, active.ID, *v.Account, accountlinks.Proof{Issuer: proof.Token.Issuer, Subject: proof.Token.Subject}, proof.AuthenticatedAt, proof.Token.Expiry)
		return err
	}
	if err := complete(); err != nil {
		if err := s.Links.RejectProof(r.Context(), active.ID, *v.Account); err != nil {
			return accountlinks.ErrInvalid
		}
	}
	http.Redirect(w, r, "/account?identityLink="+url.QueryEscape(v.Account.OperationID), http.StatusFound)
	return nil
}
