package oidc

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"
)

func (s *Service) httpClient() *http.Client {
	client := http.DefaultClient
	if s.Client != nil {
		client = s.Client
	}
	bounded := *client
	if bounded.Timeout <= 0 || bounded.Timeout > 10*time.Second {
		bounded.Timeout = 10 * time.Second
	}
	original := bounded.CheckRedirect
	bounded.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if err := protocolEndpoint(request.URL.String()); err != nil {
			return err
		}
		if len(via) > 0 && via[0].URL.Scheme == "https" && request.URL.Scheme != "https" {
			return errors.New("OIDC redirects cannot downgrade HTTPS")
		}
		if original != nil {
			return original(request, via)
		}
		if len(via) >= 10 {
			return errors.New("OIDC redirect limit exceeded")
		}
		return nil
	}
	return &bounded
}

func protocolEndpoint(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || len(value) > 4096 {
		return errors.New("OIDC protocol endpoints require absolute credential-free URLs")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))) {
		return errors.New("OIDC protocol endpoints require HTTPS outside loopback")
	}
	return nil
}
