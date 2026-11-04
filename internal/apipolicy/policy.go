package apipolicy

import (
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode"
)

type Rule struct {
	Method       string `json:"method"`
	PathPrefix   string `json:"pathPrefix"`
	ReadOnly     bool   `json:"readOnly"`
	MaxBodyBytes int    `json:"maxBodyBytes"`
}

type Spec struct {
	Authentication   *Authentication `json:"authentication,omitempty"`
	RedactFields     []string        `json:"redactFields,omitempty"`
	Rules            []Rule          `json:"rules"`
	MaxRequests      int             `json:"maxRequests"`
	MaxResponseBytes int             `json:"maxResponseBytes"`
}

var fieldName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{0,63}$`)

func canonicalPath(value string) bool {
	return len(value) > 0 && len(value) <= 2048 && strings.HasPrefix(value, "/") && !strings.ContainsAny(value, "\\%?#") && !strings.ContainsFunc(value, unicode.IsControl) && path.Clean(value) == value
}

func Validate(spec *Spec) error {
	if spec == nil {
		return nil
	}
	if err := ValidateAuthentication(spec.Authentication); err != nil {
		return err
	}
	if len(spec.RedactFields) > 32 {
		return errors.New("API diagnostics support at most 32 additional sensitive field names")
	}
	fields := map[string]bool{}
	for _, field := range spec.RedactFields {
		if !fieldName.MatchString(field) || fields[strings.ToLower(field)] {
			return errors.New("API sensitive field names must be unique identifiers within 64 characters")
		}
		fields[strings.ToLower(field)] = true
	}
	if len(spec.Rules) < 1 || len(spec.Rules) > 64 || spec.MaxRequests < 1 || spec.MaxRequests > 1000 || spec.MaxResponseBytes < 1 || spec.MaxResponseBytes > 4<<20 {
		return errors.New("API policy requires 1 through 64 rules, 1 through 1000 requests, and a response limit within 4 MiB")
	}
	seen := map[string]bool{}
	for _, rule := range spec.Rules {
		key := rule.Method + " " + rule.PathPrefix
		if !canonicalPath(rule.PathPrefix) || seen[key] || rule.MaxBodyBytes < 0 || rule.MaxBodyBytes > 1<<20 {
			return errors.New("API rules require unique methods and canonical path prefixes with body limits within 1 MiB")
		}
		seen[key] = true
		switch rule.Method {
		case "GET", "HEAD", "OPTIONS":
			if rule.MaxBodyBytes != 0 {
				return errors.New("API read methods cannot carry request bodies")
			}
		case "POST":
		case "PUT", "PATCH", "DELETE":
			if rule.ReadOnly {
				return errors.New("mutating API methods cannot be labeled read-only")
			}
		default:
			return errors.New("API request method is unsupported")
		}
	}
	return nil
}

func ReadOnly(spec *Spec) bool {
	if spec == nil || Validate(spec) != nil {
		return false
	}
	for _, rule := range spec.Rules {
		if !rule.ReadOnly {
			return false
		}
	}
	return true
}

func Match(spec *Spec, method string, target *url.URL) (Rule, error) {
	if spec == nil || target == nil {
		return Rule{}, errors.New("API request path is outside the reviewed policy")
	}
	validPath := canonicalPath(target.Path) || len(target.Path) > 1 && strings.HasSuffix(target.Path, "/") && canonicalPath(strings.TrimSuffix(target.Path, "/"))
	if !validPath || target.RawPath != "" || target.Opaque != "" || target.User != nil || target.Fragment != "" || len(target.RawQuery) > 4096 || strings.ContainsFunc(target.RawQuery, unicode.IsControl) {
		return Rule{}, errors.New("API request path is outside the reviewed policy")
	}
	var selected *Rule
	for _, rule := range spec.Rules {
		if rule.Method == method && (target.Path == rule.PathPrefix || rule.PathPrefix == "/" || strings.HasPrefix(target.Path, rule.PathPrefix+"/")) && (selected == nil || len(rule.PathPrefix) > len(selected.PathPrefix)) {
			copy := rule
			selected = &copy
		}
	}
	if selected == nil {
		return Rule{}, errors.New("API request method or path is outside the reviewed policy")
	}
	return *selected, nil
}
