package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

type Client struct {
	origin, execution, instance, token string
	http                               *http.Client
}

func New(origin, executionID, instance, token string, transport http.RoundTripper) (*Client, error) {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || !domain.ValidID(executionID) || !domain.ValidID(instance) || len(token) != 43 {
		return nil, errors.New("worker transport requires an HTTPS origin and scoped execution credentials")
	}
	if transport == nil {
		transport = &http.Transport{}
	}
	return &Client{origin: origin, execution: executionID, instance: instance, token: token, http: &http.Client{Transport: transport, Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) request(ctx context.Context, method, path string, reader io.Reader, output any, limit int64) error {
	req, err := http.NewRequestWithContext(ctx, method, c.origin+"/internal/worker/v1/executions/"+c.execution+path, reader)
	if err != nil {
		return workerprotocol.ErrInvalid
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-TestStudio-Worker-Instance", c.instance)
	req.Header.Set("Content-Type", "application/json")
	if strings.HasPrefix(path, "/attachments/") {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	response, err := c.http.Do(req)
	if err != nil {
		return workerprotocol.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		switch response.StatusCode {
		case 400:
			return workerprotocol.ErrInvalid
		case 404:
			return workerprotocol.ErrDenied
		case 507:
			return workerprotocol.ErrQuota
		case 409:
			return workerprotocol.ErrConflict
		default:
			return workerprotocol.ErrUnavailable
		}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(data)) > limit {
		return workerprotocol.ErrInvalid
	}
	if output != nil {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if decoder.Decode(output) != nil || decoder.Decode(new(any)) != io.EOF {
			return workerprotocol.ErrInvalid
		}
	}
	return nil
}

func (c *Client) Input(ctx context.Context) (workerprotocol.Packet, error) {
	var packet workerprotocol.Packet
	err := c.request(ctx, http.MethodGet, "/input", nil, &packet, execution.MaxWorkItemBytes+8192)
	if err == nil && (packet.Execution != c.execution || !time.Now().Before(packet.Deadline) || len(packet.Input) > execution.MaxWorkItemBytes) {
		err = workerprotocol.ErrInvalid
	}
	return packet, err
}

func (c *Client) Started(ctx context.Context, at time.Time) error {
	data, err := json.Marshal(struct {
		At time.Time `json:"at"`
	}{at})
	if err != nil {
		return workerprotocol.ErrInvalid
	}
	return c.request(ctx, http.MethodPost, "/started", bytes.NewReader(data), nil, 1024)
}

func (c *Client) Submit(ctx context.Context, submission workerprotocol.Submission) error {
	data, err := json.Marshal(submission)
	limit := workerprotocol.MaxTestSubmissionBytes
	if submission.Discovery != nil {
		limit = workerprotocol.MaxSubmissionBytes
	}
	if err != nil || len(data) > limit {
		return workerprotocol.ErrInvalid
	}
	return c.request(ctx, http.MethodPut, "/report", bytes.NewReader(data), nil, 1024)
}

func (c *Client) Upload(ctx context.Context, path string, reader io.Reader) error {
	if path == "" || path == "." || path == ".." || len(path) > 255 || strings.ContainsAny(path, "/\\\x00\r\n") {
		return workerprotocol.ErrInvalid
	}
	return c.request(ctx, http.MethodPut, "/attachments/"+url.PathEscape(path), reader, nil, 1024)
}

func (c *Client) String() string   { return "worker client: " + c.execution }
func (c *Client) GoString() string { return c.String() }
