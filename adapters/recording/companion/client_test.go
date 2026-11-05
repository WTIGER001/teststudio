package companion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/recording"
)

func TestDesktopConnectionUsesHTTPSOrLoopbackAndDoesNotFollowRedirects(t *testing.T) {
	for _, address := range []string{"http://studio.example.test", "http://localhost:8040", "https://user:private@studio.example.test", "https://studio.example.test/path", "https://studio.example.test?token=private", "https://studio.example.test/#secret"} {
		if _, err := NewClient(address); err == nil {
			t.Fatal("unsafe server origin accepted")
		}
	}
	received := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { received++ }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	c, err := NewClient(redirect.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Control(t.Context(), "session", "desktop", "private-device"); !errors.Is(err, ErrTransport) || received != 0 {
		t.Fatal("redirect forwarded a desktop credential", err)
	}
}

func TestDesktopTransportOmitsBrowserHeadersAndRedactsRemoteErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Origin") != "" || r.Header.Get("Sec-Fetch-Site") != "" {
			t.Error("desktop sent browser authority")
		}
		if strings.HasSuffix(r.URL.Path, "/pair") {
			if r.Header.Get("Authorization") != "" {
				t.Error("pairing reused a device credential")
			}
			json.NewEncoder(w).Encode(recording.CompanionPacket{})
			return
		}
		if r.Header.Get("Authorization") != "Bearer private-device" {
			t.Error("device capability was not bound to control")
		}
		w.WriteHeader(403)
		fmt.Fprintln(w, `{"detail":"private-server-token"}`)
	}))
	defer server.Close()
	c, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Pair(t.Context(), "session", recording.PairRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Control(t.Context(), "session", "desktop", "private-device"); !errors.Is(err, ErrAuthority) || strings.Contains(err.Error(), "private") {
		t.Fatal("remote error leaked private response", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Control(cancelled, "session", "desktop", "private-device"); !errors.Is(err, ErrTransport) {
		t.Fatal(err)
	}
}

func TestDesktopJournalKeepsCredentialsPrivateAndExcludesConcurrentCompanions(t *testing.T) {
	j, err := NewJournal(filepath.Join(t.TempDir(), "connections"))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := j.Lock()
	if err != nil {
		t.Fatal(err)
	}
	if second, err := j.Lock(); err == nil {
		second.Close()
		t.Fatal("concurrent companion acquired private recorder")
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	lock, err = j.Lock()
	if err != nil {
		t.Fatal("terminated owner retained recorder lock", err)
	}
	defer lock.Close()
	token, _ := randomToken()
	state, err := NewState("https://studio.example.test/", "session:"+token)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%+v %#v", state, state) != "desktop recording: session desktop recording: session" {
		t.Fatal("desktop state formatting disclosed credentials")
	}
	if err := j.Create(state); err != nil {
		t.Fatal(err)
	}
	if err := j.Create(state); err == nil {
		t.Fatal("existing connection overwritten")
	}
	path := filepath.Join(j.directory, "session.json")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("connection credentials are not private", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Read("session"); err == nil {
		t.Fatal("public connection file accepted")
	}
	if err := j.Save(state); err == nil {
		t.Fatal("public connection file replaced")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "other"), path); err != nil {
		t.Fatal(err)
	}
	if err := j.Save(state); err == nil {
		t.Fatal("symbolic link accepted as durable connection")
	}
}

func TestDesktopJournalRecoversBoundedSourceWithJSONEscapes(t *testing.T) {
	r, state, _, _ := newRunnerFixture(t)
	value := recording.CompanionSnapshot{Sequence: 1, State: "recording", Code: strings.Repeat("\x00", recording.ContentLimit)}
	state.Last, state.Pending, state.Final = value, &value, &value
	if err := r.Journal.Save(*state); err != nil {
		t.Fatal(err)
	}
	saved, err := r.Journal.Read(state.ID)
	if err != nil || saved.Last.Code != value.Code || saved.Pending.Code != value.Code || saved.Final.Code != value.Code {
		t.Fatal("valid bounded source became unreadable after escaping", err)
	}
}
