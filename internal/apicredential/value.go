package apicredential

import (
	"context"
	"encoding/base64"
	"errors"
	"net/textproto"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
)

var ErrUnavailable = errors.New("the selected API credential is unavailable or outside its approved scope")
var bearer = regexp.MustCompile(`^[A-Za-z0-9._~+/=-]+$`)

type Value struct {
	Origin  string             `json:"origin"`
	Header  string             `json:"header"`
	Value   string             `json:"value"`
	Network networkguard.Scope `json:"network"`
}

func Resolve(ctx context.Context, provider secrets.Provider, project, reference string, target browserpolicy.Target) (*Value, error) {
	if !target.Authenticated {
		return nil, nil
	}
	if provider == nil || reference == "" || browserpolicy.ValidateTarget(target) != nil {
		return nil, ErrUnavailable
	}
	secret, err := provider.Resolve(ctx, project, reference, secrets.APICredential)
	if err != nil || !slices.Contains(secret.Origins, target.Origin) {
		return nil, ErrUnavailable
	}
	auth := target.Requests.Authentication
	raw := string(secret.Bytes)
	if len(raw) == 0 || len(raw) > 4096 || strings.TrimSpace(raw) != raw || !utf8.ValidString(raw) || strings.ContainsFunc(raw, unicode.IsControl) {
		return nil, ErrUnavailable
	}
	value := &Value{Origin: target.Origin, Network: secret.Network}
	switch auth.Scheme {
	case "bearer":
		if !bearer.MatchString(raw) {
			return nil, ErrUnavailable
		}
		value.Header, value.Value = "Authorization", "Bearer "+raw
	case "basic":
		user, _, ok := strings.Cut(raw, ":")
		if !ok || user == "" {
			return nil, ErrUnavailable
		}
		value.Header, value.Value = "Authorization", "Basic "+base64.StdEncoding.EncodeToString(secret.Bytes)
	case "header":
		value.Header, value.Value = textproto.CanonicalMIMEHeaderKey(auth.Header), raw
	default:
		return nil, ErrUnavailable
	}
	if Validate(target, value) != nil {
		return nil, ErrUnavailable
	}
	return value, nil
}

func Validate(target browserpolicy.Target, value *Value) error {
	if !target.Authenticated {
		if value != nil {
			return ErrUnavailable
		}
		return nil
	}
	if value == nil || value.Origin != target.Origin || target.Requests == nil || target.Requests.Authentication == nil || (value.Network != networkguard.Public && value.Network != networkguard.Private) || strings.HasPrefix(target.Origin, "http:") && !target.AllowCredentialHTTP {
		return ErrUnavailable
	}
	auth := target.Requests.Authentication
	header := textproto.CanonicalMIMEHeaderKey(auth.Header)
	if auth.Scheme != "header" {
		header = "Authorization"
	}
	if value.Header != header || len(value.Value) < 1 || len(value.Value) > 8192 || strings.ContainsFunc(value.Value, func(r rune) bool { return r < 32 || r > 126 }) {
		return ErrUnavailable
	}
	if auth.Scheme == "bearer" && (!strings.HasPrefix(value.Value, "Bearer ") || len(strings.TrimPrefix(value.Value, "Bearer ")) > 4096 || !bearer.MatchString(strings.TrimPrefix(value.Value, "Bearer "))) {
		return ErrUnavailable
	}
	if auth.Scheme == "basic" {
		if !strings.HasPrefix(value.Value, "Basic ") {
			return ErrUnavailable
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value.Value, "Basic "))
		if err != nil || !strings.Contains(string(decoded), ":") {
			return ErrUnavailable
		}
	}
	return nil
}

func (v *Value) PrivateValues() []string {
	if v == nil {
		return nil
	}
	values := []string{v.Value}
	if v.Header == "Authorization" {
		if raw, ok := strings.CutPrefix(v.Value, "Bearer "); ok {
			values = append(values, raw)
		}
		if raw, ok := strings.CutPrefix(v.Value, "Basic "); ok {
			values = append(values, raw)
			if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
				values = append(values, string(decoded))
				if _, password, ok := strings.Cut(string(decoded), ":"); ok && password != "" {
					values = append(values, password)
				}
			}
		}
	}
	return values
}
