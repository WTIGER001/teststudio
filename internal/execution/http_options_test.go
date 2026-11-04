package execution

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/projects"
)

func TestHTTPHeadKeepsMethodAndAcceptedStatuses(t *testing.T) {
	var heads, gets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			heads.Add(1)
			w.Header().Set("X-Ready", "yes")
			w.WriteHeader(204)
		} else {
			gets.Add(1)
			w.WriteHeader(200)
		}
	}))
	defer server.Close()
	env := projects.Environment{Stage: "local", Endpoints: map[string]string{"api": server.URL}}
	test := definition("/head")
	test.Check.Method = "HEAD"
	test.Check.StatusCodes = []int{204}
	test.Check.Headers = map[string]string{"X-Ready": "yes"}
	r := (&HTTP{}).Execute(t.Context(), "project", test, env)
	if r.State != "passed" || r.HTTP.Status != 204 || heads.Load() != 1 || gets.Load() != 0 {
		t.Fatalf("HEAD contract: %+v", r)
	}
	test.Check.StatusCodes = []int{200}
	r = (&HTTP{}).Execute(t.Context(), "project", test, env)
	if r.FailureKind != "http-status" || heads.Load() != 2 || gets.Load() != 0 {
		t.Fatal("HEAD status mismatch triggered a GET fallback")
	}
}

func TestHTTPResponseLimitIsAnAssertionRatherThanATimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(25 * time.Millisecond); w.WriteHeader(200) }))
	defer server.Close()
	env := projects.Environment{Stage: "local", Endpoints: map[string]string{"api": server.URL}}
	test := definition("/slow")
	test.Check.MaxResponseMs = 1
	r := (&HTTP{}).Execute(t.Context(), "project", test, env)
	if r.State != "failed" || r.FailureKind != "response-time" || r.DurationMs < 20 || r.HTTP.Status != 200 {
		t.Fatalf("response limit: %+v", r)
	}
}
