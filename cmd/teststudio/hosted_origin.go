package main

import (
	"net/url"
	"strings"
)

func sameHTTPOrigin(left, right string) bool {
	a, err := url.Parse(left)
	if err != nil {
		return false
	}
	b, err := url.Parse(right)
	if err != nil {
		return false
	}
	port := func(value *url.URL) string {
		if value.Port() != "" {
			return value.Port()
		}
		if value.Scheme == "https" {
			return "443"
		}
		if value.Scheme == "http" {
			return "80"
		}
		return ""
	}
	return a.Scheme == b.Scheme && strings.EqualFold(a.Hostname(), b.Hostname()) && port(a) == port(b)
}
