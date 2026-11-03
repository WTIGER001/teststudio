package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
)

type claimKeySet struct {
	keys   coreoidc.KeySet
	claims map[string]json.RawMessage
}

func (s *claimKeySet) VerifySignature(ctx context.Context, raw string) ([]byte, error) {
	payload, err := s.keys.VerifySignature(ctx, raw)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(raw, ".")
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(header, &fields); err != nil {
		return nil, err
	}
	if data, present := fields["typ"]; present {
		var kind string
		if err := json.Unmarshal(data, &kind); err != nil || strings.EqualFold(kind, "at+jwt") || strings.EqualFold(kind, "application/at+jwt") {
			return nil, errors.New("access token cannot be used as an identity token")
		}
	}
	if err := json.Unmarshal(payload, &s.claims); err != nil {
		return nil, err
	}
	for _, name := range []string{"iss", "sub", "aud", "exp", "iat", "nbf", "nonce", "at_hash", "azp", "token_use"} {
		for key := range s.claims {
			if key != name && strings.EqualFold(key, name) {
				return nil, errors.New("identity claims use ambiguous registered names")
			}
		}
	}
	_, distributed := s.claims["_claim_names"]
	_, sources := s.claims["_claim_sources"]
	if !distributed && !sources {
		return payload, nil
	}
	basic := make(map[string]json.RawMessage, len(s.claims))
	for key, value := range s.claims {
		if key != "_claim_names" && key != "_claim_sources" {
			basic[key] = value
		}
	}
	return json.Marshal(basic)
}

func signingAlgorithms(values []string) []string {
	allowed := map[string]bool{coreoidc.RS256: true, coreoidc.RS384: true, coreoidc.RS512: true, coreoidc.ES256: true, coreoidc.ES384: true, coreoidc.ES512: true, coreoidc.PS256: true, coreoidc.PS384: true, coreoidc.PS512: true, coreoidc.EdDSA: true}
	result := []string{}
	for _, value := range values {
		if allowed[value] {
			result = append(result, value)
		}
	}
	return result
}

func (s *Service) tokenVerifier(ctx context.Context, remote *coreoidc.Provider, p Provider) (*coreoidc.IDTokenVerifier, *claimKeySet, error) {
	var metadata coreoidc.ProviderConfig
	if err := remote.Claims(&metadata); err != nil {
		return nil, nil, err
	}
	keys := &claimKeySet{keys: coreoidc.NewRemoteKeySet(coreoidc.ClientContext(ctx, s.httpClient()), metadata.JWKSURL)}
	verifier := coreoidc.NewVerifier(p.Issuer, keys, &coreoidc.Config{ClientID: p.ClientID, SupportedSigningAlgs: signingAlgorithms(metadata.Algorithms)})
	return verifier, keys, nil
}

func providerEmailVerified(claims map[string]json.RawMessage) bool {
	var verified bool
	if err := json.Unmarshal(claims["email_verified"], &verified); err != nil || !verified {
		return false
	}
	if data, present := claims["_claim_names"]; present {
		var names map[string]json.RawMessage
		if err := json.Unmarshal(data, &names); err != nil || names == nil {
			return false
		}
		for _, key := range []string{"email", "email_verified"} {
			if _, present := names[key]; present {
				return false
			}
		}
	}
	return true
}
