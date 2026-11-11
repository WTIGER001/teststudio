package gitlab

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type Client struct {
	Secrets secrets.Provider
	BaseURL string
	Network networkguard.Scope
	RootCAs *x509.CertPool
}
type credential struct {
	TokenType    string     `json:"tokenType,omitempty"`
	Token        string     `json:"token"`
	Repositories []string   `json:"repositories"`
	Write        bool       `json:"write,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
}

var segment = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,254}$`)

func validRepository(value string) bool {
	parts := strings.Split(value, "/")
	if len(parts) < 2 || len(value) > 512 || strings.HasSuffix(strings.ToLower(value), ".git") {
		return false
	}
	for _, p := range parts {
		if !segment.MatchString(p) || p == "." || p == ".." {
			return false
		}
	}
	return true
}
func (c *Client) origin() (string, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || u.Path != "" && u.Path != "/" {
		return "", errors.New("GitLab requires an operator-approved HTTPS origin")
	}
	return strings.TrimSuffix(c.BaseURL, "/"), nil
}
func (c *Client) Validate(a sourceprovider.Access, ref string) error {
	if _, err := c.origin(); err != nil {
		return err
	}
	if !domain.ValidID(a.Project) || !validRepository(a.Repository) || !sourceprovider.ValidReference(ref) || a.Credential != "" && !domain.ValidID(a.Credential) || a.RepositoryID != "" && !regexp.MustCompile(`^[1-9][0-9]{0,18}$`).MatchString(a.RepositoryID) {
		return errors.New("GitLab requires a namespace/repository, explicit reference, and scoped credential")
	}
	return nil
}
func (c *Client) access(ctx context.Context, a sourceprovider.Access) (credential, error) {
	var v credential
	if a.Credential == "" && !a.Write {
		return v, nil
	}
	if c.Secrets == nil {
		return v, errors.New("GitLab credential reference is unavailable")
	}
	secret, err := c.Secrets.Resolve(ctx, a.Project, a.Credential, sourceprovider.CredentialPurpose)
	if err != nil || definitions.DecodeJSON(secret.Bytes, &v, 65536, 8, 100) != nil || len(v.Token) < 1 || len(v.Token) > 4096 || len(v.Repositories) < 1 || len(v.Repositories) > 500 {
		return credential{}, errors.New("GitLab credential configuration is invalid")
	}
	for _, b := range v.Token {
		if b <= 32 || b >= 127 {
			return credential{}, errors.New("GitLab token contains invalid header characters")
		}
	}
	allowed := false
	for _, repo := range v.Repositories {
		if !validRepository(repo) {
			return credential{}, errors.New("GitLab approved repository list is invalid")
		}
		allowed = allowed || repo == a.Repository
	}
	if !allowed || a.Write && !v.Write {
		return credential{}, errors.New("GitLab credential does not approve this repository access")
	}
	if v.ExpiresAt != nil && !time.Now().Before(*v.ExpiresAt) {
		return credential{}, errors.New("GitLab credential has expired; replace its approved secret reference")
	}
	return v, nil
}
func (c *Client) request(ctx context.Context, a sourceprovider.Access, method, path string, body io.Reader, limit int) ([]byte, error) {
	origin, err := c.origin()
	if err != nil {
		return nil, err
	}
	v, err := c.access(ctx, a)
	if err != nil {
		return nil, err
	}
	scope := c.Network
	if scope == "" {
		scope = networkguard.Public
	}
	transport := &http.Transport{DialContext: networkguard.Dial(scope), TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: c.RootCAs}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 20 * time.Second, MaxResponseHeaderBytes: 32 << 10, DisableCompression: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, method, origin+"/api/v4"+path, body)
	if err != nil {
		return nil, errors.New("GitLab request is invalid")
	}
	req.Header.Set("User-Agent", "TestStudio sources/1.0")
	if v.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", v.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("GitLab request did not complete")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("GitLab source request failed (HTTP %d)", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, errors.New("GitLab response exceeds its limit or cannot be read")
	}
	return data, nil
}

type repository struct {
	ID         int64  `json:"id"`
	Name       string `json:"path_with_namespace"`
	Visibility string `json:"visibility"`
}

func (c *Client) repository(ctx context.Context, a sourceprovider.Access) (repository, error) {
	var repo repository
	data, err := c.request(ctx, a, "GET", "/projects/"+url.PathEscape(a.Repository), nil, 64<<10)
	if err != nil {
		return repo, err
	}
	if json.Unmarshal(data, &repo) != nil || repo.ID < 1 || repo.Name != a.Repository || a.RepositoryID != "" && a.RepositoryID != strconv.FormatInt(repo.ID, 10) {
		return repository{}, errors.New("GitLab repository identity changed or is invalid")
	}
	return repo, nil
}
func (c *Client) Resolve(ctx context.Context, a sourceprovider.Access, ref string) (sourceprovider.Revision, error) {
	if err := c.Validate(a, ref); err != nil {
		return sourceprovider.Revision{}, err
	}
	repo, err := c.repository(ctx, a)
	if err != nil {
		return sourceprovider.Revision{}, err
	}
	endpoint := "/repository/commits/" + url.PathEscape(ref)
	if kind, name, ok := strings.Cut(ref, "/"); ok {
		if kind == "heads" {
			endpoint = "/repository/branches/" + url.PathEscape(name)
		} else if kind == "tags" {
			endpoint = "/repository/tags/" + url.PathEscape(name)
		}
	}
	data, err := c.request(ctx, a, "GET", "/projects/"+strconv.FormatInt(repo.ID, 10)+endpoint, nil, 64<<10)
	if err != nil {
		return sourceprovider.Revision{}, err
	}
	var commit struct {
		ID     string `json:"id"`
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}
	if json.Unmarshal(data, &commit) != nil {
		return sourceprovider.Revision{}, errors.New("GitLab returned an invalid reference")
	}
	sha := commit.ID
	if !sourceprovider.ValidCommit(ref) {
		sha = commit.Commit.ID
	}
	if !sourceprovider.ValidCommit(sha) || sourceprovider.ValidCommit(ref) && sha != ref {
		return sourceprovider.Revision{}, errors.New("GitLab returned an invalid commit")
	}
	origin, _ := c.origin()
	return sourceprovider.Revision{RepositoryID: strconv.FormatInt(repo.ID, 10), Repository: repo.Name, URL: origin + "/" + repo.Name, Reference: ref, Commit: sha, Private: repo.Visibility != "public"}, nil
}
func (c *Client) Archive(ctx context.Context, a sourceprovider.Access, r sourceprovider.Revision) (sourceprovider.Archive, error) {
	if err := c.Validate(a, r.Commit); err != nil {
		return sourceprovider.Archive{}, err
	}
	origin, _ := c.origin()
	if r.Repository != a.Repository || r.RepositoryID == "" || a.RepositoryID != "" && a.RepositoryID != r.RepositoryID || r.URL != origin+"/"+a.Repository {
		return sourceprovider.Archive{}, errors.New("GitLab snapshot differs from the registered repository")
	}
	a.RepositoryID = r.RepositoryID
	if _, err := c.repository(ctx, a); err != nil {
		return sourceprovider.Archive{}, err
	}
	data, err := c.request(ctx, a, "GET", "/projects/"+r.RepositoryID+"/repository/archive.zip?sha="+r.Commit+"&include_lfs_blobs=false", nil, sourcebundle.CompressedLimit)
	if err != nil {
		return sourceprovider.Archive{}, err
	}
	root, err := sourcebundle.WrappedRoot(data)
	return sourceprovider.Archive{Content: data, Root: root}, err
}
