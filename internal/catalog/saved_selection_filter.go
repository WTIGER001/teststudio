package catalog

import (
	"errors"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
)

func validateSavedSelection(value SavedSelection) error {
	if !domain.ValidID(value.ID) || strings.TrimSpace(value.Name) == "" || len(value.Name) > 200 || len(value.Description) > 4096 || value.Spec.Version != 1 {
		return errors.New("saved selection requires a permanent identifier, bounded name and description, and schemaVersion 1")
	}
	switch value.Spec.Kind {
	case "explicit":
		if value.Spec.Filter != nil || len(value.Spec.IDs) < 1 || len(value.Spec.IDs) > 1000 {
			return errors.New("explicit selections require 1 through 1000 identifiers and no filter")
		}
		seen := map[string]bool{}
		for _, id := range value.Spec.IDs {
			if !domain.ValidID(id) || seen[id] {
				return errors.New("explicit selection identifiers must be valid and distinct")
			}
			seen[id] = true
		}
	case "filter":
		if value.Spec.Filter == nil || len(value.Spec.IDs) != 0 {
			return errors.New("filter selections require a versioned filter and no explicit identifiers")
		}
		filter := value.Spec.Filter
		for _, text := range []string{filter.Query, filter.Category, filter.Owner, filter.Application, filter.Type, filter.Runner, filter.Safety, filter.Source} {
			if len(text) > 200 {
				return errors.New("selection filter text accepts at most 200 bytes per field")
			}
		}
		if len(filter.Tags) > 30 {
			return errors.New("selection filters accept at most 30 tags")
		}
		seen := map[string]bool{}
		for _, tag := range filter.Tags {
			if strings.TrimSpace(tag) == "" || len(tag) > 64 || seen[tag] {
				return errors.New("selection filter tags require distinct nonempty values of at most 64 bytes")
			}
			seen[tag] = true
		}
	default:
		return errors.New("saved selection kind requires explicit or filter")
	}
	return nil
}

func matchesSavedFilter(test Test, filter SavedSelectionFilter) bool {
	if test.State != "enabled" {
		return false
	}
	for _, pair := range [][2]string{{filter.Owner, test.Owner}, {filter.Application, test.Application}, {filter.Type, test.Type}, {filter.Runner, test.Runner}, {filter.Safety, test.Safety}} {
		if pair[0] != "" && pair[0] != pair[1] {
			return false
		}
	}
	if filter.Source != "" && filter.Source != test.Source && filter.Source != test.DefinitionSource {
		return false
	}
	if filter.Category != "" {
		category, requested := selectionCategory(test.Category), selectionCategory(filter.Category)
		if category != requested && (!filter.Descendants || !strings.HasPrefix(category, requested+"/")) {
			return false
		}
	}
	for _, tag := range filter.Tags {
		if !slices.Contains(test.Tags, tag) {
			return false
		}
	}
	text := strings.ToLower(strings.Join(append([]string{test.ID, test.Title, test.Description, test.Category}, test.Tags...), " "))
	return strings.Contains(text, strings.ToLower(strings.TrimSpace(filter.Query)))
}

func selectionCategory(value string) string {
	parts := strings.Split(value, "/")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, "/")
}
