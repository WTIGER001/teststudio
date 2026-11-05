package localcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordRoundTrip(t *testing.T) {
	if os.Getuid() != 1000 {
		t.Fatal("the worker must run as its unprivileged identity")
	}
	want := struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}{Name: "Kubernetes worker", Count: 7}
	body, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "record.json")
	if err := os.WriteFile(file, body, 0600); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	got := want
	got.Name, got.Count = "", 0
	if err := json.Unmarshal(body, &got); err != nil || got != want {
		t.Fatalf("the retained record changed: %+v, %v", got, err)
	}
	t.Log("The local record survives its JSON and filesystem round trip.")
}
