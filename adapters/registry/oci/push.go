package oci

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func contentDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func matchingDigest(response *http.Response, hash string) bool {
	return response.Header.Get("Docker-Content-Digest") == hash
}

func (c *Client) writeRequest(ctx context.Context, repo, method, location string, body []byte, contentType string, auth *authorization) (*http.Response, error) {
	client := &http.Client{Transport: c.http.Transport, Timeout: time.Minute, CheckRedirect: c.http.CheckRedirect}
	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, location, bytes.NewReader(body))
		if err != nil {
			return nil, ErrInvalid
		}
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		authorize(request, auth.current)
		response, err := client.Do(request)
		if err != nil {
			return nil, ErrUnavailable
		}
		if response.StatusCode == 401 && attempt == 0 && c.tokenURL != "" {
			challenge := response.Header.Get("WWW-Authenticate")
			response.Body.Close()
			token, err := c.tokenActions(ctx, repo, "pull,push", challenge, auth.base)
			if err != nil {
				return nil, errors.Join(ErrDenied, err)
			}
			auth.current = Credential{Token: token}
			continue
		}
		return response, nil
	}
	return nil, ErrDenied
}

func (c *Client) hasBlob(ctx context.Context, repo, hash string, size int, auth *authorization) (bool, error) {
	response, err := c.writeRequest(ctx, repo, "HEAD", c.origin+"/v2/"+repo+"/blobs/"+hash, nil, "", auth)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == 404 {
		return false, nil
	}
	if response.StatusCode != 200 || !matchingDigest(response, hash) || response.ContentLength != int64(size) {
		return false, ErrPublication
	}
	return true, nil
}

func (c *Client) pushBlob(ctx context.Context, repo string, data []byte, auth *authorization, journal UploadJournal) error {
	hash := contentDigest(data)
	exists, err := c.hasBlob(ctx, repo, hash, len(data), auth)
	if err != nil || exists {
		return err
	}
	if err := journal.Opening(hash); err != nil {
		return err
	}
	response, err := c.writeRequest(ctx, repo, "POST", c.origin+"/v2/"+repo+"/blobs/uploads/", nil, "application/octet-stream", auth)
	if err != nil {
		if errors.Is(err, ErrDenied) {
			return errors.Join(err, journal.Completed())
		}
		return err
	}
	response.Body.Close()
	if response.StatusCode != 202 {
		if response.StatusCode >= 400 && response.StatusCode < 500 {
			if err := journal.Completed(); err != nil {
				return err
			}
		}
		return ErrPublication
	}
	location, err := c.uploadLocation(repo, response.Header.Get("Location"))
	if err != nil {
		return err
	}
	if err := journal.Opened(location); err != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return errors.Join(err, c.cancelUpload(cleanup, repo, location, auth))
	}
	parsed, _ := url.Parse(location)
	query := parsed.Query()
	query.Set("digest", hash)
	parsed.RawQuery = query.Encode()
	response, uploadErr := c.writeRequest(ctx, repo, "PUT", parsed.String(), data, "application/octet-stream", auth)
	if uploadErr == nil {
		response.Body.Close()
		if response.StatusCode != 201 || !matchingDigest(response, hash) {
			uploadErr = ErrPublication
		}
	}
	verification, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	exists, err = c.hasBlob(verification, repo, hash, len(data), auth)
	if err != nil || !exists {
		return errors.Join(ErrPublication, uploadErr, err)
	}
	if uploadErr != nil {
		if err := c.cancelUpload(verification, repo, location, auth); err != nil {
			return errors.Join(ErrPublication, err)
		}
	}
	return journal.Completed()
}

func (c *Client) uploadLocation(repo, location string) (string, error) {
	if len(location) < 1 || len(location) > 8192 {
		return "", ErrInvalid
	}
	base, _ := url.Parse(c.origin + "/v2/" + repo + "/blobs/uploads/")
	parsed, err := url.Parse(location)
	if err != nil {
		return "", ErrInvalid
	}
	parsed = base.ResolveReference(parsed)
	prefix := "/v2/" + repo + "/blobs/uploads/"
	if parsed.Scheme+"://"+parsed.Host != c.origin || parsed.User != nil || parsed.Fragment != "" || parsed.Opaque != "" || parsed.RawPath != "" || !strings.HasPrefix(parsed.Path, prefix) || len(parsed.Path) == len(prefix) || strings.Contains(strings.TrimPrefix(parsed.Path, prefix), "/") || strings.ContainsAny(parsed.Path, "\\\x00\r\n") {
		return "", ErrDenied
	}
	if _, err := url.ParseQuery(parsed.RawQuery); err != nil {
		return "", ErrInvalid
	}
	return parsed.String(), nil
}

func (c *Client) cancelUpload(ctx context.Context, repo, location string, auth *authorization) error {
	response, err := c.writeRequest(ctx, repo, "DELETE", location, nil, "", auth)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 202 && response.StatusCode != 204 && response.StatusCode != 404 && response.StatusCode != 410 {
		return ErrPublication
	}
	return nil
}

func (c *Client) CancelSourceUpload(ctx context.Context, runtimeReference, location string) error {
	repo, _, err := c.selected(runtimeReference)
	if err != nil {
		return err
	}
	validated, err := c.uploadLocation(repo, location)
	if err != nil {
		return err
	}
	value, err := credential(ctx, c.credentials)
	if err != nil {
		return err
	}
	return c.cancelUpload(ctx, repo, validated, &authorization{base: value, current: value})
}
