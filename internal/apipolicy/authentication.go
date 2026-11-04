package apipolicy

import (
	"errors"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
)

type Authentication struct {
	Binding string `json:"binding"`
	Scheme  string `json:"scheme"`
	Header  string `json:"header,omitempty"`
}

var credentialHeader = regexp.MustCompile(`(?i)^(?:x-[a-z0-9][a-z0-9-]{0,61}|api-key)$`)

func ValidateAuthentication(auth *Authentication) error {
	if auth == nil {
		return nil
	}
	if !domain.ValidID(auth.Binding) {
		return errors.New("API authentication requires a valid credential binding")
	}
	switch auth.Scheme {
	case "bearer", "basic":
		if auth.Header != "" {
			return errors.New("standard API authentication uses its fixed Authorization header")
		}
	case "header":
		name := strings.ToLower(auth.Header)
		if !credentialHeader.MatchString(name) || strings.HasPrefix(name, "x-forwarded-") || name == "x-original-url" || name == "x-rewrite-url" || name == "x-http-method-override" || name == "x-method-override" || name == "x-http-method" {
			return errors.New("API credential header must be an X- header or Api-Key without routing or method overrides")
		}
	default:
		return errors.New("API authentication supports bearer, basic, or header")
	}
	return nil
}

func ValidateBindings(spec *Spec, bindings []string) error {
	if len(bindings) == 0 {
		return nil
	}
	if len(bindings) != 1 || spec == nil || spec.Authentication == nil || spec.Authentication.Binding != bindings[0] {
		return errors.New("API credential bindings must match the single reviewed authentication policy")
	}
	return ValidateAuthentication(spec.Authentication)
}
