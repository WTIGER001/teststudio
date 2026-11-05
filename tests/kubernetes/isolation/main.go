package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

var forbidden = []string{"teststudio-api.teststudio-local.svc.cluster.local:80", "teststudio-artifacts.teststudio-local.svc.cluster.local:80", "postgres.teststudio-local.svc.cluster.local:5432", "teststudio-fixture.teststudio-local.svc.cluster.local:443"}

func baseline() (map[string]any, error) {
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, "10.43.0.10:53")
	}}
	for _, address := range forbidden {
		connection, err := net.DialTimeout("tcp", address, 3*time.Second)
		if err != nil {
			return nil, fmt.Errorf("the target is not reachable from the owned node: %s: %w", address, err)
		}
		connection.Close()
	}
	return map[string]any{"passed": true, "reachableLocalServices": forbidden}, nil
}

func verify() (map[string]any, error) {
	version, err := os.ReadFile("/proc/version")
	if err != nil || !strings.Contains(string(version), "gvisor") || os.Getuid() != 1000 {
		return nil, errors.New("the diagnostic requires actual non-root gVisor execution")
	}
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("a Kubernetes API credential is present")
	}
	ca, err := os.ReadFile("/ca/ca.crt")
	if err != nil {
		return nil, errors.New("the local exchange authority is unavailable")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return nil, errors.New("the local exchange authority is invalid")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	retries := 0
	for _, path := range []string{"/internal/worker/v1/executions/unowned/input", "/api/v1/projects"} {
		response, err := client.Get("https://worker.teststudio.test" + path)
		readyUntil := time.Now().Add(15 * time.Second)
		for err != nil && time.Now().Before(readyUntil) {
			time.Sleep(250 * time.Millisecond)
			retries++
			response, err = client.Get("https://worker.teststudio.test" + path)
		}
		if err != nil {
			return nil, fmt.Errorf("the permitted private TLS endpoint is unreachable: %w", err)
		}
		response.Body.Close()
		if response.StatusCode != 404 || strings.HasPrefix(path, "/internal/") && !strings.Contains(response.Header.Get("Content-Type"), "application/json") {
			return nil, errors.New("the private endpoint did not enforce its exchange-only route and authentication")
		}
	}
	denied := []string{}
	for _, address := range forbidden {
		connection, err := net.DialTimeout("tcp", address, 1500*time.Millisecond)
		if err == nil {
			connection.Close()
			return nil, fmt.Errorf("the worker reaches a forbidden local service: %s", address)
		}
		var networkError net.Error
		if !errors.As(err, &networkError) || !networkError.Timeout() && !errors.Is(err, syscall.EHOSTUNREACH) && !errors.Is(err, syscall.ENETUNREACH) && !errors.Is(err, syscall.EACCES) && !errors.Is(err, syscall.ECONNREFUSED) {
			return nil, fmt.Errorf("the connection failed without evidence of network denial: %s: %w", address, err)
		}
		denied = append(denied, address)
	}
	return map[string]any{"passed": true, "uid": os.Getuid(), "kernel": strings.TrimSpace(string(version)), "privateTLSExchangeReachable": true, "unownedExchangeDenied": true, "privateAPIRouteDenied": true, "apiTokenAbsent": true, "deniedLocalServices": denied, "privateReadinessRetries": retries}, nil
}

func main() {
	var summary map[string]any
	var err error
	if len(os.Args) == 2 && os.Args[1] == "--baseline" {
		summary, err = baseline()
	} else {
		summary, err = verify()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(summary)
}
