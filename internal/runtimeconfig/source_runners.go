package runtimeconfig

import (
	"errors"
	"strings"
)

func SourceRunners(value string) ([]string, error) {
	runners := []string{}
	seen := map[string]bool{}
	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(name)
		if name != "go" && name != "playwright" || seen[name] {
			return nil, errors.New("source bundle runners must select go or playwright without duplicates")
		}
		seen[name] = true
		runners = append(runners, name)
	}
	return runners, nil
}
