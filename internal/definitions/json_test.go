package definitions_test

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/definitions"
)

func TestStrictJSONLimitsAndDuplicateKeysApplyBeforeDecoding(t *testing.T) {
	for _, item := range []struct {
		name, source        string
		bytes, depth, nodes int
	}{
		{"duplicate", `{"nested":{"key":1,"key":2}}`, 1024, 10, 100},
		{"bytes", `{"value":"oversized"}`, 4, 10, 100},
		{"nodes", `[1,2,3,4]`, 1024, 10, 4},
		{"depth", strings.Repeat("[", 20) + strings.Repeat("]", 20), 1024, 10, 100},
		{"trailing", `{} {}`, 1024, 10, 100},
		{"invalid", `[}`, 1024, 10, 100},
	} {
		t.Run(item.name, func(t *testing.T) {
			var value any
			if err := definitions.DecodeJSON([]byte(item.source), &value, item.bytes, item.depth, item.nodes); err == nil {
				t.Fatal("invalid or excessive JSON accepted")
			}
		})
	}
	var value struct {
		Text string `json:"text"`
	}
	if err := definitions.DecodeJSON([]byte(`{"text":"<retained UTF-8>"}`), &value, 1024, 10, 100); err != nil || value.Text != "<retained UTF-8>" {
		t.Fatal("valid document rejected", err)
	}
	if err := definitions.DecodeJSON([]byte(`{"text":"value","extra":true}`), &value, 1024, 10, 100); err == nil {
		t.Fatal("unknown fields accepted")
	}
}
