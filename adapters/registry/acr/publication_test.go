package acr

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/execution"
)

type journal struct{ opening, opened, completed int }

func (j *journal) Opening(string) error { j.opening++; return nil }
func (j *journal) Opened(string) error  { j.opened++; return nil }
func (j *journal) Completed() error     { j.completed++; return nil }

func TestACRPublicationUsesSeparateWriteScopeAndNormalUploadJournal(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f := fresh(t, runner, true, true)
			runtime, err := f.client.SourceRuntime(t.Context(), f.reference)
			if err != nil {
				t.Fatal(err)
			}
			evidence := &journal{}
			result, err := f.client.PublishSource(t.Context(), runtime, oci.SourceLayer{Reference: execution.Reference{ID: "source-run", Installation: "preparation-studio", Project: "parcel", Run: "source-run", Test: "catalog", Attempt: 1, Worker: "preparer"}, SourceDigest: hash([]byte("new-source")), DiffID: hash([]byte("source-layer")), Compressed: []byte("source-layer")}, evidence)
			if err != nil || result.Project != "parcel" || result.Source != hash([]byte("new-source")) || !strings.HasPrefix(result.Manifest, f.registry+"/teststudio/") {
				t.Fatal("source publication lost normal approval metadata", result, err)
			}
			if evidence.opening != 2 || evidence.opened != 2 || evidence.completed != 2 || f.writes != 5 || f.grants != 2 || f.exchanges != 1 {
				t.Fatal("publication lost scoped grants or upload checkpoints", *evidence, f.writes, f.grants, f.exchanges)
			}
		})
	}
	read := fresh(t, "go", false, true)
	runtime, err := read.client.SourceRuntime(t.Context(), read.reference)
	if err != nil {
		t.Fatal(err)
	}
	_, err = read.client.PublishSource(t.Context(), runtime, oci.SourceLayer{Reference: execution.Reference{ID: "source-run", Installation: "preparation-studio", Project: "parcel", Run: "source-run"}, SourceDigest: hash([]byte("new-source")), DiffID: hash([]byte("source-layer")), Compressed: []byte("source-layer")}, &journal{})
	if err == nil || read.writes != 0 {
		t.Fatal("inspection client performed source publication")
	}
}
