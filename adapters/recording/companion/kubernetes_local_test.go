package companion

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/recording/localplaywright"
)

func TestDesktopCompanionAgainstOwnedKubernetes(t *testing.T) {
	filename := os.Getenv("TESTSTUDIO_KUBERNETES_RECORDING_INPUT")
	if filename == "" {
		t.Skip("supply the owned local Kubernetes recording connection")
	}
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 16384 {
		t.Fatal("private recording input is missing or unsafe")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal("private recording input cannot be read")
	}
	var input struct {
		Connection  string `json:"connection"`
		Certificate string `json:"certificate"`
		Directory   string `json:"directory"`
	}
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal("private recording input is invalid")
	}
	state, err := NewState("https://studio.teststudio.test", input.Connection)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := NewJournal(input.Directory)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := journal.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := journal.Create(state); err != nil {
		t.Fatal(err)
	}
	remote, err := NewClient(state.Server)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := os.ReadFile(input.Certificate)
	if err != nil {
		t.Fatal("owned installation certificate is unavailable")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certificate) {
		t.Fatal("owned installation certificate is invalid")
	}
	remote.http.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" || address != "studio.teststudio.test:443" {
			return nil, errors.New("native companion attempted another origin")
		}
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", "127.0.0.1:18443")
	}}
	defer remote.http.CloseIdleConnections()
	workspace, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	driver, err := localplaywright.New(t.Context(), node, filepath.Join(workspace, "runners/playwright/node_modules/playwright/cli.js"), filepath.Join(workspace, "scripts/record-browser.mjs"), filepath.Join(input.Directory, "captures"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := driver.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	if err := (&Runner{Remote: remote, Driver: driver, Journal: journal}).Run(ctx, &state); err != nil {
		t.Fatal(err)
	}
	if state.Final == nil || state.Last.State != "ready" || state.Pending != nil || state.Last.Code == "" {
		t.Fatal("native companion did not acknowledge final captured source")
	}
	t.Logf("Owned native recorder closed and acknowledged session %s", state.ID)
}
