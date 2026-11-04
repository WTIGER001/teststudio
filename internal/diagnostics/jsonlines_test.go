package diagnostics

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestGoJSONEventsRedactDecodedOutputAndPreserveEventStructure(t *testing.T) {
	first, _ := json.Marshal(map[string]any{"Action": "output", "Test": "TestAPI", "Output": "response: {\"token\":\"event-canary\",\"status\":\"up\"}\n"})
	second, _ := json.Marshal(map[string]any{"Action": "output", "Test": "TestAPI", "Output": "actual event-canary"})
	data := append(append(first, '\n'), second...)
	clean, err := New(nil, nil).JSONLines(data)
	if err != nil || bytes.Contains(clean, []byte("canary")) || !bytes.Contains(clean, []byte("TestAPI")) || !bytes.Contains(clean, []byte("up")) {
		t.Fatal("Go event redaction loses context or exposes a value", err)
	}
	for _, line := range bytes.Split(bytes.TrimSpace(clean), []byte{'\n'}) {
		if !json.Valid(line) {
			t.Fatal("redaction corrupts a JSON event")
		}
	}
	if _, err := New(nil, nil).JSONLines(append(data, []byte("\n{unfinished")...)); err == nil {
		t.Fatal("truncated JSON event stream accepted")
	}
	if _, err := New(nil, nil).JSONLines([]byte(strings.Repeat("{}\n", 4097))); err == nil {
		t.Fatal("unbounded event stream accepted")
	}
}
