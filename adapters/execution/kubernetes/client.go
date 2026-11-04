package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (p *Provider) request(ctx context.Context, method, path string, query url.Values, input, output any) error {
	var data []byte
	if input != nil {
		var err error
		data, err = json.Marshal(input)
		if err != nil || len(data) > 1<<20 {
			return hostedexecution.ErrConfiguration
		}
	}
	token, err := p.token(ctx)
	if err != nil || len(token) < 1 || len(token) > 64<<10 || strings.ContainsAny(token, "\x00\r\n") {
		return hostedexecution.ErrUnavailable
	}
	if query == nil {
		query = url.Values{}
	}
	query.Set("timeout", "4s")
	request, err := http.NewRequestWithContext(ctx, method, p.config.APIOrigin+path+"?"+query.Encode(), bytes.NewReader(data))
	if err != nil {
		return hostedexecution.ErrConfiguration
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	if method == http.MethodPatch {
		request.Header.Set("Content-Type", "application/merge-patch+json")
	} else if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := p.http.Do(request)
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		switch response.StatusCode {
		case 404:
			return errAbsent
		case 409:
			return errConflict
		case 400, 401, 403, 405, 413, 415, 422:
			return errRejected
		default:
			return hostedexecution.ErrUnavailable
		}
	}
	if response.ContentLength > 4<<20 {
		return hostedexecution.ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil || len(body) > 4<<20 {
		return hostedexecution.ErrUnavailable
	}
	if output != nil && json.Unmarshal(body, output) != nil {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func (p *Provider) podsPath() string           { return "/api/v1/namespaces/" + p.config.Namespace + "/pods" }
func (p *Provider) podPath(name string) string { return p.podsPath() + "/" + name }
