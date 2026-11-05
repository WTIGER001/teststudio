package jobsupervisor

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/testkit"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

type gatewayTestTransport struct {
	packet     workerprotocol.Packet
	connection browsergateway.RemoteConnection
	submitted  bool
}

func (c *gatewayTestTransport) Input(context.Context) (workerprotocol.Packet, error) {
	return c.packet, nil
}
func (c *gatewayTestTransport) GatewayConnection(context.Context) (browsergateway.RemoteConnection, error) {
	return c.connection, nil
}
func (c *gatewayTestTransport) Started(context.Context, time.Time) error { return nil }
func (c *gatewayTestTransport) Submit(_ context.Context, report workerprotocol.Submission) error {
	if report.Report.Result.State != "passed" {
		return workerprotocol.ErrInvalid
	}
	c.submitted = true
	return nil
}
func (c *gatewayTestTransport) Upload(context.Context, string, io.Reader) error { return nil }

func TestSupervisorStartsPrivateBridgeBeforeChildAndRemovesItAfterDelivery(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(map[bool]string{false: "matching execution", true: "foreign execution"}[mismatch], func(t *testing.T) {
			root, err := os.MkdirTemp("/tmp", "studio-supervisor-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(root)
			path := filepath.Join(root, "gateway.sock")
			token := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("g", 32)))
			t.Setenv("TESTSTUDIO_GATEWAY_TOKEN", token)
			f := newJob(t, "playwright", "gateway:"+path)
			policy := &apipolicy.Spec{Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/approved", ReadOnly: true}}, MaxRequests: 2, MaxResponseBytes: 4096}
			f.input.Browser = &browserpolicy.Target{Origin: "https://application.test", Requests: policy}
			f.input.Artifact.Browser = &browserpolicy.Spec{Endpoint: "api", Requests: policy}
			wire, err := execution.MarshalWorkItem(f.input)
			if err != nil {
				t.Fatal(err)
			}
			ca := testkit.NewCertificateAuthority(t, "supervisor gateway")
			server := testkit.NewTLSServer(t, ca, "127.0.0.1", time.Hour, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Proxy-Authorization") != "Bearer "+token || r.Header.Get("X-TestStudio-Gateway-Execution") != f.ref.ID || r.URL.String() != "https://application.test/approved" {
					t.Error("supervisor gateway transport lost execution or target scope")
					w.WriteHeader(403)
					return
				}
				io.WriteString(w, "reviewed gateway response")
			}))
			deadline := time.Now().Add(time.Minute)
			transport := &gatewayTestTransport{packet: workerprotocol.Packet{Execution: f.ref.ID, Deadline: deadline, Input: wire}, connection: browsergateway.RemoteConnection{Origin: server.URL, CertificateAuthority: ca.PEM, Session: browsergateway.RemoteSession{Execution: f.ref.ID, Token: token, Deadline: deadline}}}
			if mismatch {
				transport.connection.Session.Execution = "other-execution"
			}
			supervisor := &Supervisor{Transport: transport, Runner: f.process, GatewaySocket: path}
			outcome, err := supervisor.Run(t.Context())
			if mismatch {
				if err != ErrWork || transport.submitted {
					t.Fatal("foreign gateway authority released code", err)
				}
				if _, err := os.Stat(filepath.Join(f.root, "executed")); !os.IsNotExist(err) {
					t.Fatal("runner started before gateway scope validation")
				}
			} else if err != nil || !outcome.ReportAccepted || !transport.submitted {
				t.Fatal("supervised gateway delivery failed", err)
			}
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatal("supervisor retained its gateway socket")
			}
		})
	}
}
