package oci

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) token(ctx context.Context, repo, challenge string, auth Credential) (string, error) {
	scheme, parameters, ok := strings.Cut(challenge, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", ErrDenied
	}
	_, values, err := mime.ParseMediaType("Bearer;" + strings.ReplaceAll(parameters, ",", ";"))
	if err != nil || values["realm"] != c.tokenURL || values["service"] != c.tokenService || values["scope"] != "" && values["scope"] != "repository:"+repo+":pull" {
		return "", ErrDenied
	}
	for key := range values {
		if key != "realm" && key != "service" && key != "scope" {
			return "", ErrDenied
		}
	}
	location, err := url.Parse(c.tokenURL)
	if err != nil {
		return "", ErrConfiguration
	}
	query := location.Query()
	query.Set("service", c.tokenService)
	query.Set("scope", "repository:"+repo+":pull")
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
