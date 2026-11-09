package sources_test

import (
	"encoding/json"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"testing"
)

func savedBrowserSource(t *testing.T, id, testID string) sources.PlaywrightDraft {
	t.Helper()
	value := sources.StarterDraft()
	value.ID = id
	value.SaveAsTest = true
	var document catalog.Document
	if err := json.Unmarshal([]byte(value.Files[1].Content), &document); err != nil {
		t.Fatal(err)
	}
	document.Tests[0].ID = testID
	document.Tests[0].Selector = "@" + testID
	data, _ := json.Marshal(document)
	value.Files[1].Content = string(data)
	return value
}
func TestSavingBrowserSourceImmediatelyRegistersTestsAndRejectsCollisions(t *testing.T) {
	s, _, _ := bundleSetup(t)
	first := savedBrowserSource(t, "saved-one", "SAVED-001")
	if _, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", first, 0); err != nil {
		t.Fatal(err)
	}
	row, err := s.Store.Get(t.Context(), catalog.Bucket("tests", "alpha", "health"), "SAVED-001")
	if err != nil {
		t.Fatal("saved source is missing from the test catalog", err)
	}
	test, _ := storage.Decode[catalog.Test](row)
	if test.State != "enabled" || test.Source != "checks/saved/saved-one/checks/welcome.spec.ts" {
		t.Fatalf("wrong saved test: %+v", test)
	}
	second := savedBrowserSource(t, "saved-two", "SAVED-002")
	if _, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", second, 0); err != nil {
		t.Fatal(err)
	}
	rows, _ := s.Store.All(t.Context(), catalog.Bucket("tests", "alpha", "health"), "alpha")
	if len(rows) != 2 {
		t.Fatal("saving a second test replaced an existing one")
	}
	collision := savedBrowserSource(t, "collision", "SAVED-001")
	if _, err := s.SaveDraft(t.Context(), "admin", "alpha", "health", collision, 0); err == nil {
		t.Fatal("another source replaced the saved test")
	}
	if _, err := s.Draft(t.Context(), "admin", "alpha", "health", "collision"); err == nil {
		t.Fatal("failed save left a partial source")
	}
}
