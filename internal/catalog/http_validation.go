package catalog

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
)

var headerName = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")

func ValidateHTTPCheck(c *HTTPCheck) []Issue {
	issues := []Issue{}
	add := func(field, message string) { issues = append(issues, Issue{Field: field, Message: message}) }
	if c == nil {
		add("check", "HTTP definition is required")
		return issues
	}
	u, err := url.Parse(c.Path)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(c.Path, "/") || strings.HasPrefix(c.Path, "//") || u.Fragment != "" {
		add("check.path", "use a relative path beginning with one slash and no fragment")
	}
	if !domain.ValidID(c.Endpoint) {
		add("check.endpoint", "registered endpoint name is required")
	}
	if c.Method != "GET" && c.Method != "HEAD" {
		add("check.method", "native health checks support GET and HEAD")
	}
	if c.Method == "HEAD" && (c.RequiredText != "" || c.ForbiddenText != "" || len(c.JSON) > 0) {
		add("check", "HEAD cannot assert response bodies")
	}
	if len(c.StatusCodes) == 0 || len(c.StatusCodes) > 32 {
		add("check.statusCodes", "configure 1 through 32 accepted status codes")
	}
	for _, status := range c.StatusCodes {
		if status < 100 || status > 599 {
			add("check.statusCodes", "accepted status codes must be between 100 and 599")
			break
		}
	}
	if len(c.RequiredText) > 8192 || len(c.ForbiddenText) > 8192 {
		add("check", "response text assertions must not exceed 8192 bytes each")
	}
	if c.MaxResponseMs < 0 || c.MaxResponseMs > 3600000 {
		add("check.maxResponseMs", "maximum response time must be between 0 and 3600000 milliseconds")
	}
	if c.MinCertificateDays < 0 || c.MinCertificateDays > 36500 {
		add("check.minCertificateDays", "minimum certificate lifetime must be between 0 and 36500 days")
	}
	if len(c.Headers) > 32 {
		add("check.headers", "configure at most 32 expected response headers")
	}
	seen := map[string]bool{}
	for name, value := range c.Headers {
		key := strings.ToLower(name)
		if len(name) > 256 || !headerName.MatchString(name) || len(value) > 8192 || strings.ContainsAny(value, "\r\n\x00") || seen[key] {
			add("check.headers", "expected headers require unique valid names and bounded single-line string values")
			break
		}
		seen[key] = true
	}
	if len(c.JSON) > 32 {
		add("check.json", "configure at most 32 JSON field expectations")
	}
	for path := range c.JSON {
		if path == "" || len(path) > 256 || strings.ContainsAny(path, "\r\n\x00") || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") || strings.Contains(path, "..") {
			add("check.json", "JSON field paths require nonempty dot-separated segments within 256 bytes")
			break
		}
	}
	if body, err := json.Marshal(c.JSON); err != nil || len(body) > 65536 {
		add("check.json", "JSON field expectations must fit in 65536 bytes of valid JSON")
	}
	return issues
}
