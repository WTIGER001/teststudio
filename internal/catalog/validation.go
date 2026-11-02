package catalog

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/timing"
)

var imageDigest = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64}$`)

func ValidateTest(t Test) error {
	if !domain.ValidID(t.ID) || strings.TrimSpace(t.Title) == "" {
		return errors.New("valid test identifier and title are required")
	}
	if !slices.Contains([]string{"browser", "https", "api", "unit", "integration", "accessibility", "visual", "performance"}, t.Type) {
		return errors.New("unsupported test type")
	}
	if !slices.Contains([]string{"http", "playwright", "go"}, t.Runner) {
		return errors.New("unsupported runner")
	}
	if !slices.Contains([]string{"draft", "enabled", "disabled", "archived"}, t.State) {
		return errors.New("invalid test lifecycle state")
	}
	if !slices.Contains([]string{"safe", "unsafe", "needs-review"}, t.Safety) {
		return errors.New("safety label is required")
	}
	if t.TimeoutSeconds < 1 || t.TimeoutSeconds > 3600 {
		return errors.New("timeout must be between 1 and 3600 seconds")
	}
	if t.Timing != nil {
		if err := timing.Validate(*t.Timing); err != nil {
			return err
		}
	}
	if t.Capture != nil {
		if err := capture.Validate(*t.Capture, capture.DefaultLimits()); err != nil {
			return err
		}
	}
	return nil
}

func Validate(d Draft) []Issue {
	issues := []Issue{}
	add := func(id, field, message string) {
		issues = append(issues, Issue{TestID: id, Field: field, Message: message})
	}
	if len(d.Tests) == 0 {
		add("", "tests", "catalog contains no tests")
	}
	if len(d.Tests) > 1000 {
		add("", "tests", "catalog exceeds 1000 entries")
	}
	if err := capture.Validate(d.Capture, capture.DefaultLimits()); err != nil {
		add("", "capture", err.Error())
	}
	if err := timing.Validate(d.Timing); err != nil {
		add("", "timing", err.Error())
	}
	tests := map[string]Test{}
	for _, t := range d.Tests {
		if err := ValidateTest(t); err != nil {
			add(t.ID, "definition", err.Error())
		}
		if _, found := tests[t.ID]; found {
			add(t.ID, "id", "duplicate identifier")
		}
		tests[t.ID] = t
		if t.Category == "" {
			add(t.ID, "category", "category is required")
		}
		if strings.TrimSpace(t.Rationale) == "" {
			add(t.ID, "rationale", "safety rationale is required")
		}
		if t.Runner == "http" {
			if t.Type != "https" && t.Type != "api" {
				add(t.ID, "runner", "HTTP runner requires HTTPS or API type")
			}
			if t.Check == nil {
				add(t.ID, "check", "HTTP definition is required")
				continue
			}
			c := t.Check
			u, err := url.Parse(c.Path)
			if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(c.Path, "/") || strings.HasPrefix(c.Path, "//") || u.Fragment != "" {
				add(t.ID, "check.path", "use a relative path beginning with one slash")
			}
			if !domain.ValidID(c.Endpoint) {
				add(t.ID, "check.endpoint", "registered endpoint name is required")
			}
			if c.Method != "GET" && c.Method != "HEAD" {
				add(t.ID, "check.method", "native health checks support GET and HEAD")
			}
			if c.Method == "HEAD" && (c.RequiredText != "" || c.ForbiddenText != "" || len(c.JSON) > 0) {
				add(t.ID, "check", "HEAD cannot assert response bodies")
			}
			if len(c.StatusCodes) == 0 {
				add(t.ID, "check.statusCodes", "accepted status codes are required")
			}
			for _, status := range c.StatusCodes {
				if status < 100 || status > 599 {
					add(t.ID, "check.statusCodes", "invalid HTTP status")
				}
			}
		} else {
			if t.Source == "" || t.Selector == "" {
				add(t.ID, "source", "coded tests require source and selector")
			}
			if !imageDigest.MatchString(d.Artifact.Image) {
				add(t.ID, "artifact.image", "coded tests require an immutable image digest")
			}
			if d.Artifact.Discovered[t.ID] != t.Selector {
				add(t.ID, "selector", "selector is absent from the discovered artifact catalog")
			}
		}
	}
	state := map[string]int{}
	var visit func(string)
	visit = func(id string) {
		if state[id] == 2 {
			return
		}
		if state[id] == 1 {
			add(id, "dependencies", "dependency cycle")
			return
		}
		state[id] = 1
		for _, dep := range tests[id].Dependencies {
			if _, found := tests[dep]; !found {
				add(id, "dependencies", "unknown dependency "+dep)
				continue
			}
			visit(dep)
		}
		state[id] = 2
	}
	for _, t := range d.Tests {
		visit(t.ID)
	}
	return issues
}

func ResolveSelection(tests []Test, ids []string, protected bool) ([]Test, error) {
	available := map[string]Test{}
	for _, t := range tests {
		available[t.ID] = t
	}
	if len(ids) == 0 {
		for _, t := range tests {
			if t.State == "enabled" {
				ids = append(ids, t.ID)
			}
		}
	}
	selected := map[string]bool{}
	result := []Test{}
	var visit func(string) error
	visiting := map[string]bool{}
	visit = func(id string) error {
		if selected[id] {
			return nil
		}
		if visiting[id] {
			return errors.New("dependency cycle")
		}
		t, ok := available[id]
		if !ok {
			return fmt.Errorf("unknown selected test %s", id)
		}
		if t.State != "enabled" {
			return fmt.Errorf("test %s is %s", id, t.State)
		}
		if protected && t.Safety != "safe" {
			return fmt.Errorf("test %s is not safe for the protected target", id)
		}
		visiting[id] = true
		for _, dep := range t.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		selected[id] = true
		result = append(result, t)
		return nil
	}
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	if len(result) == 0 {
		return nil, errors.New("no selected tests")
	}
	return result, nil
}
