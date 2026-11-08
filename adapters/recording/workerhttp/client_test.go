package workerhttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/recordingworker"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type transportRuntime struct {
	starts, inputs int
	sequence       int
}

func (r *transportRuntime) Plan(b recordingworker.Binding, target browserpolicy.Target) (recordingworker.Reference, error) {
	return recordingworker.Reference{Fingerprint: b.Fingerprint()}, nil
}
func (r *transportRuntime) Start(context.Context, recordingworker.Binding, recordingworker.Reference, browserpolicy.Target) (recording.InteractiveDriver, error) {
	r.starts++
	return &transportDriver{r}, nil
}
func (r *transportRuntime) Cleanup(context.Context, recordingworker.Binding, recordingworker.Reference) error {
	return nil
}

type transportDriver struct{ *transportRuntime }

func (r *transportRuntime) Read(context.Context, string) (recording.Snapshot, error) {
	return recording.Snapshot{State: "recording", Code: "captured private source"}, nil
}
func (r *transportRuntime) Stop(context.Context, string) (recording.Snapshot, error) {
	return recording.Snapshot{State: "ready", Code: "captured private source"}, nil
}
func (r *transportRuntime) View(context.Context, string) (recording.View, error) {
	return recording.View{Sequence: r.sequence, Image: "private-frame", Steps: []recording.Step{}}, nil
}
func (r *transportRuntime) Interact(ctx context.Context, id string, action recording.Action) (recording.View, error) {
	r.inputs++
	r.sequence = action.Sequence
	return r.View(ctx, id)
}
func (d *transportDriver) Start(context.Context, string, string) error { return nil }

