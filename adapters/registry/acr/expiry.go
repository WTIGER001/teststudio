package acr

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/definitions"
)

func tokenExpiry(value string, now time.Time) (time.Time, error) {
	parts := strings.Split(value, ".")
	if len(value) > 65536 || len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" || strings.ContainsAny(value, "\r\n\x00") {
		return time.Time{}, oci.ErrInvalid
	}
	for _, part := range parts {
		for _, c := range part {
			if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
				return time.Time{}, oci.ErrInvalid
			}
		}
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	var claims map[string]json.RawMessage
	if err != nil || definitions.DecodeJSON(data, &claims, 64<<10, 12, 1000) != nil {
		return time.Time{}, oci.ErrInvalid
	}
	var expires int64
	if json.Unmarshal(claims["exp"], &expires) != nil || expires <= 0 || expires > 253402300799 {
		return time.Time{}, oci.ErrInvalid
	}
	if raw, ok := claims["nbf"]; ok {
		var at *int64
		if json.Unmarshal(raw, &at) != nil || at == nil || *at < 0 || *at > now.Add(30*time.Second).Unix() {
			return time.Time{}, oci.ErrInvalid
		}
	}
	until := time.Unix(expires, 0).Add(-30 * time.Second)
	if !until.After(now) {
		return time.Time{}, oci.ErrUnavailable
	}
	return until, nil
}
