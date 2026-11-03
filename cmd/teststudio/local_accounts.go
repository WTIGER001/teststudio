package main

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

func validateLocalAccountOrigin(origin *url.URL) error {
	if origin.Scheme == "https" {
		return nil
	}
	ip := net.ParseIP(origin.Hostname())
	if origin.Scheme == "http" && (strings.EqualFold(origin.Hostname(), "localhost") || (ip != nil && ip.IsLoopback())) {
		return nil
	}
	return errors.New("managed local accounts require an HTTPS public origin or loopback HTTP")
}
