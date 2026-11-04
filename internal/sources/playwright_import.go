package sources

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
)

func (s *Service) EditorSource(ctx context.Context, user, project, suite, id string) (PlaywrightDraft, error) {
	bundle, err := s.ReadBundle(ctx, user, project, suite, id)
	if err != nil {
		return PlaywrightDraft{}, err
	}
	snapshot, err := sourcebundle.ValidateRoot(bundle.Content, bundle.Manifest, suite, bundle.Root)
	if err != nil {
		return PlaywrightDraft{}, err
	}
	if snapshot.Manifest.Runner != "playwright" || (snapshot.Manifest.BuildProfile != "playwright-node" && snapshot.Manifest.BuildProfile != "playwright-environment") {
		return PlaywrightDraft{}, errors.New("only supported local Playwright bundles can open in the browser editor")
	}
	value := PlaywrightDraft{ID: domain.NewID(), Name: "Imported browser source", ManifestPath: bundle.Manifest, Files: []EditorFile{}}
	size := 0
	for _, path := range snapshot.Paths() {
		if strings.EqualFold(path, "teststudio-origin.json") {
			continue
		}
		content := snapshot.Files[path]
		size += len(content)
		if len(value.Files) >= 50 || len(content) > 256<<10 || size > ContentLimit {
			return PlaywrightDraft{}, errors.New("source exceeds the editor limits of 50 files, 256 KiB per file, or 1 MiB total; keep using the complete ZIP or repository workflow")
		}
		if !utf8.Valid(content) || strings.ContainsRune(string(content), 0) {
			return PlaywrightDraft{}, fmt.Errorf("%s is not an editable UTF-8 text file; keep using the complete ZIP or repository workflow", path)
		}
		value.Files = append(value.Files, EditorFile{Path: path, Content: string(content), Executable: snapshot.Executable[path]})
	}
	if err := validateDraft(value); err != nil {
		return PlaywrightDraft{}, fmt.Errorf("source exceeds the editor workspace limits: %w; keep using the complete ZIP or repository workflow", err)
	}
	first := snapshot.Catalog.Tests[0].Source
	slices.SortStableFunc(value.Files, func(a, b EditorFile) int {
		if a.Path == b.Path {
			return 0
		}
		if a.Path == first {
			return -1
		}
		if b.Path == first {
			return 1
		}
		return strings.Compare(a.Path, b.Path)
	})
	return value, nil
}
