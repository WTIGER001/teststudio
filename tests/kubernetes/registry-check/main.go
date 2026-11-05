package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	credentials := flag.String("credentials", "", "Private local registry credential file")
	caFile := flag.String("ca", "", "Local registry authority")
	flag.Parse()
	var users map[string]struct{ Username, Password string }
	data, err := os.ReadFile(*credentials)
	if err != nil || json.Unmarshal(data, &users) != nil {
		panic("local registry credentials are unavailable")
	}
	data, err = os.ReadFile(*caFile)
	roots := x509.NewCertPool()
	if err != nil || !roots.AppendCertsFromPEM(data) {
		panic("local registry authority is unavailable")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, scenario := range []struct {
		Role, Method, Path string
		Status             int
	}{{"", "GET", "/v2/", 401}, {"reader", "GET", "/v2/", 200}, {"writer", "GET", "/v2/", 200}, {"reader", "POST", "/v2/teststudio/source/blobs/uploads/", 403}, {"reader", "GET", "/v2/another-project/source/tags/list", 404}} {
		request, _ := http.NewRequest(scenario.Method, "https://registry.teststudio.test"+scenario.Path, nil)
		if scenario.Role != "" {
			user := users[scenario.Role]
			request.SetBasicAuth(user.Username, user.Password)
		}
		response, err := client.Do(request)
		if err != nil {
			panic("local registry request failed: " + err.Error())
		}
		response.Body.Close()
		if response.StatusCode != scenario.Status {
			panic(fmt.Sprintf("local registry %s %s: got %d, want %d", scenario.Role, scenario.Method, response.StatusCode, scenario.Status))
		}
	}
	request, _ := http.NewRequest(http.MethodPost, "https://registry.teststudio.test/v2/teststudio/registry-check/blobs/uploads/", nil)
	writer := users["writer"]
	request.SetBasicAuth(writer.Username, writer.Password)
	response, err := client.Do(request)
	if err != nil {
		panic("local registry upload start failed")
	}
	response.Body.Close()
	location := response.Header.Get("Location")
	uploadURL, err := url.Parse(location)
	if response.StatusCode != 202 || err != nil {
		panic("local registry did not return an upload session")
	}
	body := make([]byte, 32)
	rand.Read(body)
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(body))
	query := uploadURL.Query()
	query.Set("digest", digest)
	uploadURL.RawQuery = query.Encode()
	request, _ = http.NewRequest(http.MethodPut, uploadURL.String(), bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/octet-stream")
	request.SetBasicAuth(writer.Username, writer.Password)
	response, err = client.Do(request)
	if err != nil {
		panic("local registry blob write failed")
	}
	response.Body.Close()
	if response.StatusCode != 201 {
		panic(fmt.Sprintf("local registry blob write returned %d", response.StatusCode))
	}
	request, _ = http.NewRequest(http.MethodDelete, "https://registry.teststudio.test/v2/teststudio/registry-check/blobs/"+digest, nil)
	request.SetBasicAuth(writer.Username, writer.Password)
	response, err = client.Do(request)
	if err != nil {
		panic("local registry blob cleanup failed")
	}
	response.Body.Close()
	if response.StatusCode != 202 || uploadURL.Scheme != "https" || uploadURL.Host != "registry.teststudio.test" {
		panic(fmt.Sprintf("unexpected upload location or cancellation: %s://%s, status %d", uploadURL.Scheme, uploadURL.Host, response.StatusCode))
	}
	fmt.Println(`{"passed":true,"verifiedTLS":true,"anonymousDenied":true,"readerWriteDenied":true,"readerAndWriterAuthenticated":true,"foreignRepositoryDenied":true,"writerBlobVerifiedAndRemoved":true}`)
}
