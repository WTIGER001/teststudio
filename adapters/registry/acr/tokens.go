package acr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/definitions"
)

type cachedToken struct {
	value string
	until time.Time
}

type authentication struct {
	registry, tenant, prefix string
	publish                  bool
	credential               azcore.TokenCredential
	now                      func() time.Time
	slots                    chan struct{}
	closed                   atomic.Bool
	identity, refresh        string
	refreshUntil             time.Time
	entries                  map[string]cachedToken
	http                     *http.Client
}

func (a *authentication) String() string   { return "private ACR authentication" }
func (a *authentication) GoString() string { return a.String() }

func (a *authentication) clear() {
	a.identity, a.refresh = "", ""
	a.refreshUntil = time.Time{}
	a.entries = map[string]cachedToken{}
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (a *authentication) Token(ctx context.Context, repo, actions string) (string, error) {
	if !oci.RepositoryWithinPrefix(repo, a.prefix) || actions != "pull" && (!a.publish || actions != "pull,push") {
		return "", oci.ErrDenied
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	select {
	case a.slots <- struct{}{}:
		defer func() { <-a.slots }()
	case <-ctx.Done():
		return "", oci.ErrUnavailable
	}
	if a.closed.Load() || ctx.Err() != nil {
		return "", oci.ErrUnavailable
	}
	current, err := a.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://containerregistry.azure.net/.default"}, TenantID: a.tenant})
	now := a.now()
	if err != nil || current.Token == "" || len(current.Token) > 65536 || strings.ContainsAny(current.Token, "\r\n\x00") || !current.ExpiresOn.After(now.Add(30*time.Second)) || ctx.Err() != nil {
		a.clear()
		return "", oci.ErrUnavailable
	}
	hash := sha256.Sum256([]byte(current.Token))
	identity := hex.EncodeToString(hash[:])
	if identity != a.identity || !now.Before(a.refreshUntil) {
		a.clear()
	}
	if a.refresh == "" {
		value, until, err := a.post(ctx, "exchange", url.Values{"grant_type": {"access_token"}, "service": {a.registry}, "tenant": {a.tenant}, "access_token": {current.Token}}, "refresh_token")
		if err != nil {
			a.clear()
			return "", err
		}
		a.identity, a.refresh = identity, value
		a.refreshUntil = minTime(until, minTime(current.ExpiresOn.Add(-30*time.Second), now.Add(5*time.Minute)))
	}
	a.refreshUntil = minTime(a.refreshUntil, current.ExpiresOn.Add(-30*time.Second))
	key := repo + ":" + actions
	if cached, ok := a.entries[key]; ok && now.Before(cached.until) && now.Before(a.refreshUntil) {
		return cached.value, nil
	}
	value, until, err := a.post(ctx, "token", url.Values{"grant_type": {"refresh_token"}, "service": {a.registry}, "scope": {"repository:" + repo + ":" + actions}, "refresh_token": {a.refresh}}, "access_token")
	if err != nil {
		a.clear()
		return "", err
	}
	until = minTime(until, a.refreshUntil)
	if !until.After(a.now()) {
		a.clear()
		return "", oci.ErrUnavailable
	}
	if len(a.entries) >= 32 {
		oldest := ""
		var at time.Time
		for key, entry := range a.entries {
			if oldest == "" || entry.until.Before(at) {
				oldest, at = key, entry.until
			}
		}
		delete(a.entries, oldest)
	}
	a.entries[key] = cachedToken{value: value, until: until}
	return value, nil
}

func (a *authentication) post(ctx context.Context, endpoint string, values url.Values, field string) (string, time.Time, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+a.registry+"/oauth2/"+endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return "", time.Time{}, oci.ErrInvalid
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := a.http.Do(request)
	if err != nil {
		return "", time.Time{}, oci.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return "", time.Time{}, oci.ErrDenied
	}
	if response.StatusCode != 200 {
		return "", time.Time{}, oci.ErrUnavailable
	}
	if response.ContentLength > 128<<10 {
		return "", time.Time{}, oci.ErrInvalid
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (128<<10)+1))
	if err != nil {
		return "", time.Time{}, oci.ErrUnavailable
	}
	var body map[string]json.RawMessage
	if definitions.DecodeJSON(data, &body, 128<<10, 4, 32) != nil {
		return "", time.Time{}, oci.ErrInvalid
	}
	var value string
	if json.Unmarshal(body[field], &value) != nil {
		return "", time.Time{}, oci.ErrInvalid
	}
	until, err := tokenExpiry(value, a.now())
	return value, until, err
}
