package embeddedplaywright

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/recording"
)

func TestEmbeddedRecordingCapturesNewsAssertionAndClosesBrowser(t *testing.T) {
	if os.Getenv("TESTSTUDIO_EMBEDDED_RECORDING") != "1" {
		t.Skip("enable the focused embedded recording check with the installed trusted browser")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	driver, err := New(t.Context(), node, filepath.Join(root, "runners/playwright/node_modules/playwright/cli.js"), filepath.Join(root, "scripts/record-browser-embedded.mjs"), filepath.Join(t.TempDir(), "recordings"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := driver.Close(); err != nil {
			t.Error(err)
		}
	})
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><style>body{margin:0}a{display:block;padding:15px}h2{margin:20px}</style><nav><a href="/news">News</a></nav>`))
		if r.URL.Path == "/news" {
			_, _ = w.Write([]byte(`<main><article><h2>Regional harvest begins</h2><p>A new independent news fixture.</p></article><article><h2>Second independent story</h2></article></main>`))
		}
	}))
	t.Cleanup(target.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	if err := driver.Start(ctx, "news", target.URL); err != nil {
		t.Fatal(err)
	}
	view, err := driver.Interact(ctx, "news", recording.Action{Sequence: 1, Action: "click", X: 35, Y: 25})
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); !strings.HasSuffix(view.URL, "/news") && time.Now().Before(deadline); {
		view, err = driver.View(ctx, "news")
		if err != nil {
			t.Fatal(err)
		}
	}
	if !strings.HasSuffix(view.URL, "/news") || len(view.Image) == 0 {
		t.Fatal("news navigation did not appear in the live view")
	}
	if _, err := driver.Interact(ctx, "news", recording.Action{Sequence: 2, Action: "assert-visible"}); err != nil {
		t.Fatal(err)
	}
	view, err = driver.Interact(ctx, "news", recording.Action{Sequence: 3, Action: "click", X: 120, Y: 85})
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		snapshot, err := driver.Read(ctx, "news")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(snapshot.Code, "toBeVisible()") {
			break
		}
		view, err = driver.View(ctx, "news")
		if err != nil {
			t.Fatal(err)
		}
	}
	before := len(view.Steps)
	view, err = driver.Interact(ctx, "news", recording.Action{Sequence: 3, Action: "click", X: 120, Y: 85})
	if err != nil || len(view.Steps) != before {
		t.Fatal("exact retry repeated an action", err)
	}
	if _, err := driver.Interact(ctx, "news", recording.Action{Sequence: 3, Action: "click", X: 35, Y: 25}); err == nil {
		t.Fatal("changed duplicate action was accepted")
	}

	actions := []recording.Action{
		{Action: "resize", Width: 390, Height: 844},
		{Action: "assert-pick"}, {Action: "click", X: 120, Y: 85},
		{Action: "assert-add", Key: "at-least", Count: 3},
		{Action: "assert-add", Key: "at-least", Count: 1},
		{Action: "assert-pick"}, {Action: "click", X: 120, Y: 85},
		{Action: "assert-add", Key: "exactly", Count: 2},
		{Action: "assert-pick"}, {Action: "click", X: 120, Y: 85},
		{Action: "assert-add", Key: "text", Text: "Regional harvest"},
		{Action: "assert-url"},
	}
	for index, action := range actions {
		action.Sequence = index + 4
		view, err = driver.Interact(ctx, "news", action)
		if err != nil {
			t.Fatalf("action %s: %v", action.Action, err)
		}
		if view.Width != 390 || view.Height != 844 {
			t.Fatal("actual viewport dimensions were not retained")
		}
		if action.Action == "click" && (view.Target == nil || view.Target.Matches != 2) {
			t.Fatalf("article grouping failed: %+v", view.Target)
		}
		if action.Count == 3 && view.Message == "" {
			t.Fatal("mismatched count was not explained")
		}
	}

	snapshot, err := driver.Stop(ctx, "news")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != "ready" || !strings.Contains(snapshot.Code, "name: 'News'") || !strings.Contains(snapshot.Code, "toBeVisible()") || len(snapshot.Steps) < 3 {
		t.Fatalf("recorded outcome state=%s steps=%+v code=%s", snapshot.State, snapshot.Steps, snapshot.Code)
	}

	for _, expected := range []string{"setViewportSize({ width: 390, height: 844 })", "toBeGreaterThanOrEqual(1)", "toHaveCount(2)", "toContainText(\"Regional harvest\")", "toHaveURL("} {
		if !strings.Contains(snapshot.Code, expected) {
			t.Fatalf("missing generated assertion %s: %s", expected, snapshot.Code)
		}
	}
	if strings.Contains(snapshot.Code, "teststudio_") {
		t.Fatal("internal extension marker escaped into source")
	}

	item, _ := driver.item("news")
	select {
	case <-item.done:
	default:
		t.Fatal("owned browser process did not stop")
	}
}
