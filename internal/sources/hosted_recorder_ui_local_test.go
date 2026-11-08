package sources_test

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/recording/isolatedcontainer"
	"github.com/wtiger001/teststudio/adapters/recording/workerhttp"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/recordingworker"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

type observedHostedClient struct {
	*workerhttp.Client
	observer *observedRecorder
	close    func() error
}

func (c *observedHostedClient) Start(ctx context.Context, id, address string) error {
	err := c.Client.Start(ctx, id, address)
	c.observer.observe()
	return err
}
func (c *observedHostedClient) Read(ctx context.Context, id string) (recording.Snapshot, error) {
	c.observer.observe()
	return c.Client.Read(ctx, id)
}
func (c *observedHostedClient) Stop(ctx context.Context, id string) (recording.Snapshot, error) {
	c.observer.observe()
	return c.Client.Stop(ctx, id)
}
func (c *observedHostedClient) Close() error { return c.close() }

func TestPrivateRecordingWorkerCompletesNewsRecordingThroughStudioExecution(t *testing.T) {
	if os.Getenv("TESTSTUDIO_HOSTED_RECORDING_UI") != "1" {
		t.Skip("enable the focused private worker recording UI check")
	}
	runRecordingAuthoringUI(t, true)
}

func newHostedRecordingClient(t *testing.T, source *sources.Service, private, output string, observer *observedRecorder) *observedHostedClient {
	t.Helper()
	image, gateway, seccomp := os.Getenv("TESTSTUDIO_RECORDING_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_GATEWAY_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_SECCOMP")
	if image == "" || gateway == "" || seccomp == "" {
		t.Fatal("the private worker check requires current immutable recording and gateway images and the pinned sandbox policy")
	}
	runtime := &isolatedcontainer.Runtime{Image: image, GatewayImage: gateway, Seccomp: seccomp, Directory: filepath.Join(private, "isolated")}
	worker := &recordingworker.Worker{Store: source.Store, Auth: source.Auth, Runtime: runtime, ID: "authoring-recorder", Instance: domain.NewID(), Capacity: 2, IdleLimit: time.Minute, AllowPrivate: true}
	if err := worker.Open(t.Context()); err != nil {
		t.Fatal(err)
	}
	ca := testkit.NewCertificateAuthority(t, "Independent private recorder authority")
	serverID := testkit.NewTLSIdentity(t, ca, "127.0.0.1", false)
	clientID := testkit.NewTLSIdentity(t, ca, "api.recording.test", true)
	tlsConfig, err := workerhttp.ServerTLS(serverID.Certificate, serverID.Key, serverID.Authority, "api.recording.test")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(workerhttp.Handler(worker))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = tlsConfig
	server.StartTLS()
	t.Cleanup(server.Close)
	client, err := workerhttp.New(workerhttp.Settings{Origin: server.URL, ServerCA: serverID.Authority, Certificate: clientID.Certificate, Key: clientID.Key}, source.Store)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := worker.StartMaintenance(ctx)
	closed := false
	return &observedHostedClient{Client: client, observer: observer, close: func() error {
		if closed {
			return nil
		}
		closed = true
		cancel()
		<-done
		client.Close()
		cleanup, stop := context.WithTimeout(context.Background(), 45*time.Second)
		defer stop()
		if err := worker.Close(cleanup); err != nil {
			return err
		}
		page, err := source.Store.List(cleanup, recordingworker.LeaseBucket, "beta", "", 100)
		if err != nil {
			return err
		}
		refs := []recordingworker.Reference{}
		for _, row := range page.Items {
			lease, err := storage.Decode[recordingworker.Lease](row)
			if err != nil {
				return err
			}
			if lease.CleanupPending {
				t.Error("recording worker cleanup remains pending")
			}
			if err := runtime.Cleanup(cleanup, lease.Binding, lease.Reference); err != nil {
				return err
			}
			refs = append(refs, lease.Reference)
		}
		data, _ := json.MarshalIndent(map[string]any{"resources": refs, "remaining": 0}, "", "  ")
		return os.WriteFile(filepath.Join(output, "recording-resources.json"), data, 0600)
	}}
}
