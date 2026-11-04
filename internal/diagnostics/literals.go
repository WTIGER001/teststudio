package diagnostics

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
)

func LiteralReplacer(values []string) *strings.Replacer {
	variants := []string{}
	for _, value := range values {
		if value == "" {
			continue
		}
		encoded, _ := json.Marshal(value)
		variants = append(variants, value, string(encoded[1:len(encoded)-1]), url.QueryEscape(value), url.PathEscape(value), base64.StdEncoding.EncodeToString([]byte(value)), base64.RawURLEncoding.EncodeToString([]byte(value)))
	}
	slices.SortFunc(variants, func(a, b string) int { return len(b) - len(a) })
	pairs := []string{Mask, Mask}
	seen := map[string]bool{Mask: true}
	for _, value := range variants {
		if !seen[value] {
			pairs = append(pairs, value, Mask)
			seen[value] = true
		}
	}
	return strings.NewReplacer(pairs...)
}
