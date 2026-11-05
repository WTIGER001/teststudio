package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestOwnedServiceCommand(t *testing.T) {
	if os.Getenv("TESTSTUDIO_SERVICE_COMMAND_CHECK") != "1" {
		t.Skip("owned service subprocess helper")
	}
	for index, value := range os.Args {
		if value == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[index+1:]...)
			break
		}
	}
	flag.CommandLine = flag.NewFlagSet("owned-service-command", flag.ContinueOnError)
	if err := run(); err != nil {
		t.Fatal(err)
	}
}

func startOwnedServiceCommand(t *testing.T, address string, options ...string) func() {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	arguments := append([]string{"-test.run=^TestOwnedServiceCommand$", "--"}, options...)
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Env = append(os.Environ(), "TESTSTUDIO_SERVICE_COMMAND_CHECK=1")
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			defer cancel()
			command.Process.Signal(syscall.SIGTERM)
			select {
			case err := <-done:
				if err != nil {
					t.Error(err)
				}
			case <-time.After(3 * time.Second):
				cancel()
				<-done
				t.Error("service command did not stop")
			}
			connection, err := net.DialTimeout("tcp", address, time.Second)
			if err == nil {
				connection.Close()
				t.Error("service command listener remained open")
			}
		})
	}
	t.Cleanup(stop)
	return stop
}
