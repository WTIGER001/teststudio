package registryimage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryAssemblyHandoffReleasesJournalAndPreservesBothPublishedRunners(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f := fixture(t, runner, nil)
			input := buildInput(t, runner)
			builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
			result, err := builder.Build(t.Context(), input)
			if err != nil {
				t.Fatal(err)
			}
			f.mu.Lock()
			posts, manifests := f.posts, f.manifests
			f.mu.Unlock()
			if err := builder.CompleteAssembly(t.Context(), input.Reference); err != nil {
				t.Fatal(err)
			}
			if err := builder.CompleteAssembly(t.Context(), input.Reference); err != nil {
				t.Fatal("assembly handoff is not idempotent", err)
			}
			if _, err := os.Stat(filepath.Join(builder.Directory, journalName(input.Reference))); !os.IsNotExist(err) {
				t.Fatal("handoff retained the upload journal", err)
			}
			actual, err := f.client.Inspect(t.Context(), result.Image)
			f.mu.Lock()
			unchanged := f.posts == posts && f.manifests == manifests
			f.mu.Unlock()
			if err != nil || actual.Project != input.Reference.Project || actual.Preparation != input.Reference.ID || actual.Source != input.Snapshot.Digest || actual.Image.Reference != result.Image || !unchanged {
				t.Fatal("handoff deleted or rewrote its immutable source artifact", err)
			}
		})
	}
}
