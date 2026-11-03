package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

func validateBulkMetadata(patch *BulkMetadata) error {
	if patch == nil || (patch.Title == nil && patch.Description == nil && patch.Category == nil && patch.Tags == nil && patch.Owner == nil && patch.Safety == nil && patch.Rationale == nil && patch.Conditions == nil) {
		return errors.New("metadata requires at least one explicitly selected field")
	}
	for _, field := range []struct {
		name     string
		value    *string
		maximum  int
		required bool
	}{
		{"title", patch.Title, 200, true}, {"description", patch.Description, 4096, false},
		{"category", patch.Category, 200, true}, {"owner", patch.Owner, 200, false},
		{"rationale", patch.Rationale, 4096, true}, {"conditions", patch.Conditions, 4096, false},
	} {
		if field.value != nil && (len(*field.value) > field.maximum || field.required && strings.TrimSpace(*field.value) == "") {
			return fmt.Errorf("metadata.%s requires %s at most %d bytes", field.name, map[bool]string{true: "nonempty text of", false: "text of"}[field.required], field.maximum)
		}
	}
	if patch.Safety != nil {
		if !slices.Contains([]string{"safe", "unsafe", "needs-review"}, *patch.Safety) {
			return errors.New("metadata.safety requires safe, unsafe, or needs-review")
		}
		if patch.Rationale == nil || strings.TrimSpace(*patch.Rationale) == "" {
			return errors.New("metadata.rationale is required when selecting a safety label")
		}
	}
	if patch.Tags != nil {
		if len(*patch.Tags) > 30 {
			return errors.New("metadata.tags accepts at most 30 tags")
		}
		seen := map[string]bool{}
		for _, tag := range *patch.Tags {
			if tag == "" || tag != strings.TrimSpace(tag) || len(tag) > 64 || seen[tag] {
				return errors.New("metadata.tags requires distinct, trimmed tags of 1 through 64 bytes")
			}
			seen[tag] = true
		}
	}
	return nil
}

func patchBulkMetadata(test Test, patch BulkMetadata) Test {
	for _, field := range []struct{ input, destination *string }{
		{patch.Title, &test.Title}, {patch.Description, &test.Description}, {patch.Category, &test.Category},
		{patch.Owner, &test.Owner}, {patch.Safety, &test.Safety}, {patch.Rationale, &test.Rationale}, {patch.Conditions, &test.Conditions},
	} {
		if field.input != nil {
			*field.destination = *field.input
		}
	}
	if patch.Tags != nil {
		test.Tags = append([]string{}, (*patch.Tags)...)
	}
	return test
}
