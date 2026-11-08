package workerhttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/recordingworker"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Settings struct {
	Origin                 string `json:"origin"`
	ServerCA               string `json:"serverCAFile"`
	Certificate            string `json:"clientCertificateFile"`
	Key                    string `json:"clientKeyFile"`
	ServerCAEnvironment    string `json:"serverCAEnvironment,omitempty"`
	CertificateEnvironment string `json:"clientCertificateEnvironment,omitempty"`
	KeyEnvironment         string `json:"clientKeyEnvironment,omitempty"`
}

func (s Settings) String() string   { return "private recording worker configuration" }
func (s Settings) GoString() string { return s.String() }

func Load(filename string) (Settings, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Settings{}, errors.New("recording worker configuration is unavailable")
	}
	defer file.Close()
	return decodeSettings(io.LimitReader(file, 65537))
}

func LoadEnvironment(name string) (Settings, error) {
	value, ok := os.LookupEnv(name)
	if !ok || name == "" || len(value) > 65536 {
		return Settings{}, errors.New("recording worker configuration environment is unavailable")
	}
	return decodeSettings(bytes.NewBufferString(value))
}

func decodeSettings(input io.Reader) (Settings, error) {
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	var settings Settings
	var extra any
	if decoder.Decode(&settings) != nil || decoder.Decode(&extra) != io.EOF {
		return Settings{}, errors.New("recording worker configuration is invalid")
	}
	return settings, nil
}

type Client struct {
	Store  *storage.Store
	origin string
	http   *http.Client
}

func New(settings Settings, store *storage.Store) (*Client, error) {
	u, err := url.Parse(settings.Origin)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery || settings.Origin != u.String() {
		return nil, errors.New("recording worker requires an exact private HTTPS origin")
	}
	ca, err := material(settings.ServerCA, settings.ServerCAEnvironment)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, errors.New("recording worker certificate authority is invalid")
	}
	certPEM, err := material(settings.Certificate, settings.CertificateEnvironment)
	if err != nil {
		return nil, err
	}
	keyPEM, err := material(settings.Key, settings.KeyEnvironment)
	if err != nil {
		return nil, err
	}
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, errors.New("recording worker client certificate is unavailable")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{certificate}}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 35 * time.Second, MaxIdleConnsPerHost: 8}
	return &Client{Store: store, origin: settings.Origin, http: &http.Client{Transport: transport, Timeout: 40 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("recording worker redirects are forbidden")
	}}}, nil
}

func (c *Client) call(ctx context.Context, id, operation string, action *recording.Action) (recordingworker.Response, error) {
	row, err := c.Store.Get(ctx, recording.Bucket(), id)
	if err != nil {
		return recordingworker.Response{}, err
	}
	session, err := storage.Decode[recording.Session](row)
	if err != nil {
		return recordingworker.Response{}, err
	}
	data, _ := json.Marshal(recordingworker.Request{Binding: recordingworker.Bind(session), Action: action})
	request, err := http.NewRequestWithContext(ctx, "POST", c.origin+"/internal/recording/"+url.PathEscape(id)+"/"+operation, bytes.NewReader(data))
	if err != nil {
		return recordingworker.Response{}, errors.New("recording worker request is unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return recordingworker.Response{}, errors.New("the private recording worker is unavailable; refresh the browser view before continuing")
	}
	defer response.Body.Close()
	data, err = io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return recordingworker.Response{}, errors.New("the recording worker response exceeded its limit")
	}
	if response.StatusCode != 200 {
		if response.StatusCode == 403 {
			return recordingworker.Response{}, authorization.ErrForbidden
		}
		if response.StatusCode == 404 {
			return recordingworker.Response{}, storage.ErrNotFound
		}
		if response.StatusCode == 409 {
			return recordingworker.Response{}, storage.ErrConflict
		}
		var problem struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &problem) != nil || len(problem.Message) > 512 || problem.Message == "" {
			return recordingworker.Response{}, errors.New("the recording worker rejected this request")
		}
		return recordingworker.Response{}, errors.New(problem.Message)
	}
	var result recordingworker.Response
	if json.Unmarshal(data, &result) != nil {
		return result, errors.New("the recording worker returned an invalid response")
	}
	return result, nil
}

func (c *Client) Start(ctx context.Context, id, address string) error {
	value, err := c.call(ctx, id, "start", nil)
	if err == nil && value.Snapshot.State != "recording" {
		return errors.New("the recording browser did not start; review its retained session before retrying")
	}
	return err
}
func (c *Client) Read(ctx context.Context, id string) (recording.Snapshot, error) {
	value, err := c.call(ctx, id, "read", nil)
	return value.Snapshot, err
}
func (c *Client) Stop(ctx context.Context, id string) (recording.Snapshot, error) {
	value, err := c.call(ctx, id, "stop", nil)
	return value.Snapshot, err
}
func (c *Client) View(ctx context.Context, id string) (recording.View, error) {
	value, err := c.call(ctx, id, "view", nil)
	if err == nil && value.View == nil {
		err = errors.New("the recording browser view is unavailable")
	}
	if value.View == nil {
		return recording.View{}, err
	}
	return *value.View, err
}
func (c *Client) Interact(ctx context.Context, id string, action recording.Action) (recording.View, error) {
	value, err := c.call(ctx, id, "input", &action)
	if err == nil && value.View == nil {
		err = errors.New("the recording browser view is unavailable")
	}
	if value.View == nil {
		return recording.View{}, err
	}
	return *value.View, err
}
func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) CheckReady(ctx context.Context) error {
	request, _ := http.NewRequestWithContext(ctx, "GET", c.origin+"/healthz", nil)
	response, err := c.http.Do(request)
	if err != nil {
		return errors.New("the private recording worker is unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return errors.New("the recording worker has not completed ownership recovery")
	}
	return nil
}