func transportFixture(t *testing.T) (*recordingworker.Worker, *transportRuntime, recording.Session) {
	t.Helper()
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "transport.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	directory := &identity.Directory{Store: db}
	if err := directory.SaveUser(t.Context(), identity.User{ID: "admin", Name: "Administrator"}, 0); err != nil {
		t.Fatal(err)
	}
	auth, err := authorization.New(t.Context(), testkit.PermissionStore(t, db), directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Engine.AssignRole(t.Context(), permissions.PrincipalRef{Kind: permissions.PrincipalUser, ID: "admin"}, "builtin.teststudio.installation-admin", nil); err != nil {
		t.Fatal(err)
	}
	p := &projects.Service{Store: db, Auth: auth}
	if err := p.SaveProject(t.Context(), "admin", projects.Project{ID: "recording", Name: "Recording"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.SaveSuite(t.Context(), "admin", "recording", projects.Suite{ID: "browser", Name: "Browser"}, 0); err != nil {
		t.Fatal(err)
	}
	runtime := &transportRuntime{}
	w := &recordingworker.Worker{Store: db, Auth: auth, Runtime: runtime, ID: "worker", Instance: "one", Capacity: 2, IdleLimit: time.Minute}
	if err := w.Open(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	session := recording.Session{Input: recording.Input{ID: "session", URL: "https://example.test/", TestID: "NEWS-001", Title: "News stories"}, Mode: "embedded", Project: "recording", Suite: "browser", Actor: "admin", State: "starting", CreatedAt: time.Now().UTC(), DeadlineAt: time.Now().Add(recording.Lifetime).UTC()}
	change, _ := storage.Put(recording.Bucket(), session.ID, session.Project, 0, session)
	if err := db.Commit(t.Context(), change); err != nil {
		t.Fatal(err)
	}
	return w, runtime, session
}

func TestPrivateRecordingTransportRequiresMutualTLSAndRecoversAPIClient(t *testing.T) {
	w, runtime, session := transportFixture(t)
	ca := testkit.NewCertificateAuthority(t, "Owned recording transport CA")
	serverID := testkit.NewTLSIdentity(t, ca, "127.0.0.1", false)
	clientID := testkit.NewTLSIdentity(t, ca, "api.recording.test", true)
	config, err := ServerTLS(serverID.Certificate, serverID.Key, serverID.Authority, "api.recording.test")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(Handler(w))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = config
	server.StartTLS()
	t.Cleanup(server.Close)
	settings := Settings{Origin: server.URL, ServerCA: serverID.Authority, Certificate: clientID.Certificate, Key: clientID.Key}
	client, err := New(settings, w.Store)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	if err := client.Start(t.Context(), session.ID, session.URL); err != nil {
		t.Fatal(err)
	}
	cloned, err := New(settings, w.Store)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cloned.Close)
	if view, err := cloned.View(t.Context(), session.ID); err != nil || view.Image != "private-frame" || runtime.starts != 1 {
		t.Fatal("API client recovery opened another browser", err)
	}
	for name, filename := range map[string]string{"TESTSTUDIO_TEST_RECORDING_CA": serverID.Authority, "TESTSTUDIO_TEST_RECORDING_CERT": clientID.Certificate, "TESTSTUDIO_TEST_RECORDING_KEY": clientID.Key} {
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv(name, string(data))
	}
	environmentSettings := Settings{Origin: server.URL, ServerCAEnvironment: "TESTSTUDIO_TEST_RECORDING_CA", CertificateEnvironment: "TESTSTUDIO_TEST_RECORDING_CERT", KeyEnvironment: "TESTSTUDIO_TEST_RECORDING_KEY"}
	dataSettings, _ := json.Marshal(environmentSettings)
	t.Setenv("TESTSTUDIO_TEST_RECORDING_CONFIG", string(dataSettings))
	loaded, err := LoadEnvironment("TESTSTUDIO_TEST_RECORDING_CONFIG")
	if err != nil {
		t.Fatal(err)
	}
	environmentClient, err := New(loaded, w.Store)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(environmentClient.Close)
	if _, err := environmentClient.View(t.Context(), session.ID); err != nil {
		t.Fatal("mounted secret environment could not connect", err)
	}
	invalid := loaded
	invalid.Key = clientID.Key
	if _, err := New(invalid, w.Store); err == nil {
		t.Fatal("ambiguous credential sources were accepted")
	}
	input := recording.Action{Action: "click", Sequence: 1, X: 20, Y: 20}
	for i := 0; i < 2; i++ {
		if _, err := cloned.Interact(t.Context(), session.ID, input); err != nil {
			t.Fatal(err)
		}
	}
	if runtime.inputs != 1 {
		t.Fatal("transport retry repeated browser input")
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	withoutCertificate := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13}}}
	t.Cleanup(withoutCertificate.CloseIdleConnections)
	if response, err := withoutCertificate.Get(server.URL + "/healthz"); err == nil {
		response.Body.Close()
		t.Fatal("worker accepted an anonymous TLS client")
	}
	wrongID := testkit.NewTLSIdentity(t, ca, "another-api.recording.test", true)
	wrongSettings := settings
	wrongSettings.Certificate, wrongSettings.Key = wrongID.Certificate, wrongID.Key
	wrong, err := New(wrongSettings, w.Store)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(wrong.Close)
	if _, err := wrong.View(t.Context(), session.ID); err == nil {
		t.Fatal("worker accepted a different client identity")
	}
	request := recordingworker.Request{Binding: recordingworker.Bind(session)}
	for _, header := range []string{"Origin", "Cookie", "Authorization", "Sec-Fetch-Site"} {
		data, _ := json.Marshal(request)
		r, _ := http.NewRequest("POST", server.URL+"/internal/recording/session/view", bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set(header, "private-canary")
		response, err := client.http.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 403 {
			t.Fatal("browser credentials entered private transport", header)
		}
	}
	request.Binding.Actor = "another-user"
	data, _ := json.Marshal(request)
	r, _ := http.NewRequest("POST", server.URL+"/internal/recording/session/view", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("transport accepted another actor's binding")
	}
	if snapshot, err := client.Stop(t.Context(), session.ID); err != nil || snapshot.State != "ready" {
		t.Fatal(err)
	}
}
