package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
)

var ErrAuthority = errors.New("recording authority ended; captured source remains on this computer")
var ErrTransport = errors.New("the TestStudio connection failed; the owned recorder stops and retains local source")

type Remote interface {
	Pair(context.Context, string, recording.PairRequest) (recording.CompanionPacket, error)
	Control(context.Context, string, string, string) (recording.CompanionControl, error)
	Submit(context.Context, string, string, string, recording.CompanionSnapshot) (recording.CompanionControl, error)
}

type Client struct {
	origin string
	http   *http.Client
}

func Origin(value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.Path != "" && u.Path != "/" || strings.ContainsAny(value, "\r\n\t") {
		return "", errors.New("use the TestStudio HTTPS server origin without a path, credentials, or query")
	}
	address, _ := netip.ParseAddr(u.Hostname())
	if u.Scheme != "https" && (u.Scheme != "http" || !address.IsLoopback()) {
		return "", errors.New("recording requires HTTPS; plain HTTP is allowed only for a loopback server")
	}
	u.Path = ""
	return u.String(), nil
}

func NewClient(origin string) (*Client, error) {
	canonical, err := Origin(origin)
	if err != nil {
		return nil, err
	}
	return &Client{origin: canonical, http: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) post(ctx context.Context, id, action, token string, input, output any) error {
	if !domain.ValidID(id) {
		return ErrAuthority
	}
	data, err := json.Marshal(input)
	if err != nil {
		return ErrTransport
	}
	r, err := http.NewRequestWithContext(ctx, "POST", c.origin+"/api/v1/recording-companion/sessions/"+id+"/"+action, bytes.NewReader(data))
	if err != nil {
		return ErrTransport
	}
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.http.Do(r)
	if err != nil {
		return ErrTransport
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return ErrAuthority
	}
	if response.StatusCode != 200 {
		return ErrTransport
	}
	data, err = io.ReadAll(io.LimitReader(response.Body, recording.ContentLimit+32769))
	if err != nil || len(data) > recording.ContentLimit+32768 {
		return ErrTransport
	}
	if err := json.Unmarshal(data, output); err != nil {
		return ErrTransport
	}
	return nil
}

func (c *Client) Pair(ctx context.Context, id string, input recording.PairRequest) (recording.CompanionPacket, error) {
	var packet recording.CompanionPacket
	err := c.post(ctx, id, "pair", "", input, &packet)
	return packet, err
}

func (c *Client) Control(ctx context.Context, id, instance, token string) (recording.CompanionControl, error) {
	var control recording.CompanionControl
	err := c.post(ctx, id, "control", token, map[string]string{"instance": instance}, &control)
	return control, err
}

func (c *Client) Submit(ctx context.Context, id, instance, token string, snapshot recording.CompanionSnapshot) (recording.CompanionControl, error) {
	var control recording.CompanionControl
	err := c.post(ctx, id, "snapshots", token, struct {
		Instance string                      `json:"instance"`
		Snapshot recording.CompanionSnapshot `json:"snapshot"`
	}{instance, snapshot}, &control)
	return control, err
}
