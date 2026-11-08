package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/recording/workerhttp"
	"github.com/wtiger001/teststudio/internal/recordingworker"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestRecordingWorkerCommandHelper(t *testing.T) {
	if os.Getenv("TESTSTUDIO_RECORDING_COMMAND_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			break
		}
	}
	flag.CommandLine = flag.NewFlagSet("recording-worker", flag.ContinueOnError)
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestRecordingWorkerRejectsInvalidConfigurationBeforeStorage(t *testing.T) {
	for _, options := range [][]string{{"--local", "--addr=0.0.0.0:8045"}, {"--capacity=17"}, {"--idle-limit=1s"}, {"extra-argument"}} {
		originalFlags, originalArgs := flag.CommandLine, os.Args
		root := t.TempDir()
		database := filepath.Join(root, "must-not-exist.db")
		flag.CommandLine = flag.NewFlagSet("recording-worker", flag.ContinueOnError)
		flag.CommandLine.SetOutput(io.Discard)
		os.Args = append([]string{"recording-worker", "--directory=" + root, "--database=" + database}, options...)
		err := run()
		flag.CommandLine, os.Args = originalFlags, originalArgs
		if err == nil {
			t.Fatal("invalid worker settings were accepted")
		}
		if _, err := os.Stat(database); !os.IsNotExist(err) {
			t.Fatal("invalid worker settings opened storage")
		}
	}
}

func TestRecordingWorkerCommandServesPrivateTLSAndShutsDown(t *testing.T) {
	image, gateway, seccomp := os.Getenv("TESTSTUDIO_RECORDING_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_GATEWAY_IMAGE"), os.Getenv("TESTSTUDIO_RECORDING_SECCOMP")
	if image == "" || gateway == "" || seccomp == "" {
		t.Skip("supply the cached recording runtime, gateway, and pinned sandbox policy")
	}
	root := t.TempDir()
	ca := testkit.NewCertificateAuthority(t, "Owned command verification CA")
	serverID := testkit.NewTLSIdentity(t, ca, "127.0.0.1", false)
	clientID := testkit.NewTLSIdentity(t, ca, "teststudio-api.recording.internal", true)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(root, "worker.db")
	args := []string{"-test.run=^TestRecordingWorkerCommandHelper$", "--", "--local", "--addr=" + address, "--database=" + database, "--directory=" + filepath.Join(root, "transport"), "--runtime-image=" + image, "--gateway-image=" + gateway, "--seccomp=" + seccomp, "--tls-certificate=" + serverID.Certificate, "--tls-key=" + serverID.Key, "--client-ca=" + serverID.Authority}
	command := exec.Command(executable, args...)
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "TMPDIR=" + os.Getenv("TMPDIR"), "TESTSTUDIO_RECORDING_COMMAND_HELPER=1"}
	log, err := os.OpenFile(filepath.Join(root, "worker.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	command.Stdout, command.Stderr = log, log
	if err := command.Start(); err != nil {
		log.Close()
		t.Fatal(err)
	}
	log.Close()
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = command.Process.Signal(syscall.SIGTERM)
		select {
		case err := <-done:
			if err != nil {
				t.Error("owned recording worker did not shut down cleanly", err)
			}
		case <-time.After(15 * time.Second):
			_ = command.Process.Kill()
			<-done
			t.Error("owned recording worker needed forced shutdown")
		}
	}
	t.Cleanup(stop)
	client, err := workerhttp.New(workerhttp.Settings{Origin: "https://" + address, ServerCA: serverID.Authority, Certificate: clientID.Certificate, Key: clientID.Key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	ready := false
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		err := client.CheckReady(ctx)
		cancel()
		if err == nil {
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		data, _ := os.ReadFile(filepath.Join(root, "worker.log"))
		t.Fatalf("private worker did not become ready: %s", strings.TrimSpace(string(data)))
	}
	stop()
	db, err := storage.Open(t.Context(), database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.List(t.Context(), recordingworker.LeaseBucket, "", "", 10)
	if err != nil || len(rows.Items) != 0 {
		t.Fatal("idle worker launched a browser", err)
	}
	row, err := db.Get(t.Context(), recordingworker.PoolBucket, "recording")
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := storage.Decode[recordingworker.Pool](row)
	if time.Now().Before(pool.Until) {
		t.Fatal("shutdown retained live worker authority")
	}
}
