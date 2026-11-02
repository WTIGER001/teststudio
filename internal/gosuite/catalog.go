package gosuite

import (
	"errors"
	"regexp"
	"strings"
)

type Entry struct {
	ID      string `json:"id"`
	Package string `json:"package"`
	Test    string `json:"test"`
}

type Manifest struct {
	Schema string  `json:"schemaVersion"`
	Tests  []Entry `json:"tests"`
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var packagePath = regexp.MustCompile(`^\.(?:/[A-Za-z0-9_.-]+)*$`)
var testName = regexp.MustCompile(`^(Test|Example|Fuzz)[A-Za-z0-9_]*(?:/[A-Za-z0-9_# .:=+-]+)*$`)

func Parse(selector string) (Entry, error) {
	pkg, test, ok := strings.Cut(selector, "::")
	if !ok || len(selector) > 1024 || !packagePath.MatchString(pkg) || !testName.MatchString(test) {
		return Entry{}, errors.New("Go selector requires one explicit local package and a registered test path")
	}
	for _, part := range strings.Split(pkg, "/") {
		if part == ".." || part == "..." {
			return Entry{}, errors.New("Go package selectors cannot traverse parents or recurse")
		}
	}
	return Entry{Package: pkg, Test: test}, nil
}

func (e Entry) Selector() string { return e.Package + "::" + e.Test }

func Validate(entries []Entry) error {
	if len(entries) < 1 || len(entries) > 1000 {
		return errors.New("Go catalog requires 1 through 1000 registered tests")
	}
	ids := map[string]bool{}
	for i, entry := range entries {
		if !identifier.MatchString(entry.ID) || ids[entry.ID] {
			return errors.New("Go catalog identifiers must be valid and unique")
		}
		ids[entry.ID] = true
		if _, err := Parse(entry.Selector()); err != nil {
			return err
		}
		for _, other := range entries[:i] {
			if entry.Package == other.Package && (entry.Test == other.Test || strings.HasPrefix(entry.Test, other.Test+"/") || strings.HasPrefix(other.Test, entry.Test+"/")) {
				return errors.New("register a Go parent aggregate or its independent subtests without overlapping selectors")
			}
		}
	}
	return nil
}

func Pattern(test string) string {
	parts := strings.Split(test, "/")
	for i, part := range parts {
		parts[i] = "^" + regexp.QuoteMeta(part) + "$"
	}
	return strings.Join(parts, "/")
}
