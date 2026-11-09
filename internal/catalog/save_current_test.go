package catalog_test

import (
	"errors"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func TestSaveCurrentTestMakesChangesUsableAndPreservesHistory(t *testing.T) {
	s, _ := setup(t)
	test := check("SAVE-001")
	if err := s.SaveCurrentTest(t.Context(), "admin", "alpha", "health", test, 0); err != nil {
		t.Fatal(err)
	}
	suiteRow, err := s.Store.Get(t.Context(), projects.Bucket("suites", "alpha"), "health")
	if err != nil {
		t.Fatal(err)
	}
	suite, _ := storage.Decode[projects.Suite](suiteRow)
	if suite.ActiveRelease == "" {
		t.Fatal("saving required a second activation step")
	}
	previous := suite.ActiveRelease
	confirmation, err := s.ReadSavedTest(t.Context(), "admin", "alpha", "health", test.ID)
	if err != nil || confirmation.SavedVersion != previous {
		t.Fatal("save confirmation did not retain its exact executable version", err)
	}
	test.Title = "Updated saved check"
	if err := s.SaveCurrentTest(t.Context(), "admin", "alpha", "health", test, 1); err != nil {
		t.Fatal(err)
	}
	suiteRow, _ = s.Store.Get(t.Context(), suiteRow.Bucket, suiteRow.ID)
	suite, _ = storage.Decode[projects.Suite](suiteRow)
	if suite.ActiveRelease == previous {
		t.Fatal("save did not use the current definition")
	}
	old, _ := s.Store.Get(t.Context(), catalog.Bucket("releases", "alpha", "health"), previous)
	retained, _ := storage.Decode[catalog.Release](old)
	if retained.Draft.Tests[0].Title == test.Title {
		t.Fatal("a save rewrote historical execution data")
	}
	currentVersion := suite.ActiveRelease
	if err := s.UseCurrent(t.Context(), "admin", "alpha", "health"); err != nil {
		t.Fatal(err)
	}
	suiteRow, _ = s.Store.Get(t.Context(), suiteRow.Bucket, suiteRow.ID)
	suite, _ = storage.Decode[projects.Suite](suiteRow)
	if suite.ActiveRelease != currentVersion {
		t.Fatal("retry created another version without a definition change")
	}
	if err := s.SaveCurrentTest(t.Context(), "admin", "alpha", "health", test, 1); !errors.Is(err, storage.ErrConflict) {
		t.Fatal("stale save did not conflict", err)
	}
}
