package recordingworker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wtiger001/teststudio/internal/storage"
)

func TestOriginPolicyRequiresBoundedExactOriginsAndFreezesTheLease(t *testing.T) {
	file := filepath.Join(t.TempDir(), "origins.json")
	for _, value := range []string{`{"https://site.test/path":["https://api.test"]}`, `{"https://site.test":["https://*.test"]}`, `{"https://site.test":["https://api.test/path"]}`, `{"https://site.test":["https://site.test"]}`, `{"https://site.test":["https://api.test","https://api.test"]}`, `{"https://site.test":["https://user:secret@api.test"]}`, `{} {}`} {
		if err := os.WriteFile(file, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOriginPolicy(file); err == nil {
			t.Fatal("invalid origin policy was accepted")
		}
	}
	if err := os.WriteFile(file, []byte(`{"https://example.test":["https://static.test","https://api.test"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	policy, err := LoadOriginPolicy(file)
	if err != nil {
		t.Fatal(err)
	}
	w, _, _ := setupWorker(t)
	w.Origins = policy
	b := newBinding(t, w, "origin-policy")
	if _, err := w.Call(t.Context(), "start", Request{Binding: b}); err != nil {
		t.Fatal(err)
	}
	row, err := w.Store.Get(t.Context(), LeaseBucket, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	lease, _ := storage.Decode[Lease](row)
	want := []string{"https://api.test", "https://static.test"}
	if !reflect.DeepEqual(lease.Target.AdditionalOrigins, want) {
		t.Fatal("reserved network policy did not retain the reviewed origins")
	}
	w.Origins["https://example.test"] = []string{"https://later.test"}
	if _, err := w.Call(t.Context(), "start", Request{Binding: b}); err != nil {
		t.Fatal(err)
	}
	row, _ = w.Store.Get(t.Context(), LeaseBucket, b.ID)
	lease, _ = storage.Decode[Lease](row)
	if !reflect.DeepEqual(lease.Target.AdditionalOrigins, want) {
		t.Fatal("an existing recording silently adopted a different origin policy")
	}
	if err := w.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
}
