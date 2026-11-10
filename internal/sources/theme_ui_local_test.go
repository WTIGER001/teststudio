package sources_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/adapters/execution/localcontainer"
	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestButtonThemesKeepFirstFrameContrastThroughStudioExecution(t *testing.T) {
	runtime := os.Getenv("TESTSTUDIO_BROWSER_RUNTIME_IMAGE")
	if runtime == "" {
		t.Skip("supply the cached immutable local Playwright source runtime")
	}
	root := os.Getenv("TESTSTUDIO_THEME_UI_OUTPUT")
	if root == "" {
		root = t.TempDir()
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	s, c, _ := bundleSetup(t)
	s.BundleRunners = []string{"playwright"}
	s.Builders = map[string]preparation.ArtifactBuilder{"playwright": &localimage.Builder{RuntimeImage: runtime, TempDirectory: filepath.Join(root, "source-contexts")}}
	s.RecoveryBuilders = s.Builders
	s.Inspectors = map[string]preparation.ArtifactInspector{"playwright": &localcontainer.Executor{Runner: "playwright", AllowProjectImages: true, RequireProject: true}}
	s.RecoveryInspectors = s.Inspectors
	t.Cleanup(func() { cleanupRepositoryUIJobs(t, s, root) })
	projectService := &projects.Service{Store: s.Store, Auth: s.Auth}
	if err := projectService.SaveSuite(t.Context(), "admin", "alpha", projects.Suite{ID: "ui", Name: "Button theme rendering", Applications: []string{"app"}, RequiresTarget: false}, 0); err != nil {
		t.Fatal(err)
	}
	specification, err := os.ReadFile("../../tests/authoring/button-themes.spec.ts")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := os.ReadFile("../../web/src/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	draft := sources.StarterDraft()
	draft.ID, draft.Name = "button-theme", "First-frame button theme contrast"
	draft.Files[0] = sources.EditorFile{Path: "checks/button-themes.spec.ts", Content: string(specification)}
	draft.Files = append(draft.Files, sources.EditorFile{Path: "theme.css", Content: string(styles)})
	var document catalog.Document
	if err := json.Unmarshal([]byte(draft.Files[1].Content), &document); err != nil {
		t.Fatal(err)
	}
	document.Tests[0].ID, document.Tests[0].Title, document.Tests[0].Selector = "STUDIO-COLOR-001", "First-frame button theme contrast", "@STUDIO-COLOR-001"
	document.Tests[0].Source = draft.Files[0].Path
	metadata, _ := json.Marshal(document)
	draft.Files[1].Content = string(metadata)
	saved, err := s.SaveDraft(t.Context(), "admin", "alpha", "ui", draft, 0)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := s.PrepareDraft(t.Context(), "admin", "alpha", "ui", draft.ID, "theme-rendering", saved.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := s.ProcessBuildNext(t.Context(), "theme-rendering-builder"); err != nil || !processed {
		t.Fatal(processed, err)
	}
	row, err := s.ReadJob(t.Context(), "admin", "alpha", "ui", prepared.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, _ := storage.Decode[sources.Job](row)
	if job.State != "ready" {
		t.Fatalf("theme preparation state=%s error=%s", job.State, job.Error)
	}
	if _, err := s.Publish(t.Context(), "admin", "alpha", "ui", job.Upload, 1, true); err != nil {
		t.Fatal(err)
	}
	blobs, err := filesystem.New(filepath.Join(root, "artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { blobs.Close() })
	runner := &runs.Service{Store: s.Store, Auth: s.Auth, Catalog: c, Evidence: &evidence.Service{Store: s.Store, Blobs: blobs}, Location: "local", PollInterval: 20 * time.Millisecond, Playwright: &localcontainer.Executor{Runner: "playwright", ApproveArtifact: s.ApproveArtifact}}
	ctx, cancel := context.WithCancel(t.Context())
	done := runner.Start(ctx)
	t.Cleanup(func() { cancel(); <-done })
	run, err := runner.Create(t.Context(), "admin", "alpha", runs.Request{Suite: "ui", TimeoutSeconds: 45}, "theme-rendering-run", "manual")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(55 * time.Second)
	for time.Now().Before(deadline) {
		run, _, err = runner.Get(t.Context(), "alpha", run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if runs.Terminal(run.State) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	data, _ := json.MarshalIndent(map[string]any{"run": run}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "summary.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if run.State != "completed" || run.Outcome != "passed" || run.Counts.Passed != 1 || run.EvidenceState != "complete" {
		t.Fatalf("theme execution state=%s outcome=%s results=%+v", run.State, run.Outcome, run.Results)
	}
	t.Logf("Theme rendering run %s passes immediate dark and light contrast with retained screenshots and keyboard focus", run.ID)
}
