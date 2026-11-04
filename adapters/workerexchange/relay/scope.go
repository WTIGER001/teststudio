package relay

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/wtiger001/teststudio/internal/identifier"
	"github.com/wtiger001/teststudio/internal/workerwire"
)

type operation struct {
	path, execution, token, instance string
	limit                            int64
	input                            bool
}

func origin(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != raw {
		return nil, errConfiguration
	}
	if rawPort := u.Port(); rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return nil, errConfiguration
		}
	}
	if _, err := http.NewRequest(http.MethodGet, raw, nil); err != nil {
		return nil, errConfiguration
	}
	return u, nil
}

func scope(r *http.Request) (operation, bool) {
	var op operation
	if r.URL.IsAbs() || r.URL.Opaque != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.Fragment != "" || len(r.Header.Values("Cookie"))+len(r.Header.Values("Origin"))+len(r.Header.Values("Upgrade"))+len(r.Header.Values("Content-Encoding"))+len(r.Trailer) != 0 {
		return op, false
	}
	pieces := strings.Split(r.URL.Path, "/")
	if len(pieces) < 7 || pieces[0] != "" || pieces[1] != "internal" || pieces[2] != "worker" || pieces[3] != "v1" || pieces[4] != "executions" || !identifier.Valid(pieces[5]) {
		return op, false
	}
	op.execution = pieces[5]
	op.path = "/internal/worker/v1/executions/" + url.PathEscape(op.execution) + "/" + pieces[6]
	switch {
	case len(pieces) == 7 && pieces[6] == "input" && r.Method == http.MethodGet:
		op.input = true
	case len(pieces) == 7 && pieces[6] == "started" && r.Method == http.MethodPost:
		op.limit = 1024
	case len(pieces) == 7 && pieces[6] == "report" && r.Method == http.MethodPut:
		op.limit = workerwire.MaxSubmissionBytes
	case len(pieces) == 8 && pieces[6] == "attachments" && r.Method == http.MethodPut:
		name := pieces[7]
		if name == "" || name == "." || name == ".." || len(name) > 255 || strings.ContainsAny(name, "/\\\x00\r\n") {
			return op, false
		}
		op.path += "/" + url.PathEscape(name)
		op.limit = 64 << 20
	default:
		return op, false
	}
	if r.URL.EscapedPath() != op.path || len(r.Header.Values("Authorization")) != 1 || len(r.Header.Values("X-TestStudio-Worker-Instance")) != 1 {
		return op, false
	}
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || len(token) != 43 {
		return op, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != token {
		return op, false
	}
	instance := r.Header.Get("X-TestStudio-Worker-Instance")
	if !identifier.Valid(instance) {
		return op, false
	}
	op.token, op.instance = token, instance
	return op, true
}

func authority(u *url.URL) string {
	port := 443
	if u.Port() != "" {
		port, _ = strconv.Atoi(u.Port())
	}
	return strings.ToLower(u.Hostname()) + ":" + strconv.Itoa(port)
}
