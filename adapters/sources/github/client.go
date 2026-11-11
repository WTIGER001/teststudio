package github

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type Client struct {
	Secrets        secrets.Provider
	APIBase        string
	DownloadOrigin string
	Network        networkguard.Scope
	RootCAs        *x509.CertPool
}

var repositoryName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9_.-]{1,100}$`)
var repositoryID = regexp.MustCompile(`^[1-9][0-9]{0,18}$`)

func (c *Client) Validate(access sourceprovider.Access, reference string) error {
	if !domain.ValidID(access.Project) || !repositoryName.MatchString(access.Repository) || strings.HasSuffix(strings.ToLower(access.Repository), ".git") || strings.HasSuffix(access.Repository, "/.") || strings.HasSuffix(access.Repository, "/..") || !sourceprovider.ValidReference(reference) || (access.Credential != "" && !domain.ValidID(access.Credential)) || (access.RepositoryID != "" && !repositoryID.MatchString(access.RepositoryID)) {
		return errors.New("GitHub requires owner/repository, a heads/branch, tags/tag, or exact commit, and valid scoped references")
	}
	return nil
}
func (c *Client) Resolve(ctx context.Context, access sourceprovider.Access, reference string) (sourceprovider.Revision, error) {
	var revision sourceprovider.Revision
	if err := c.Validate(access, reference); err != nil {
		return revision, err
	}
	session, err := c.session(ctx, access)
	if err != nil {
		return revision, err
	}
	defer session.close()
	repository, err := session.repository(ctx, access)
	if err != nil {
		return revision, err
	}
	response, err := session.request(ctx, "GET", session.api+"/repos/"+access.Repository+"/commits/"+url.PathEscape(reference), nil, session.token, "application/vnd.github.sha")
	if err != nil {
		return revision, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return revision, statusError("reference resolution", response.StatusCode)
	}
	data, err := readBounded(response.Body, 256)
	if err != nil {
		return revision, errors.New("GitHub returned an invalid commit identifier")
	}
	commit := strings.TrimSpace(string(data))
	if !sourceprovider.ValidCommit(commit) || (sourceprovider.ValidCommit(reference) && reference != commit) {
		return revision, errors.New("GitHub returned an invalid commit identifier")
	}
	return sourceprovider.Revision{RepositoryID: strconv.FormatInt(repository.ID, 10), Repository: repository.Name, URL: "https://github.com/" + repository.Name, Reference: reference, Commit: commit, Private: repository.Private}, nil
}
func (c *Client) Archive(ctx context.Context, access sourceprovider.Access, revision sourceprovider.Revision) (sourceprovider.Archive, error) {
	var result sourceprovider.Archive
	if err := c.Validate(access, revision.Commit); err != nil {
		return result, err
	}
	if !strings.EqualFold(revision.Repository, access.Repository) || revision.RepositoryID == "" || (access.RepositoryID != "" && access.RepositoryID != revision.RepositoryID) || revision.URL != "https://github.com/"+revision.Repository {
		return result, errors.New("resolved source does not match the registered repository")
	}
	session, err := c.session(ctx, access)
	if err != nil {
		return result, err
	}
	defer session.close()
	pinned := access
	pinned.RepositoryID = revision.RepositoryID
	if _, err := session.repository(ctx, pinned); err != nil {
		return result, err
	}
	response, err := session.request(ctx, "GET", session.api+"/repos/"+access.Repository+"/zipball/"+revision.Commit, nil, session.token, "application/vnd.github+json")
	if err != nil {
		return result, err
	}
	if response.StatusCode == http.StatusFound {
		location := response.Header.Get("Location")
		response.Body.Close()
		destination, err := url.Parse(location)
		expected, parseErr := url.Parse(session.download)
		if err != nil || parseErr != nil || destination.Scheme != "https" || destination.User != nil || destination.Fragment != "" || !strings.EqualFold(destination.Host, expected.Host) || destination.RawPath != "" || !strings.HasPrefix(strings.ToLower(destination.Path), "/"+strings.ToLower(revision.Repository)+"/") || path.Base(destination.Path) != revision.Commit {
			return result, errors.New("GitHub archive redirect does not match its approved origin and commit")
		}
		response, err = session.request(ctx, "GET", destination.String(), nil, "", "application/zip")
		if err != nil {
			return result, err
		}
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return result, statusError("archive download", response.StatusCode)
	}
	if response.ContentLength > sourcebundle.CompressedLimit {
		return result, errors.New("GitHub source archive exceeds its 32 MiB limit")
	}
	result.Content, err = readBounded(response.Body, sourcebundle.CompressedLimit)
	if err != nil {
		return sourceprovider.Archive{}, errors.New("GitHub source archive exceeds its limit or cannot be read")
	}
	result.Root, err = sourcebundle.WrappedRoot(result.Content)
	if err != nil {
		return sourceprovider.Archive{}, err
	}
	return result, nil
}

type session struct {
	client               *http.Client
	transport            *http.Transport
	api, download, token string
	cancel               context.CancelFunc
	ctx                  context.Context
}

func (s *session) close() { s.transport.CloseIdleConnections(); s.cancel() }
func (c *Client) session(ctx context.Context, access sourceprovider.Access) (*session, error) {
	api := c.APIBase
	if api == "" {
		api = "https://api.github.com"
	}
	download := c.DownloadOrigin
	if download == "" {
		download = "https://codeload.github.com"
	}
	for _, value := range []string{api, download} {
		u, err := url.Parse(value)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || (u.Path != "" && path.Clean(u.Path) != u.Path) {
			return nil, errors.New("GitHub provider origins must use configured HTTPS endpoints without credentials")
		}
	}
	parsedDownload, _ := url.Parse(download)
	if parsedDownload.Path != "" && parsedDownload.Path != "/" {
		return nil, errors.New("GitHub download origin must not contain a path")
	}
	scope := c.Network
	if scope == "" {
		scope = networkguard.Public
	}
	transport := &http.Transport{DialContext: networkguard.Dial(scope), TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: c.RootCAs}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 20 * time.Second, MaxResponseHeaderBytes: 32 << 10, DisableCompression: true}
	deadline, cancel := context.WithTimeout(ctx, 2*time.Minute)
	result := &session{api: strings.TrimSuffix(api, "/"), download: strings.TrimSuffix(download, "/"), transport: transport, cancel: cancel, ctx: deadline, client: &http.Client{Transport: transport, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	if access.Credential != "" {
		token, err := c.sourceToken(deadline, result, access)
		if err != nil {
			result.close()
			return nil, err
		}
		result.token = token
	}
	return result, nil
}
func (s *session) request(ctx context.Context, method, destination string, body any, token, accept string) (*http.Response, error) {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, errors.New("GitHub request could not be encoded")
		}
		payload = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(s.ctx, method, destination, payload)
	if err != nil {
		return nil, errors.New("GitHub request could not be constructed")
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "TestStudio sources/1.0")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if strings.HasPrefix(destination, s.api+"/") {
		request.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, errors.New("GitHub request did not complete")
	}
	return response, nil
}

type repository struct {
	ID      int64  `json:"id"`
	Name    string `json:"full_name"`
	Private bool   `json:"private"`
}

func (s *session) repository(ctx context.Context, access sourceprovider.Access) (repository, error) {
	var result repository
	response, err := s.request(ctx, "GET", s.api+"/repos/"+access.Repository, nil, s.token, "application/vnd.github+json")
	if err != nil {
		return result, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return result, statusError("repository access", response.StatusCode)
	}
	data, err := readBounded(response.Body, 1<<20)
	if err != nil || json.Unmarshal(data, &result) != nil || result.ID <= 0 || !strings.EqualFold(result.Name, access.Repository) {
		return repository{}, errors.New("GitHub returned invalid repository identity metadata")
	}
	if access.RepositoryID != "" && access.RepositoryID != strconv.FormatInt(result.ID, 10) {
		return repository{}, errors.New("GitHub repository identity changed; register a new source explicitly")
	}
	return result, nil
}
func statusError(operation string, status int) error {
	return fmt.Errorf("GitHub %s returned HTTP %d", operation, status)
}
func readBounded(reader io.Reader, limit int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, errors.New("provider response exceeded its read limit")
	}
	return data, nil
}
