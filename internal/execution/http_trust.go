package execution

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
)

type HTTPTrustBundle struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

type HTTPTrustResolver interface {
	Resolve(context.Context, string, string) (*x509.CertPool, *HTTPTrustBundle, error)
}

type HTTPRuntimeProfiler interface {
	HTTPProfile(context.Context, string, catalog.Test, projects.Environment) (string, error)
}

func HTTPOrigin(address string) (string, error) {
	u, err := url.Parse(address)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Fragment != "" {
		return "", errors.New("registered HTTP target is invalid")
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", errors.New("registered HTTP target port is invalid")
		}
		port = strconv.Itoa(number)
	}
	if port == "" || u.Scheme == "https" && port == "443" || u.Scheme == "http" && port == "80" {
		if strings.Contains(host, ":") {
			host = "[" + host + "]"
		}
	} else {
		host = net.JoinHostPort(host, port)
	}
	return u.Scheme + "://" + host, nil
}

func (e *HTTP) trust(ctx context.Context, project string, test catalog.Test, environment projects.Environment) (*x509.CertPool, *HTTPTrustBundle, error) {
	if e.Trust == nil {
		return e.RootCAs, nil, nil
	}
	if test.Check == nil {
		return nil, nil, errors.New("HTTP trust requires a registered endpoint")
	}
	origin, err := HTTPOrigin(environment.Endpoints[test.Check.Endpoint])
	if err != nil {
		return nil, nil, err
	}
	roots, bundle, err := e.Trust.Resolve(ctx, project, origin)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, errors.New("the project HTTP trust configuration is unavailable")
	}
	if bundle != nil && environment.Stage != "local" && !environment.AllowPrivate {
		return nil, nil, errors.New("custom HTTP trust requires an explicitly permitted private environment")
	}
	if bundle == nil {
		roots = e.RootCAs
	}
	return roots, bundle, nil
}

func (e *HTTP) HTTPProfile(ctx context.Context, project string, test catalog.Test, environment projects.Environment) (string, error) {
	_, bundle, err := e.trust(ctx, project, test, environment)
	if err != nil {
		return "", err
	}
	if bundle == nil {
		return "native-http-v1", nil
	}
	return "native-http-v1:trust/" + bundle.ID + "/" + bundle.Digest, nil
}
