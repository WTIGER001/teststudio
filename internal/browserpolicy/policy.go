package browserpolicy

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/domain"
)

type Spec struct {
	Endpoint string          `json:"endpoint"`
	Requests *apipolicy.Spec `json:"requests,omitempty"`
}

type Target struct {
	Origin       string          `json:"origin"`
	AllowPrivate bool            `json:"allowPrivateNetwork,omitempty"`
	Requests     *apipolicy.Spec `json:"requests,omitempty"`
}

func Validate(spec *Spec) error {
	if spec != nil && !domain.ValidID(spec.Endpoint) {
		return errors.New("browser execution requires one registered endpoint identifier")
	}
	if spec != nil {
		return apipolicy.Validate(spec.Requests)
	}
	return nil
}

func Origin(address string) (string, error) {
	u, err := url.Parse(address)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(address, "\\\r\n\t#") {
		return "", errors.New("browser target requires an exact credential-free HTTP or HTTPS origin")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if number, err := strconv.Atoi(port); err != nil || number < 1 || number > 65535 {
		return "", errors.New("browser target requires a valid TCP port")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if u.Scheme == "https" && port == "443" || u.Scheme == "http" && port == "80" {
		u.Host = host
		if strings.Contains(host, ":") {
			u.Host = "[" + host + "]"
		}
	} else {
		u.Host = net.JoinHostPort(host, port)
	}
	u.Path = ""
	return u.String(), nil
}

func ValidateTarget(target Target) error {
	if err := apipolicy.Validate(target.Requests); err != nil {
		return err
	}
	origin, err := Origin(target.Origin)
	if err != nil || origin != target.Origin {
		return errors.New("browser target requires a canonical origin")
	}
	u, _ := url.Parse(origin)
	ip := net.ParseIP(u.Hostname())
	if u.Hostname() == "localhost" || strings.HasSuffix(u.Hostname(), ".localhost") || ip != nil && (ip.IsLoopback() || ip.IsUnspecified()) {
		return errors.New("the container browser cannot use host localhost; register a worker-reachable origin such as host.docker.internal and allow private network access")
	}
	return nil
}
