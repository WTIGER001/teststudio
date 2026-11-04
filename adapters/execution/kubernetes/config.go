package kubernetes

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

var dnsName = regexp.MustCompile(`^[a-z0-9](?:[-a-z0-9]{0,61}[a-z0-9])?$`)
var immutable = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64}$`)

func origin(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.Path == "" && parsed.RawQuery == "" && !parsed.ForceQuery && parsed.Fragment == "" && parsed.Opaque == ""
}

func validPeer(peer Peer) bool {
	if !dnsName.MatchString(peer.Namespace) || len(peer.Labels) < 1 || len(peer.Labels) > 16 {
		return false
	}
	for key, value := range peer.Labels {
		if len(key) < 1 || len(key) > 253 || len(value) > 63 || strings.ContainsAny(key+value, "\x00\r\n,= ") {
			return false
		}
	}
	return true
}

func Validate(config Config) error {
	if config.ExchangePort < 0 || config.ExchangePort > 65535 {
		return hostedexecution.ErrConfiguration
	}
	if !origin(config.APIOrigin) || !origin(config.ExchangeOrigin) || !dnsName.MatchString(config.Namespace) || !dnsName.MatchString(config.ServiceAccount) || !dnsName.MatchString(config.PolicyName) || !dnsName.MatchString(config.RuntimeClass) || (config.Architecture != "amd64" && config.Architecture != "arm64") || !validPeer(config.ExchangePeer) || !validPeer(config.DNSPeer) || len(config.ImagePullSecrets) > 16 || len(config.NodeSelector) > 16 {
		return hostedexecution.ErrConfiguration
	}
	for _, value := range config.ImagePullSecrets {
		if !dnsName.MatchString(value) {
			return hostedexecution.ErrConfiguration
		}
	}
	for key, value := range config.NodeSelector {
		if key == "kubernetes.io/os" || key == "kubernetes.io/arch" || len(key) < 1 || len(key) > 253 || len(value) > 63 || strings.ContainsAny(key+value, "\x00\r\n,= ") {
			return hostedexecution.ErrConfiguration
		}
	}
	return nil
}

func readPrivate(name string, limit int64) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, hostedexecution.ErrConfiguration
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, hostedexecution.ErrConfiguration
	}
	return data, nil
}

func New(options Options) (*Provider, error) {
	if Validate(options.Config) != nil || options.Store == nil || options.Inspector == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	transport := options.Transport
	if transport == nil {
		data, err := readPrivate(options.Config.CAFile, 1<<20)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(data) {
			return nil, hostedexecution.ErrConfiguration
		}
		transport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second}
	}
	token := options.Token
	if token == nil {
		token = func(context.Context) (string, error) {
			data, err := readPrivate(options.Config.TokenFile, 64<<10)
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(data)), nil
		}
	}
	return &Provider{discoveryCleanup: options.DiscoveryCleanup, config: options.Config, store: options.Store, inspector: options.Inspector, token: token, http: &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (p *Provider) Close() {
	p.http.CloseIdleConnections()
	if p.closeRegistry != nil {
		p.closeRegistry()
	}
}

func (p *Provider) InspectImage(ctx context.Context, reference string) (hostedexecution.Image, error) {
	image, err := p.inspector.InspectImage(ctx, reference)
	if err != nil {
		return image, hostedexecution.ErrUnavailable
	}
	if image.OS != "linux" || image.Architecture != p.config.Architecture {
		return image, hostedexecution.ErrConfiguration
	}
	image.ExecutionProfile = "pod-v1-" + fingerprint([]any{p.config.APIOrigin, p.config.Namespace, p.config.RuntimeClass, p.config.NodeSelector, "cpu2-memory2Gi"})[:20]
	return image, nil
}

var errConflict = errors.New("Kubernetes resource revision conflict")
var errAbsent = errors.New("Kubernetes resource does not exist")
var errRejected = errors.New("Kubernetes resource request was rejected")
