package oidc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"strings"
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
	verifier, keys, err := s.tokenVerifier(ctx, remote, p)
	if err != nil {
		return verifiedIdentity{}, err
	}
	verified, err := verifier.Verify(ctx, raw)
	if err != nil {
		return verifiedIdentity{}, errors.New("identity token verification failed")
	}
	if verified.Subject == "" || len(verified.Subject) > 255 || strings.IndexFunc(verified.Subject, func(r rune) bool { return r < 0x20 || r > 0x7e }) >= 0 || verified.IssuedAt.After(time.Now().Add(time.Minute)) || subtle.ConstantTimeCompare([]byte(verified.Nonce), []byte(v.Nonce)) != 1 {
		return verifiedIdentity{}, errors.New("identity token nonce, subject, or issue time is invalid")
	}
	claims := keys.claims

	var issued float64
	if err := json.Unmarshal(claims["iat"], &issued); err != nil {
		return verifiedIdentity{}, errors.New("identity token issue time is required")
	}
	if issued <= 0 {
		return verifiedIdentity{}, errors.New("identity token issue time is invalid")
	}
	if data, present := claims["azp"]; present || len(verified.Audience) > 1 {
		var party string
		if err := json.Unmarshal(data, &party); err != nil || party != p.ClientID {
			return verifiedIdentity{}, errors.New("authorized party is invalid")
		}
	}
	if data, present := claims["token_use"]; present {
		var use string
		if err := json.Unmarshal(data, &use); err != nil || use != "id" {
			return verifiedIdentity{}, errors.New("identity token use is invalid")
		}
	}
	if data, present := claims["at_hash"]; present {
		var hash string
		if err := json.Unmarshal(data, &hash); err != nil || hash == "" {
			return verifiedIdentity{}, errors.New("access token binding is invalid")
		}
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
