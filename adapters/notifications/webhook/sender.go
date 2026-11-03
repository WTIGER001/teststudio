package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/secrets"
)

type Sender struct{ RootCAs *x509.CertPool }

func (s *Sender) Send(ctx context.Context, kind string, secret secrets.Value, message notifications.Message) notifications.Outcome {
	fail := func(reason string, retry, unknown bool) notifications.Outcome {
		return notifications.Outcome{Error: reason, Retryable: retry, AcceptanceUnknown: unknown}
	}
	value := strings.TrimSpace(string(secret.Bytes))
	if len(value) == 0 || len(value) > 4096 {
		return fail("Webhook secret has an invalid URL length.", false, false)
	}
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (secret.Network == networkguard.Loopback || secret.Network == networkguard.Private))) {
		return fail("Webhook secret is not a permitted HTTP or HTTPS destination.", false, false)
	}
	var body []byte
	switch kind {
	case "webhook":
		body, err = json.Marshal(message)
	case "discord":
		q := u.Query()
		q.Set("wait", "true")
		u.RawQuery = q.Encode()
		text := fmt.Sprintf("TestStudio · %s / %s · %s\n%d passed · %d failed · %d timed out · %d flaky · %d skipped\nRun: %s · Evidence: %s\n<%s>", message.Project, message.Suite, message.Outcome, message.Counts.Passed, message.Counts.Failed, message.Counts.TimedOut, message.Counts.Flaky, message.Counts.Skipped, message.State, message.Evidence, message.URL)
		if utf8.RuneCountInString(text) > 2000 {
			return fail("Notification exceeds the Discord text limit.", false, false)
		}
		body, err = json.Marshal(map[string]any{"content": text, "allowed_mentions": map[string]any{"parse": []string{}}, "flags": 4})
	default:
		return fail("Notification transport is unavailable.", false, false)
	}
	if err != nil || len(body) > 16384 {
		return fail("Notification body exceeds its contract.", false, false)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	transport := &http.Transport{DialContext: networkguard.Dial(secret.Network), TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: s.RootCAs}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 8 * time.Second, MaxResponseHeaderBytes: 16 << 10}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return fail("Webhook request could not be constructed.", false, false)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "TestStudio/1.0 notifications")
	request.Header.Set("X-TestStudio-Delivery-ID", message.Delivery)
	response, err := client.Do(request)
	if err != nil {
		if errors.Is(err, networkguard.ErrScope) {
			return fail("Webhook destination is outside its permitted network scope.", false, false)
		}
		return fail("Webhook request did not complete. Remote acceptance is unknown.", true, true)
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return notifications.Outcome{HTTPStatus: response.StatusCode}
	}
	retry := response.StatusCode == 408 || response.StatusCode == 425 || response.StatusCode == 429 || response.StatusCode >= 500
	result := notifications.Outcome{HTTPStatus: response.StatusCode, Retryable: retry, Error: fmt.Sprintf("Webhook returned HTTP %d.", response.StatusCode)}
	if retry {
		result.RetryAfter = retryAfter(response.Header.Get("Retry-After"), time.Now())
	}
	if kind == "discord" && response.StatusCode == 429 {
		result.GlobalRateLimit = strings.EqualFold(response.Header.Get("X-RateLimit-Global"), "true")
		data, err := io.ReadAll(io.LimitReader(response.Body, 4097))
		var limit struct {
			Seconds float64 `json:"retry_after"`
			Global  bool    `json:"global"`
		}
		if err == nil && len(data) <= 4096 && json.Unmarshal(data, &limit) == nil {
			result.RetryAfter = max(result.RetryAfter, seconds(limit.Seconds))
			result.GlobalRateLimit = result.GlobalRateLimit || limit.Global
		}
	}
	return result
}
func seconds(n float64) time.Duration {
	if math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 {
		return 0
	}
	return time.Duration(min(n, 3600) * float64(time.Second))
}
func retryAfter(value string, now time.Time) time.Duration {
	if n, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
		return seconds(n)
	}
	if at, err := http.ParseTime(value); err == nil {
		return min(time.Hour, max(0, at.Sub(now)))
	}
	return 0
}
