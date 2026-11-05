package runtimeconfig

import (
	"errors"
	"net/netip"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

func ValidDatabaseHost(host string) bool {
	if len(host) < 3 || len(host) > 253 || host != strings.ToLower(host) || !strings.Contains(host, ".") {
		return false
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if char != '-' && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
				return false
			}
		}
	}
	return true
}

func ValidateDatabaseHost(connection, host string, environment []string) error {
	if host == "" {
		return nil
	}
	invalid := errors.New("database-host requires one complete PostgreSQL URL for the selected DNS hostname, verified TLS, and no routing or PostgreSQL environment overrides")
	if !ValidDatabaseHost(host) || len(connection) > 16<<10 || strings.ContainsAny(connection, "\x00\r\n") {
		return invalid
	}
	for _, entry := range environment {
		name, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "PG") && value != "" {
			return invalid
		}
	}
	u, err := url.Parse(connection)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Opaque != "" || u.Fragment != "" || u.User == nil || (u.Host != host && u.Host != host+":5432") || u.Path == "" || u.Path == "/" || !strings.HasPrefix(u.Path, "/") || strings.Contains(u.Path[1:], "/") || strings.ContainsAny(u.Path, "\x00\r\n") {
		return invalid
	}
	password, present := u.User.Password()
	if !present || password == "" || u.User.Username() == "" || strings.ContainsAny(u.User.Username()+password, "\x00\r\n") {
		return invalid
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || query.Get("sslmode") != "verify-full" {
		return invalid
	}
	for key, values := range query {
		if len(values) != 1 || strings.ContainsAny(values[0], "\x00\r\n") {
			return invalid
		}
		switch key {
		case "sslmode":
		case "sslrootcert":
			if values[0] != "system" && (!filepath.IsAbs(values[0]) || filepath.Clean(values[0]) != values[0] || len(values[0]) > 1024) {
				return invalid
			}
		case "connect_timeout":
			number, err := strconv.Atoi(values[0])
			if err != nil || number < 1 || number > 30 || strconv.Itoa(number) != values[0] {
				return invalid
			}
		case "application_name":
			if len(values[0]) == 0 || len(values[0]) > 64 {
				return invalid
			}
		default:
			return invalid
		}
	}
	return nil
}
