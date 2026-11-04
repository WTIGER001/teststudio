package sources

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
)

var recordedTestDeclaration = regexp.MustCompile(`(?m)^test\((?:'test'|"test"),\s*async\s*\(\{\s*page\s*\}\)\s*=>\s*\{`)

func RecordedDraft(value recording.Session) (PlaywrightDraft, error) {
	if value.State != "ready" || !domain.ValidID(value.TestID) || !recording.ValidURL(value.URL) || len(value.Code) > recording.ContentLimit {
		return PlaywrightDraft{}, errors.New("finish the browser recording before opening its generated source")
	}
	matches := recordedTestDeclaration.FindAllStringIndex(value.Code, 2)
	if len(matches) != 1 {
		return PlaywrightDraft{}, errors.New("the recording must contain one generated Playwright test before it can become a catalog draft; review the retained code")
	}
	title, _ := json.Marshal(value.Title)
	tag, _ := json.Marshal("@" + value.TestID)
	declaration := "test(" + string(title) + ", { tag: " + string(tag) + " }, async ({ page }) => {"
	code := value.Code[:matches[0][0]] + declaration + value.Code[matches[0][1]:]
	code, err := bindRecordedNavigation(code, value.URL)
	if err != nil {
		return PlaywrightDraft{}, err
	}
	draft := StarterDraft()
	draft.ID, draft.Name, draft.TargetURL = domain.NewID(), value.Title, value.URL
	draft.Files[0] = EditorFile{Path: "checks/recording.spec.ts", Content: code}
	var document catalog.Document
	if err := json.Unmarshal([]byte(draft.Files[1].Content), &document); err != nil {
		return PlaywrightDraft{}, err
	}
	test := &document.Tests[0]
	test.ID, test.Title, test.Selector, test.Source = value.TestID, value.Title, "@"+value.TestID, draft.Files[0].Path
	test.Category, test.Safety, test.Rationale = "Recorded browser flows", "needs-review", "Recorded browser steps require review before use against application data"
	test.TimeoutSeconds, test.Capture.Screenshots = 30, "failure"
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return PlaywrightDraft{}, err
	}
	draft.Files[1].Content = strings.TrimSpace(string(content)) + "\n"
	return draft, validateDraft(draft)
}
