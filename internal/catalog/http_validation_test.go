package catalog_test

import (
	"math"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestHTTPPublicationRejectsInvalidAssertionsAndLimits(t *testing.T) {
	cases := []struct {
		name, field string
		change      func(*catalog.HTTPCheck)
	}{
		{"response-time", "check.maxResponseMs", func(c *catalog.HTTPCheck) { c.MaxResponseMs = -1 }},
		{"certificate-days", "check.minCertificateDays", func(c *catalog.HTTPCheck) { c.MinCertificateDays = 36501 }},
		{"status", "check.statusCodes", func(c *catalog.HTTPCheck) { c.StatusCodes = []int{600} }},
		{"header-type", "check.headers", func(c *catalog.HTTPCheck) { c.Headers = map[string]string{"Invalid Name": "value"} }},
		{"duplicate-header", "check.headers", func(c *catalog.HTTPCheck) { c.Headers = map[string]string{"X-Ready": "yes", "x-ready": "yes"} }},
		{"header-line", "check.headers", func(c *catalog.HTTPCheck) { c.Headers = map[string]string{"X-Ready": "yes\nno"} }},
		{"body-size", "check", func(c *catalog.HTTPCheck) { c.RequiredText = strings.Repeat("x", 8193) }},
		{"json-path", "check.json", func(c *catalog.HTTPCheck) { c.JSON = map[string]any{"status..value": "up"} }},
		{"json-value", "check.json", func(c *catalog.HTTPCheck) { c.JSON = map[string]any{"status": math.Inf(1)} }},
		{"head-body", "check", func(c *catalog.HTTPCheck) { c.Method = "HEAD"; c.JSON = map[string]any{"status": "up"} }},
		{"fragment", "check.path", func(c *catalog.HTTPCheck) { c.Path = "/health#hidden" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			definition := check("HTTPS-1")
			tc.change(definition.Check)
			draft := catalog.Draft{Tests: []catalog.Test{definition}}
			found := false
			for _, issue := range catalog.Validate(draft) {
				if issue.TestID == definition.ID && issue.Field == tc.field {
					found = true
				}
			}
			if !found {
				t.Fatalf("invalid %s was not rejected at %s", tc.name, tc.field)
			}
		})
	}
}

func TestHTTPInvalidDraftCannotBecomeAnActiveRelease(t *testing.T) {
	s, _ := setup(t)
	definition := check("HTTPS-1")
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", definition, 0); err != nil {
		t.Fatal(err)
	}
	draft, err := s.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	active, err := s.Publish(t.Context(), "admin", "alpha", "health", draft.ID, "valid-assertions", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	definition.Check.Headers = map[string]string{"Invalid Name": "value"}
	if err := s.SaveTest(t.Context(), "admin", "alpha", "health", definition, 1); err != nil {
		t.Fatal(err)
	}
	draft, err = s.CreateDraft(t.Context(), "admin", "alpha", "health", catalog.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "health", draft.ID, "invalid-assertions", 1, true); err == nil {
		t.Fatal("invalid assertions became an active release")
	}
	row, err := s.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, err := storage.Decode[projects.Suite](row)
	if err != nil || suite.ActiveRelease != active.ID {
		t.Fatal("invalid publication displaced the active release")
	}
}
