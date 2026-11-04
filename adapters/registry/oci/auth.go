package oci

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) token(ctx context.Context, repo, challenge string, auth Credential) (string, error) {
	return c.tokenActions(ctx, repo, "pull", challenge, auth)
}

func (c *Client) tokenActions(ctx context.Context, repo, actions, challenge string, auth Credential) (string, error) {
	scheme, parameters, ok := strings.Cut(challenge, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", ErrDenied
	}
	values, err := challengeParameters(parameters)
	if err != nil || values["realm"] != c.tokenURL || values["service"] != c.tokenService || !approvedScope(values["scope"], repo, actions) {
		return "", ErrDenied
	}
	for key := range values {
		if key != "realm" && key != "service" && key != "scope" {
			return "", ErrDenied
		}
	}
	if c.tokens != nil {
		value, err := c.tokens(ctx, repo, actions)
		if err != nil {
			for _, known := range []error{ErrUnavailable, ErrDenied, ErrInvalid, ErrConfiguration} {
				if errors.Is(err, known) {
					return "", known
				}
			}
			return "", ErrDenied
		}
		if value == "" || len(value) > 65536 || strings.ContainsAny(value, "\r\n\x00") {
			return "", ErrInvalid
		}
		return value, nil
	}
	location, err := url.Parse(c.tokenURL)
	if err != nil {
		return "", ErrConfiguration
	}
	query := location.Query()
	query.Set("service", c.tokenService)
	query.Set("scope", "repository:"+repo+":"+actions)
	location.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location.String(), nil)
	if err != nil {
		return "", ErrInvalid
	}
	authorize(request, auth)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return "", ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return "", ErrDenied
	}
	data, err := readResponse(response, 128<<10)
	if err != nil {
		return "", err
	}
	var result struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if json.Unmarshal(data, &result) != nil {
		return "", ErrInvalid
	}
	if result.Token != "" && result.AccessToken != "" && result.Token != result.AccessToken {
		return "", ErrInvalid
	}
	if result.Token == "" {
		result.Token = result.AccessToken
	}
	if result.Token == "" || len(result.Token) > 65536 || strings.ContainsAny(result.Token, "\r\n\x00") {
		return "", ErrInvalid
	}
	return result.Token, nil
}

func challengeParameters(parameters string) (map[string]string, error) {
	if len(parameters) > 16384 {
		return nil, ErrDenied
	}
	quoted, escaped := false, false
	var converted strings.Builder
	for _, value := range parameters {
		if escaped {
			escaped = false
		} else if quoted && value == '\\' {
			escaped = true
		} else if value == '"' {
			quoted = !quoted
		} else if value == ',' && !quoted {
			value = ';'
		}
		converted.WriteRune(value)
	}
	_, values, err := mime.ParseMediaType("Bearer;" + converted.String())
	return values, err
}

func approvedScope(scope, repo, actions string) bool {
	if scope == "" {
		return true
	}
	requested, ok := strings.CutPrefix(scope, "repository:"+repo+":")
	if !ok || requested == "" {
		return false
	}
	permitted := map[string]bool{}
	for _, action := range strings.Split(actions, ",") {
		permitted[action] = true
	}
	seen := map[string]bool{}
	for _, action := range strings.Split(requested, ",") {
		if !permitted[action] || seen[action] {
			return false
		}
		seen[action] = true
	}
	return true
}
