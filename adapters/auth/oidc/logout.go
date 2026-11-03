package oidc

import (
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type LogoutResponse struct {
	SignedOut                 bool         `json:"signedOut"`
	ProviderLogout            *LogoutOffer `json:"providerLogout,omitempty"`
	ProviderLogoutUnavailable bool         `json:"providerLogoutUnavailable,omitempty"`
}

type LogoutOffer struct {
	ProviderName string    `json:"providerName"`
	URL          string    `json:"url"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type logoutTransaction struct {
	UserID           string    `json:"userId"`
	ProviderID       string    `json:"providerId"`
	ProviderRevision int64     `json:"providerRevision"`
	State            string    `json:"state"`
	OfferDigest      string    `json:"offerDigest"`
	Stage            string    `json:"stage"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type logoutIndex struct {
	BrowserID string `json:"browserId"`
}

var errLogout = errors.New("provider sign-out could not be confirmed; the TestStudio session remains signed out")

func validateLogoutEndpoint(value string) error {
	u, err := url.Parse(value)
	if err != nil || len(value) > 2048 || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("invalid provider logout endpoint")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))) {
		return errors.New("provider logout requires HTTPS outside loopback")
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return errors.New("invalid provider logout endpoint query")
	}
	for _, key := range []string{"client_id", "post_logout_redirect_uri", "state", "id_token_hint", "logout_hint", "client_secret", "access_token", "refresh_token", "id_token"} {
		if _, present := query[key]; present {
			return errors.New("provider logout endpoint cannot include reserved authentication parameters")
		}
	}
	return nil
}

func (s *Service) OfferLogout(ctx context.Context, current sessions.Session) (*LogoutOffer, *http.Cookie, error) {
	if current.AuthenticationMethod != "oidc" {
		return nil, nil, nil
	}
	if len(current.ID) != 64 {
		return nil, nil, errLogout
	}
	if _, err := s.Store.Get(ctx, "sessions", current.ID); !errors.Is(err, storage.ErrNotFound) {
		return nil, nil, errLogout
	}
	row, err := s.Store.Get(ctx, "providers", current.Provider)
	if err != nil {
		return nil, nil, err
	}
	provider, err := storage.Decode[Provider](row)
	if err != nil {
		return nil, nil, err
	}
	if row.Project != "" || provider.ID != current.Provider || row.Revision != current.ProviderRevision || provider.Disabled {
		return nil, nil, errLogout
	}
	if provider.LogoutEndpoint == "" {
		return nil, nil, nil
	}
	if err := ValidateProvider(provider); err != nil {
		return nil, nil, err
	}
	token, browser := sessions.Random(), sessions.Random()
	until := time.Now().UTC().Add(5 * time.Minute)
	transaction := logoutTransaction{UserID: current.UserID, ProviderID: provider.ID, ProviderRevision: row.Revision, State: sessions.Random(), OfferDigest: sessions.Digest(token), Stage: "ready", ExpiresAt: until}
	private, err := storage.Put("logout-transactions", sessions.Digest(browser), "", 0, transaction)
	if err != nil {
		return nil, nil, err
	}
	index, err := storage.Put("logout-offers", transaction.OfferDigest, "", 0, logoutIndex{BrowserID: private.ID})
	if err != nil {
		return nil, nil, err
	}
	audit, err := domain.Audit(current.UserID, "", "identity.provider-logout.offer", provider.ID, nil, map[string]int64{"providerRevision": row.Revision})
	if err != nil {
		return nil, nil, err
	}
	if err := s.Store.Commit(ctx, storage.Change{Record: row, Assert: true}, private, index, audit); err != nil {
		return nil, nil, err
	}
	cookie := &http.Cookie{Name: "teststudio_logout", Value: browser, Path: "/auth", HttpOnly: true, Secure: s.Sessions.Secure, SameSite: http.SameSiteLaxMode, MaxAge: 300}
	return &LogoutOffer{ProviderName: provider.Name, URL: "/auth/logout/" + token, ExpiresAt: until}, cookie, nil
}

func (s *Service) logoutTransaction(ctx context.Context, r *http.Request) (storage.Record, logoutTransaction, error) {
	cookie, err := r.Cookie("teststudio_logout")
	if err != nil || len(cookie.Value) != 43 {
		return storage.Record{}, logoutTransaction{}, errLogout
	}
	row, err := s.Store.Get(ctx, "logout-transactions", sessions.Digest(cookie.Value))
	if err != nil {
		return row, logoutTransaction{}, errLogout
	}
	value, err := storage.Decode[logoutTransaction](row)
	if err != nil || row.Project != "" || !time.Now().Before(value.ExpiresAt) {
		return storage.Record{}, logoutTransaction{}, errLogout
	}
	return row, value, nil
}

