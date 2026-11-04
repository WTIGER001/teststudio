package catalog

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

func publicationDiff(previous, next Draft) ([]PublicationChange, int) {
	before, after := map[string]Test{}, map[string]Test{}
	for _, test := range previous.Tests {
		before[test.ID] = test
	}
	for _, test := range next.Tests {
		after[test.ID] = test
	}
	changes, unchanged := []PublicationChange{}, 0
	for _, test := range next.Tests {
		current := test
		old, exists := before[test.ID]
		if !exists {
			changes = append(changes, PublicationChange{ID: test.ID, Title: test.Title, Action: "added", Fields: []string{}, After: &current})
			continue
		}
		oldJSON, _ := json.Marshal(old)
		newJSON, _ := json.Marshal(test)
		var oldFields, newFields map[string]any
		_ = json.Unmarshal(oldJSON, &oldFields)
		_ = json.Unmarshal(newJSON, &newFields)
		keys := map[string]bool{}
		for key := range oldFields {
			keys[key] = true
		}
		for key := range newFields {
			keys[key] = true
		}
		fields := []string{}
		for key := range keys {
			if !reflect.DeepEqual(oldFields[key], newFields[key]) {
				fields = append(fields, key)
			}
		}
		if len(fields) == 0 {
			unchanged++
			continue
		}
		slices.Sort(fields)
		changes = append(changes, PublicationChange{ID: test.ID, Title: test.Title, Action: "changed", Fields: fields, Before: &old, After: &current, SafetyChanged: old.Safety != test.Safety || old.Rationale != test.Rationale || old.Conditions != test.Conditions})
	}
	for _, test := range previous.Tests {
		if _, exists := after[test.ID]; !exists {
			current := test
			changes = append(changes, PublicationChange{ID: test.ID, Title: test.Title, Action: "removed", Fields: []string{}, Before: &current})
		}
	}
	slices.SortFunc(changes, func(a, b PublicationChange) int { return strings.Compare(a.ID, b.ID) })
	return changes, unchanged
}
