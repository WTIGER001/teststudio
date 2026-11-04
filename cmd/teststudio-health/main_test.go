package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthProbeRequiresTheBoundedTestStudioReadyResponse(t *testing.T) {
	for _, item := range []struct {
		status int
		body   string
		ready  bool
	}{{200, `{"status":"up","service":"teststudio"}`, true}, {503, `{"status":"up","service":"teststudio"}`, false}, {200, `{"status":"down","service":"teststudio"}`, false}, {200, "<html>Proxy login</html>", false}, {200, strings.Repeat(" ", 4097), false}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(item.status)
			_, _ = w.Write([]byte(item.body))
		}))
		err := run([]string{"--url", server.URL})
		server.Close()
		if (err == nil) != item.ready {
			t.Fatal(item.status, item.ready, err)
		}
	}
}
func TestHealthProbeRejectsRedirectsAndRetainsItsDeadline(t *testing.T) {
	var followed atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed.Add(1) }))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 302) }))
	defer redirect.Close()
	if run([]string{"--url", redirect.URL}) == nil || followed.Load() != 0 {
		t.Fatal("probe followed an unreviewed redirect")
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	started := time.Now()
	if run([]string{"--url", slow.URL, "--timeout=10ms"}) == nil || time.Since(started) > time.Second {
		t.Fatal("probe did not retain its deadline")
	}
}
