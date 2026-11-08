package transfer

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/testkit"
)

func TestRecordingWorkerCutoverWaitsForCleanupAndExcludesRuntimeAuthority(t *testing.T) {
	db, err := testkit.Open(t, filepath.Join(t.TempDir(), "recording-cutover.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	changes := []storage.Change{}
	for _, item := range []struct {
		bucket, id string
		value      any
	}{{"recording-worker-pools", "worker", map[string]any{"instance": "one", "until": time.Now().Add(time.Minute)}}, {"recording-worker-leases", "session", map[string]any{"cleanupPending": true}}, {"browser-recordings", "session", map[string]string{"state": "ready", "code": "retained recording"}}} {
		change, _ := storage.Put(item.bucket, item.id, "", 0, item.value)
		changes = append(changes, change)
	}
	if err := db.Commit(t.Context(), changes...); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if _, err := Export(t.Context(), db, &output); !errors.Is(err, ErrBusy) || output.Len() != 0 {
		t.Fatal("active worker entered cutover", err)
	}
	pool, _ := storage.Put("recording-worker-pools", "worker", "", 1, map[string]any{"instance": "one", "until": time.Now().Add(-time.Minute)})
	if err := db.Commit(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(t.Context(), db, &output); !errors.Is(err, ErrBusy) {
		t.Fatal("pending cleanup entered cutover", err)
	}
	lease, _ := storage.Put("recording-worker-leases", "session", "", 1, map[string]any{"cleanupPending": false})
	if err := db.Commit(t.Context(), lease); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(t.Context(), db, &output); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(output.Bytes(), []byte("recording-worker-")) || !bytes.Contains(output.Bytes(), []byte("retained recording")) {
		t.Fatal("cutover copied runtime ownership or lost private history")
	}
	valid := Record{Bucket: "ordinary-history", ID: "retained", Revision: 1, Payload: `{}`, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if !validRecord(valid) {
		t.Fatal("valid ordinary history was rejected")
	}
	valid.Bucket = "recording-worker-leases"
	if validRecord(valid) {
		t.Fatal("import accepted recording lease ownership")
	}
	valid.Bucket = "recording-worker-pools"
	if validRecord(valid) {
		t.Fatal("import accepted runtime ownership")
	}
}
