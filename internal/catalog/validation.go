package catalog

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/gosuite"
	"github.com/wtiger001/teststudio/internal/timing"
)

var imageDigest = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64}$`)

var localImageDigest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

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
	for _, name := range HTTPParameterNames(t.Check) {
		if !slices.Contains(t.RequiredParameters, name) {
			return fmt.Errorf("declare parameter %s in requiredParameters", name)
		}
	}
	for _, names := range [][]string{t.RequiredParameters, t.RequiredBindings} {
		if len(names) > 64 {
			return errors.New("tests support at most 64 required parameters or secret bindings")
		}
		seen := map[string]bool{}
		for _, name := range names {
			if !domain.ValidID(name) || seen[name] {
				return errors.New("required configuration names must be valid and distinct")
			}
			seen[name] = true
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
	if err := browserpolicy.Validate(d.Artifact.Browser); err != nil {
		add("", "artifact.browser", err.Error())
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
	goEntries := []gosuite.Entry{}
	for _, t := range d.Tests {
		if d.Artifact.Browser != nil && d.Artifact.Browser.Requests != nil && t.Type != "api" {
			add(t.ID, "artifact.browser.requests", "request-restricted execution requires API test types")
		}
		if d.Artifact.Browser != nil {
			if t.Runner != "playwright" && (t.Runner != "go" || t.Type != "api" || d.Artifact.Browser.Requests == nil) {
				add(t.ID, "artifact.browser", "the environment profile requires Playwright tests or request-restricted Go API tests")
			}
			if err := apipolicy.ValidateBindings(d.Artifact.Browser.Requests, t.RequiredBindings); err != nil {
				add(t.ID, "requiredBindings", err.Error())
			}
		}
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
			for _, issue := range ValidateHTTPCheck(t.Check) {
				add(t.ID, issue.Field, issue.Message)
			}
		} else {
			if t.Runner == "go" {
				if t.Type != "unit" && t.Type != "integration" && t.Type != "api" {
					add(t.ID, "type", "Go runner requires unit, integration, or API type")
				}
				entry, err := gosuite.Parse(t.Selector)
				if err != nil {
					add(t.ID, "selector", err.Error())
				} else {
					entry.ID = t.ID
					goEntries = append(goEntries, entry)
				}
			}
			if t.Source == "" || t.Selector == "" {
				add(t.ID, "source", "coded tests require source and selector")
			}
			if !imageDigest.MatchString(d.Artifact.Image) && !localImageDigest.MatchString(d.Artifact.Image) {
				add(t.ID, "artifact.image", "coded tests require an immutable image digest")
			}
			if d.Artifact.Discovered[t.ID] != t.Selector {
				add(t.ID, "selector", "selector is absent from the discovered artifact catalog")
			}
		}
	}
	if len(goEntries) > 0 {
		if err := gosuite.Validate(goEntries); err != nil {
			add("", "selector", err.Error())
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
