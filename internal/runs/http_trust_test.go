package runs_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httptrust "github.com/wtiger001/teststudio/adapters/httptrust/filesystem"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestPrivateHTTPSManualAndScheduledRunsRetainTLSHistoryAndFreezeTrust(t *testing.T) {
	s, p, _ := setup(t, nil)
	authority := testkit.NewCertificateAuthority(t, "Owned scheduled HTTPS authority")
	var hits atomic.Int32
	server := testkit.NewTLSServer(t, authority, "127.0.0.1", 72*time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service-Ready", "yes")
		_, _ = w.Write([]byte(`{"status":"up","items":[{"id":7}]}`))
	}))
	file := filepath.Join(t.TempDir(), "local-ca.pem")
	if err := os.WriteFile(file, authority.PEM, 0600); err != nil {
		t.Fatal(err)
	}
	binding := httptrust.Binding{ID: "local-ca", Name: "Owned private service", Project: "project", Origin: server.URL, File: file}
	store, err := httptrust.New([]httptrust.Binding{binding})
	if err != nil {
		t.Fatal(err)
	}
	s.HTTP = &execution.HTTP{Trust: store}
	environment, revision := currentEnvironment(t, s)
	environment.Endpoints = map[string]string{"api": server.URL}
	if err := p.SaveEnvironment(t.Context(), "admin", "project", environment, revision); err != nil {
		t.Fatal(err)
	}
	definition, revision := currentTest(t, s, "HEALTH-1")
	definition.Check.Headers = map[string]string{"X-Service-Ready": "yes"}
	definition.Check.RequiredText, definition.Check.ForbiddenText = `"status":"up"`, "unavailable"
	definition.Check.JSON = map[string]any{"status": "up", "items.0.id": 7}
	definition.Check.MinCertificateDays, definition.Check.MaxResponseMs = 1, 2000
	if err := s.Catalog.SaveTest(t.Context(), "admin", "project", "health", definition, revision); err != nil {
		t.Fatal(err)
	}
	publishAdmission(t, s, "tls-http-release")
	request := runs.Request{Suite: "health", Environment: "local"}
	manual, err := s.Create(t.Context(), "admin", "project", request, "tls-manual", "manual")
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Now().UTC()
	scheduler := &schedules.Service{Store: s.Store, Auth: s.Auth, Runs: s, Now: func() time.Time { return clock }}
	if err := scheduler.Save(t.Context(), "admin", "project", scheduleDefinition("tls"), 0); err != nil {
		t.Fatal(err)
	}
	pending, _ := readSchedule(t, scheduler, "tls")
	clock = *pending.NextAt
	tickSchedule(t, scheduler)
	fired, _ := readSchedule(t, scheduler, "tls")
	stop := start(t, s)
	for _, id := range []string{manual.ID, fired.LastRun} {
		completed := finished(t, s, id)
		if completed.State != "completed" || completed.Outcome != "passed" || completed.Counts.Total != 1 || completed.Counts.Passed != 1 || completed.EvidenceState != "complete" {
			t.Fatal("private HTTPS loses its normal run contract", completed)
		}
		result := completed.Results[0]
		if result.HTTP == nil || result.HTTP.TrustBundle == nil || result.HTTP.TrustBundle.ID != binding.ID || result.HTTP.CertificateExpires == nil || result.Timing == nil || result.Timing.Censored || result.DurationMs <= 0 {
			t.Fatal("private HTTPS loses TLS or comparable timing evidence", result)
		}
		if !strings.Contains(result.Timing.Prediction.Context.Profile, result.HTTP.TrustBundle.Digest) {
			t.Fatal("timing profile is not bound to actual trust material")
		}
	}
	stop()
	estimate, err := s.Estimate(t.Context(), "admin", "project", request)
	if err != nil || estimate.Tests[definition.ID].Samples != 2 || hits.Load() != 2 {
		t.Fatal("manual and scheduled HTTPS do not share response-time history", estimate.Tests[definition.ID].Samples, hits.Load(), err)
	}
	queued, err := s.Create(t.Context(), "admin", "project", request, "before-trust-rotation", "manual")
	if err != nil {
		t.Fatal(err)
	}
	additional := testkit.NewCertificateAuthority(t, "Owned rotation authority")
	if err := os.WriteFile(file, append(append([]byte{}, authority.PEM...), additional.PEM...), 0600); err != nil {
		t.Fatal(err)
	}
	rotated, err := httptrust.New([]httptrust.Binding{binding})
	if err != nil {
		t.Fatal(err)
	}
	s.HTTP = &execution.HTTP{Trust: rotated}
	stop = start(t, s)
	blocked := finished(t, s, queued.ID)
	stop()
	if blocked.State != "blocked" || !strings.Contains(blocked.Error, "trust changed") || hits.Load() != 2 || blocked.Timing.Tests[definition.ID].Context.Profile != queued.Timing.Tests[definition.ID].Context.Profile {
		t.Fatal("trust rotation runs queued work or rewrites frozen predictions", blocked.State, blocked.Error)
	}
	current, err := s.Estimate(t.Context(), "admin", "project", request)
	if err != nil || current.Tests[definition.ID].Samples != 0 || current.Tests[definition.ID].Context.Profile == estimate.Tests[definition.ID].Context.Profile {
		t.Fatal("changed trust reuses incompatible timing history", err)
	}
	resumed, err := s.Create(t.Context(), "admin", "project", request, "after-trust-rotation", "manual")
	if err != nil {
		t.Fatal(err)
	}
	start(t, s)
	if final := finished(t, s, resumed.ID); final.Outcome != "passed" || hits.Load() != 3 {
		t.Fatal("newly reviewed trust cannot execute a fresh run", final.Outcome)
	}
}
