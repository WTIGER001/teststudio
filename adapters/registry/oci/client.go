package oci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var digest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var repository = regexp.MustCompile(`^[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*)*$`)
var runtimeLabel = regexp.MustCompile(`^(?:go|playwright)[A-Za-z0-9._+-]{1,120}$`)

func New(options Options) (*Client, error) {
	origin, err := url.Parse(options.Origin)
	if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.ForceQuery || origin.Fragment != "" || origin.Opaque != "" || strings.ContainsAny(origin.Host, "\\\r\n\x00") || !repository.MatchString(options.RepositoryPrefix) || len(options.RepositoryPrefix) > 255 || (options.Architecture != "amd64" && options.Architecture != "arm64") {
		return nil, ErrConfiguration
	}
	if options.TokenURL != "" {
		token, err := url.Parse(options.TokenURL)
		if err != nil || token.Scheme != "https" || token.Hostname() == "" || token.User != nil || token.RawQuery != "" || token.ForceQuery || token.Fragment != "" || token.Opaque != "" || len(options.TokenService) > 255 || strings.ContainsAny(options.TokenService, "\r\n\x00,;\" ") {
			return nil, ErrConfiguration
		}
	} else if options.TokenService != "" {
		return nil, ErrConfiguration
	}
	redirects := map[string]bool{}
	if len(options.BlobRedirectOrigins) > 16 {
		return nil, ErrConfiguration
	}
	for _, value := range options.BlobRedirectOrigins {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Opaque != "" {
			return nil, ErrConfiguration
		}
		redirects[value] = true
	}
	transport := options.Transport
	if transport == nil {
		transport = &http.Transport{TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second, IdleConnTimeout: 30 * time.Second}
	}
	return &Client{origin: options.Origin, repositoryPrefix: options.RepositoryPrefix, architecture: options.Architecture, tokenURL: options.TokenURL, tokenService: options.TokenService, credentials: options.Credentials, blobRedirectOrigins: redirects, http: &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) selected(reference string) (string, string, error) {
	name, hash, ok := strings.Cut(reference, "@")
	if !ok || !digest.MatchString(hash) || !strings.HasPrefix(name, strings.TrimPrefix(c.origin, "https://")+"/") {
		return "", "", ErrInvalid
	}
	repo := strings.TrimPrefix(name, strings.TrimPrefix(c.origin, "https://")+"/")
	if len(repo) > 255 || !repository.MatchString(repo) || (repo != c.repositoryPrefix && !strings.HasPrefix(repo, c.repositoryPrefix+"/")) {
		return "", "", ErrInvalid
	}
	return repo, hash, nil
}

func credential(ctx context.Context, source func(context.Context) (Credential, error)) (Credential, error) {
	if source == nil {
		return Credential{}, nil
	}
	value, err := source(ctx)
	if err != nil || len(value.Username) > 4096 || len(value.Password) > 65536 || len(value.Token) > 65536 || strings.ContainsAny(value.Username+value.Password+value.Token, "\x00\r\n") || value.Token != "" && (value.Username != "" || value.Password != "") {
		return Credential{}, ErrDenied
	}
	return value, nil
}

func authorize(request *http.Request, value Credential) {
	if value.Token != "" {
		request.Header.Set("Authorization", "Bearer "+value.Token)
	} else if value.Username != "" || value.Password != "" {
		request.SetBasicAuth(value.Username, value.Password)
	}
}

func readResponse(response *http.Response, limit int64) ([]byte, error) {
	if response.ContentLength > limit {
		return nil, ErrInvalid
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, ErrUnavailable
	}
	if int64(len(data)) > limit {
		return nil, ErrInvalid
	}
	return data, nil
}

func (c *Client) fetch(ctx context.Context, repo, kind, hash, accept string, auth *authorization) ([]byte, string, error) {
	location := c.origin + "/v2/" + repo + "/" + kind + "/" + hash
	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if err != nil {
			return nil, "", ErrInvalid
		}
		request.Header.Set("Accept", accept)
		authorize(request, auth.current)
		response, err := c.http.Do(request)
		if err != nil {
			return nil, "", ErrUnavailable
		}
		if response.StatusCode == http.StatusUnauthorized && attempt == 0 && c.tokenURL != "" {
			challenge := response.Header.Get("WWW-Authenticate")
			response.Body.Close()
			token, err := c.token(ctx, repo, challenge, auth.base)
			if err != nil {
				return nil, "", err
			}
			auth.current = Credential{Token: token}
			continue
		}
		if kind == "blobs" {
			response, err = c.redirectBlob(ctx, response)
			if err != nil {
				return nil, "", err
			}
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			if response.StatusCode == 401 || response.StatusCode == 403 {
				return nil, "", ErrDenied
			}
			return nil, "", ErrUnavailable
		}
		data, readErr := readResponse(response, metadataLimit)
		response.Body.Close()
		if readErr != nil {
			return nil, "", readErr
		}
		sum := sha256.Sum256(data)
		if "sha256:"+hex.EncodeToString(sum[:]) != hash {
			return nil, "", ErrInvalid
		}
		if header := response.Header.Get("Docker-Content-Digest"); header != "" && header != hash {
			return nil, "", ErrInvalid
		}
		typeName, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
		if err != nil {
			return nil, "", ErrInvalid
		}
		return data, typeName, nil
	}
	return nil, "", ErrDenied
}

func decode(data []byte, output any) error {
	if json.Unmarshal(data, output) != nil {
		return ErrInvalid
	}
	return nil
}

func (c *Client) ValidateReference(reference string) error {
	_, _, err := c.selected(reference)
	return err
}
