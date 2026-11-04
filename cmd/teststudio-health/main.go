package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("teststudio-health", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	address := flags.String("url", "http://127.0.0.1:8040/healthz", "API health endpoint")
	timeout := flags.Duration("timeout", 2*time.Second, "Probe deadline")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *timeout <= 0 || *timeout > time.Minute {
		return errors.New("invalid health probe options")
	}
	parsed, err := url.Parse(*address)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("invalid health probe endpoint")
	}
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: *timeout, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(*address)
	if err != nil {
		return errors.New("TestStudio health probe could not reach the API")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("TestStudio API is not ready")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil || len(data) > 4096 {
		return errors.New("TestStudio health response is unavailable")
	}
	var state struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	if json.Unmarshal(data, &state) != nil || state.Status != "up" || state.Service != "teststudio" {
		return errors.New("TestStudio health response is invalid")
	}
	return nil
}
