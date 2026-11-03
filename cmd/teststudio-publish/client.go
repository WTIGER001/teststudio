package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/wtiger001/teststudio/internal/definitions"
)

type loginFile struct {
	Schema    string    `json:"schemaVersion"`
	Origin    string    `json:"origin"`
	Cookie    string    `json:"cookie"`
	CSRF      string    `json:"csrfToken"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type apiClient struct {
	client  *http.Client
	session loginFile
}
type responseError struct {
	status  int
	message string
}

func (e responseError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.status, e.message) }

var sessionToken = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func serverOrigin(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("use the installation's exact public origin without paths or credentials")
	}
	if parsed.Scheme != "https" && (parsed.Scheme != "http" || !loopback(parsed.Hostname())) {
		return nil, errors.New("server connections require HTTPS outside loopback")
	}
	parsed.Path = ""
	return parsed, nil
}
func loopback(host string) bool {
	ip := net.ParseIP(host)
	return host == "localhost" || ip != nil && ip.IsLoopback()
}
func sessionPath() string {
	root, err := os.UserConfigDir()
	if err != nil {
		return ".teststudio-session.json"
	}
	return filepath.Join(root, "TestStudio", "publisher-session.json")
}
func newClient(session loginFile) *apiClient {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	origin, _ := url.Parse(session.Origin)
	if session.Cookie != "" {
		jar.SetCookies(origin, []*http.Cookie{{Name: "teststudio_session", Value: session.Cookie, Path: "/", Secure: origin.Scheme == "https"}})
	}
	return &apiClient{client: client, session: session}
}
func readClient(filename string) (*apiClient, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("session file must be a private regular file with mode 0600")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 8193))
	if err != nil {
		return nil, err
	}
	var session loginFile
	if err := definitions.DecodeJSON(data, &session, 8192, 8, 100); err != nil {
		return nil, err
	}
	if session.Schema != "teststudio.publisher-session/v1" || !sessionToken.MatchString(session.Cookie) || len(session.CSRF) < 1 || len(session.CSRF) > 256 {
		return nil, errors.New("invalid publisher session file")
	}
	origin, err := serverOrigin(session.Origin)
	if err != nil {
		return nil, err
	}
	session.Origin = origin.String()
	if !time.Now().Before(session.ExpiresAt) {
		return nil, errors.New("CLI session expired; sign in again")
	}
	return newClient(session), nil
}
func (c *apiClient) request(ctx context.Context, method, path string, body any, revision int64, key string, result any) error {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.session.Origin+path, payload)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if method != "GET" {
		request.Header.Set("Origin", c.session.Origin)
		if c.session.CSRF != "" {
			request.Header.Set("X-CSRF-Token", c.session.CSRF)
		}
	}
	if revision > 0 {
		request.Header.Set("If-Match", fmt.Sprintf("\"%d\"", revision))
	}
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return errors.New("server request failed; inspect stored state before retrying a change")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (16<<20)+1))
	if err != nil || len(data) > 16<<20 {
		return errors.New("server response is unavailable or exceeds its byte limit")
	}
	if response.StatusCode != http.StatusOK {
		var problem struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &problem)
		if problem.Message == "" {
			problem.Message = http.StatusText(response.StatusCode)
		}
		return responseError{status: response.StatusCode, message: problem.Message}
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return errors.New("server response is invalid; inspect stored state before retrying")
		}
	}
	return nil
}
func saveLogin(filename string, session loginFile) error {
	if info, err := os.Lstat(filename); err == nil && !info.Mode().IsRegular() {
		return errors.New("session path must be a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	directory := filepath.Dir(filename)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".session-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), filename)
}