func (s *Service) logoutProvider(ctx context.Context, value logoutTransaction) (Provider, storage.Record, error) {
	row, err := s.Store.Get(ctx, "providers", value.ProviderID)
	if err != nil {
		return Provider{}, row, errLogout
	}
	provider, err := storage.Decode[Provider](row)
	if err != nil || provider.ID != value.ProviderID || row.Project != "" || row.Revision != value.ProviderRevision || provider.Disabled || provider.LogoutEndpoint == "" {
		return Provider{}, storage.Record{}, errLogout
	}
	if err := ValidateProvider(provider); err != nil {
		return Provider{}, storage.Record{}, errLogout
	}
	return provider, row, nil
}

func (s *Service) logoutEndpoint(ctx context.Context, provider Provider) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client := s.httpClient()
	remote, err := coreoidc.NewProvider(coreoidc.ClientContext(ctx, client), provider.Issuer)
	if err != nil {
		return nil, errLogout
	}
	var metadata struct {
		Endpoint string `json:"end_session_endpoint"`
	}
	if err := remote.Claims(&metadata); err != nil || metadata.Endpoint != provider.LogoutEndpoint {
		return nil, errLogout
	}
	endpoint, err := url.Parse(metadata.Endpoint)
	if err != nil {
		return nil, errLogout
	}
	return endpoint, nil
}

func (s *Service) LaunchLogout(w http.ResponseWriter, r *http.Request, token string) error {
	if len(token) != 43 {
		return errLogout
	}
	row, value, err := s.logoutTransaction(r.Context(), r)
	if err != nil {
		return err
	}
	if value.Stage != "ready" || subtle.ConstantTimeCompare([]byte(value.OfferDigest), []byte(sessions.Digest(token))) != 1 {
		return errLogout
	}
	index, err := s.Store.Get(r.Context(), "logout-offers", value.OfferDigest)
	if err != nil {
		return errLogout
	}
	binding, err := storage.Decode[logoutIndex](index)
	if err != nil || index.Project != "" || binding.BrowserID != row.ID {
		return errLogout
	}
	provider, providerRow, err := s.logoutProvider(r.Context(), value)
	if err != nil {
		return err
	}
	endpoint, err := s.logoutEndpoint(r.Context(), provider)
	if err != nil {
		return err
	}
	if !time.Now().Before(value.ExpiresAt) {
		return errLogout
	}
	value.Stage = "launched"
	change, err := storage.Put(row.Bucket, row.ID, "", row.Revision, value)
	if err != nil {
		return err
	}
	if err := s.Store.Commit(r.Context(), change, storage.Change{Record: index, Delete: true}, storage.Change{Record: providerRow, Assert: true}); err != nil {
		return errLogout
	}
	query := endpoint.Query()
	query.Set("client_id", provider.ClientID)
	query.Set("post_logout_redirect_uri", s.Origin+"/auth/logout/callback/"+provider.ID)
	query.Set("state", value.State)
	endpoint.RawQuery = query.Encode()
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, endpoint.String(), http.StatusFound)
	return nil
}

func (s *Service) CompleteLogout(w http.ResponseWriter, r *http.Request, provider string) error {
	row, value, err := s.logoutTransaction(r.Context(), r)
	if err != nil {
		return err
	}
	if value.Stage != "launched" || value.ProviderID != provider || subtle.ConstantTimeCompare([]byte(value.State), []byte(r.URL.Query().Get("state"))) != 1 {
		return errLogout
	}
	_, providerRow, err := s.logoutProvider(r.Context(), value)
	if err != nil {
		return err
	}
	action, outcome := "identity.provider-logout.complete", "complete"
	if r.URL.Query().Get("error") != "" {
		action, outcome = "identity.provider-logout.unconfirmed", "unconfirmed"
	}
	audit, err := domain.Audit(value.UserID, "", action, provider, nil, map[string]int64{"providerRevision": value.ProviderRevision})
	if err != nil {
		return err
	}
	if err := s.Store.Commit(r.Context(), storage.Change{Record: row, Delete: true}, audit, storage.Change{Record: providerRow, Assert: true}); err != nil {
		return errLogout
	}
	http.SetCookie(w, &http.Cookie{Name: "teststudio_logout", Value: "", Path: "/auth", HttpOnly: true, Secure: s.Sessions.Secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, "/projects?providerLogout="+outcome, http.StatusFound)
	return nil
}
