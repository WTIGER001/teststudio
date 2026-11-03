package oidc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/wtiger001/teststudio/internal/accountlinks"
	"golang.org/x/oauth2"
)

type verifiedIdentity struct {
	Token           *coreoidc.IDToken
	Claims          map[string]json.RawMessage
	AuthenticatedAt time.Time
}

func (v verifiedIdentity) text(name string) string {
	var value string
	_ = json.Unmarshal(v.Claims[name], &value)
	return value
}

func (s *Service) exchangeIdentity(ctx context.Context, p Provider, remote *coreoidc.Provider, cfg oauth2.Config, v transaction, code string) (verifiedIdentity, error) {
	token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(v.Verifier))
	if err != nil {
		return verifiedIdentity{}, errors.New("authorization code exchange failed")
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return verifiedIdentity{}, errors.New("identity token is missing")
	}
	verified, err := remote.Verifier(&coreoidc.Config{ClientID: p.ClientID}).Verify(ctx, raw)
	if err != nil {
		return verifiedIdentity{}, errors.New("identity token verification failed")
	}
	if verified.Subject == "" || verified.IssuedAt.After(time.Now().Add(time.Minute)) || subtle.ConstantTimeCompare([]byte(verified.Nonce), []byte(v.Nonce)) != 1 {
		return verifiedIdentity{}, errors.New("identity token nonce, subject, or issue time is invalid")
	}
	var claims map[string]json.RawMessage
	if err := verified.Claims(&claims); err != nil {
		return verifiedIdentity{}, err
	}
	text := func(name string) string { var value string; _ = json.Unmarshal(claims[name], &value); return value }
	if len(verified.Audience) > 1 && text("azp") != p.ClientID {
		return verifiedIdentity{}, errors.New("authorized party is invalid")
	}
	if verified.AccessTokenHash != "" {
		if err := verified.VerifyAccessToken(token.AccessToken); err != nil {
			return verifiedIdentity{}, errors.New("access token binding is invalid")
		}
	}

	result := verifiedIdentity{Token: verified, Claims: claims}
	if v.Account != nil {
		var authTime int64
		if err := json.Unmarshal(claims["auth_time"], &authTime); err != nil {
			return verifiedIdentity{}, errors.New("fresh authentication time is required")
		}
		result.AuthenticatedAt = time.Unix(authTime, 0).UTC()
		if !accountlinks.FreshAuthentication(result.AuthenticatedAt, v.Account.StartedAt, time.Now()) {
			return verifiedIdentity{}, errors.New("fresh authentication is required")
		}
	}
	return result, nil
}
