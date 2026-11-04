package ecsworker

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type sdkHTTP struct {
	base   aws.HTTPClient
	region string
}

type sdkBody struct {
	io.Reader
	io.Closer
}

func privateHTTP() *http.Client {
	return &http.Client{
		Transport:     &http.Transport{DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 4 * time.Second, IdleConnTimeout: 30 * time.Second, MaxIdleConns: 32, MaxIdleConnsPerHost: 8},
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func (c *sdkHTTP) Do(request *http.Request) (*http.Response, error) {
	services := []string{"sts", "ec2", "ecs", "api.ecr", "elasticloadbalancing", "scheduler", "route53resolver", "route53profiles"}
	allowed := false
	for _, service := range services {
		if request.URL.Host == service+"."+c.region+".amazonaws.com" {
			allowed = true
		}
	}
	if !allowed || request.URL.Scheme != "https" || request.URL.User != nil || request.URL.Fragment != "" || strings.ContainsAny(request.URL.Host, "\\\r\n\x00") || request.Host != "" && request.Host != request.URL.Host {
		return nil, hostedexecution.ErrConfiguration
	}
	response, err := c.base.Do(request)
	if err != nil {
		return nil, hostedexecution.ErrUnavailable
	}
	if response == nil || response.Body == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 || response.ContentLength > 8<<20 {
		response.Body.Close()
		return nil, hostedexecution.ErrUnavailable
	}
	response.Body = &sdkBody{Reader: io.LimitReader(response.Body, 8<<20), Closer: response.Body}
	return response, nil
}

func (c *sdkHTTP) CloseIdleConnections() {
	if close, ok := c.base.(interface{ CloseIdleConnections() }); ok {
		close.CloseIdleConnections()
	}
}
