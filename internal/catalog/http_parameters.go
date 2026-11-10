package catalog

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var parameterReference = regexp.MustCompile(`\{\{([A-Za-z0-9][A-Za-z0-9._-]{0,127})\}\}`)

func HTTPParameterNames(check *HTTPCheck) []string {
	names := []string{}
	if check == nil {
		return names
	}
	for _, value := range []string{check.Path, check.RequiredText, check.ForbiddenText} {
		for _, match := range parameterReference.FindAllStringSubmatch(value, -1) {
			if !slices.Contains(names, match[1]) {
				names = append(names, match[1])
			}
		}
	}
	return names
}

func ResolveHTTPParameters(test Test, parameters map[string]string) (Test, error) {
	if test.Check == nil {
		return test, nil
	}
	for _, name := range HTTPParameterNames(test.Check) {
		if !slices.Contains(test.RequiredParameters, name) {
			return test, fmt.Errorf("declare parameter %s in requiredParameters", name)
		}
		if value, found := parameters[name]; !found || strings.TrimSpace(value) == "" {
			return test, fmt.Errorf("required parameter %s is missing", name)
		}
	}
	replace := func(value string, escape func(string) string) string {
		return parameterReference.ReplaceAllStringFunc(value, func(match string) string {
			value := parameters[match[2:len(match)-2]]
			if escape != nil {
				return escape(value)
			}
			return value
		})
	}
	check := *test.Check
	path, query, found := strings.Cut(check.Path, "?")
	check.Path = replace(path, url.PathEscape)
	if found {
		check.Path += "?" + replace(query, url.QueryEscape)
	}
	check.RequiredText, check.ForbiddenText = replace(check.RequiredText, nil), replace(check.ForbiddenText, nil)
	test.Check = &check
	return test, nil
}
