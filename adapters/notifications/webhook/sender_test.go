package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/secrets"
)

func message() notifications.Message {
	return notifications.Message{Schema: "teststudio.notification/v1", Event: "run.finished", Delivery: "delivery-1", Project: "project", Suite: "health", Run: "run-1", State: "completed", Outcome: "timed-out", Counts: notifications.Counts{Total: 1, TimedOut: 1, Completed: 1}, URL: "https://studio.invalid/projects/project/runs/run-1"}
}
func TestWebhookPayloadAndStableIDDoNotIncludeDestinationSecrets(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.Header.Get("X-TestStudio-Delivery-ID") != "delivery-1" {
			t.Error("wrong delivery identity")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil || body["outcome"] != "timed-out" || body["url"] != message().URL {
			t.Error("wrong webhook payload")
		}
		encoded, _ := json.Marshal(body)
		if strings.Contains(string(encoded), "private-token") {
			t.Error("secret entered message body")
		}
		w.WriteHeader(204)
	}))
	defer server.Close()
	value := secrets.Value{Bytes: []byte(server.URL + "/private-token"), Network: networkguard.Loopback}
	r := (&Sender{}).Send(t.Context(), "webhook", value, message())
	if r.HTTPStatus != 204 || r.Error != "" || calls.Load() != 1 {
		t.Fatalf("webhook delivery: %+v", r)
	}
	value.Network = networkguard.Public
	r = (&Sender{}).Send(t.Context(), "webhook", value, message())
	if r.Error == "" || r.Retryable || r.AcceptanceUnknown || calls.Load() != 1 || strings.Contains(r.Error, "private-token") {
		t.Fatalf("public transport crossed local scope: %+v", r)
	}
}
func TestDiscordConfirmsAcceptanceSuppressesMentionsAndHonorsRateLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("wait") != "true" {
			t.Error("Discord did not request confirmation")
		}
		var body struct {
			Content  string `json:"content"`
			Mentions struct {
				Parse []string `json:"parse"`
			} `json:"allowed_mentions"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || !strings.Contains(body.Content, "1 timed out") || body.Mentions.Parse == nil || len(body.Mentions.Parse) != 0 {
			t.Error("invalid Discord message")
		}
		w.Header().Set("Retry-After", "1.25")
		w.WriteHeader(429)
		w.Write([]byte(`{"retry_after":2.5,"global":true,"message":"private-token"}`))
	}))
	defer server.Close()
	r := (&Sender{}).Send(t.Context(), "discord", secrets.Value{Bytes: []byte(server.URL + "/private-token"), Network: networkguard.Loopback}, message())
	if r.HTTPStatus != 429 || !r.Retryable || !r.GlobalRateLimit || r.RetryAfter != 2500*time.Millisecond || strings.Contains(r.Error, "private-token") {
		t.Fatalf("Discord retry contract: %+v", r)
	}
}
func TestWebhookRedirectsAndPermanentFailuresCannotLeakOrRetrySecrets(t *testing.T) {
	var followed atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(204) }))
	defer target.Close()
	for _, status := range []int{302, 400, 401, 404, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", target.URL+"/secret")
				w.WriteHeader(status)
				w.Write([]byte("private-token"))
			}))
			defer server.Close()
			r := (&Sender{}).Send(t.Context(), "webhook", secrets.Value{Bytes: []byte(server.URL + "/private-token"), Network: networkguard.Loopback}, message())
			if r.HTTPStatus != status || r.Retryable != (status >= 500) || strings.Contains(r.Error, "private-token") {
				t.Fatalf("redacted status: %+v", r)
			}
		})
	}
	if followed.Load() != 0 {
		t.Fatal("webhook followed a redirect")
	}
}
