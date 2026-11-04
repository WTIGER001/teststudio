package kubernetes

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
	"github.com/wtiger001/teststudio/internal/storage"
)

type RegistrySettings struct {
	Origin              string   `json:"origin"`
	RepositoryPrefix    string   `json:"repositoryPrefix"`
	TokenURL            string   `json:"tokenURL,omitempty"`
	TokenService        string   `json:"tokenService,omitempty"`
	BlobRedirectOrigins []string `json:"blobRedirectOrigins,omitempty"`
	CAFile              string   `json:"caFile,omitempty"`
	UsernameEnvironment string   `json:"usernameEnvironment,omitempty"`
	PasswordEnvironment string   `json:"passwordEnvironment,omitempty"`
	BearerEnvironment   string   `json:"bearerEnvironment,omitempty"`
}

type Settings struct {
	Worker   Config           `json:"worker"`
	Registry RegistrySettings `json:"registry"`
}

var registryEnvironment = regexp.MustCompile(`^TESTSTUDIO_REGISTRY_[A-Z0-9_]{1,96}$`)

func (s Settings) validate() error {
	r := s.Registry
	if Validate(s.Worker) != nil || !filepath.IsAbs(s.Worker.CAFile) || !filepath.IsAbs(s.Worker.TokenFile) || r.CAFile != "" && !filepath.IsAbs(r.CAFile) {
		return hostedexecution.ErrConfiguration
	}
	if r.BearerEnvironment != "" && (r.UsernameEnvironment != "" || r.PasswordEnvironment != "") || (r.UsernameEnvironment == "") != (r.PasswordEnvironment == "") {
		return hostedexecution.ErrConfiguration
	}
	for _, name := range []string{r.UsernameEnvironment, r.PasswordEnvironment, r.BearerEnvironment} {
		if name != "" && !registryEnvironment.MatchString(name) {
			return hostedexecution.ErrConfiguration
		}
	}
	return nil
}

func LoadSettings(file, variable string, lookup func(string) (string, bool)) (*Settings, error) {
	var settings Settings
	found, err := runtimeconfig.LoadJSON(file, variable, "Kubernetes worker", lookup, &settings)
	if err != nil || !found {
		return nil, err
	}
	r := settings.Registry
	if err := settings.validate(); err != nil {
		return nil, err
	}
	client, err := oci.New(oci.Options{Origin: r.Origin, RepositoryPrefix: r.RepositoryPrefix, Architecture: settings.Worker.Architecture, TokenURL: r.TokenURL, TokenService: r.TokenService, BlobRedirectOrigins: r.BlobRedirectOrigins})
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	client.Close()
	return &settings, nil
}

func (s Settings) Open(store *storage.Store, lookup func(string) (string, bool), discoveryCleanup func(context.Context, execution.Reference) error) (*Provider, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if lookup == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	data, err := readPrivate(s.Worker.TokenFile, 64<<10)
	if err != nil || strings.TrimSpace(string(data)) == "" {
		return nil, hostedexecution.ErrConfiguration
	}
	registry, err := s.OpenRegistry(lookup)
	if err != nil {
		return nil, err
	}
	provider, err := New(Options{Config: s.Worker, Store: store, Inspector: registry, DiscoveryCleanup: discoveryCleanup})
	if err != nil {
		registry.Close()
		return nil, err
	}
	provider.closeRegistry = registry.Close
	return provider, nil
}

func (s Settings) OpenRegistry(lookup func(string) (string, bool)) (*oci.Client, error) {
	if s.validate() != nil || lookup == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	r := s.Registry
	options := oci.Options{Origin: r.Origin, RepositoryPrefix: r.RepositoryPrefix, Architecture: s.Worker.Architecture, TokenURL: r.TokenURL, TokenService: r.TokenService, BlobRedirectOrigins: r.BlobRedirectOrigins}
	if r.CAFile != "" {
		data, err := readPrivate(r.CAFile, 1<<20)
		if err != nil {
			return nil, err
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(data) {
			return nil, hostedexecution.ErrConfiguration
		}
		options.Transport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second}
	}
	options.Credentials = func(context.Context) (oci.Credential, error) {
		read := func(name string) (string, error) {
			if name == "" {
				return "", nil
			}
			value, ok := lookup(name)
			if !ok || value == "" {
				return "", errors.New("registry credential is unavailable")
			}
			return value, nil
		}
		username, err := read(r.UsernameEnvironment)
		if err != nil {
			return oci.Credential{}, err
		}
		password, err := read(r.PasswordEnvironment)
		if err != nil {
			return oci.Credential{}, err
		}
		token, err := read(r.BearerEnvironment)
		if err != nil {
			return oci.Credential{}, err
		}
		return oci.Credential{Username: username, Password: password, Token: token}, nil
	}
	registry, err := oci.New(options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return registry, nil
}
