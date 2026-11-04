package catalog

import (
	"errors"
	"slices"
	"strings"
	"unicode"
)

func normalizeListQuery(q ListQuery) (ListQuery, error) {
	q.Query = strings.ToLower(strings.TrimSpace(q.Query))
	q.Category = selectionCategory(q.Category)
	if q.State == "" {
		q.State = "all"
	}
	if q.Sort == "" {
		q.Sort = "title"
	}
	if q.Limit == 0 {
		q.Limit = 30
	}
	if q.Limit < 1 || q.Limit > 100 || !slices.Contains([]string{"all", "active", "draft", "enabled", "disabled", "archived"}, q.State) || !slices.Contains([]string{"title", "title-desc", "id", "category", "updated"}, q.Sort) {
		return q, errors.New("catalog limit must be 1 through 100 with a supported lifecycle state and sort")
	}
	for _, value := range []string{q.Query, q.Suite, q.Application, q.Category, q.Tag, q.Owner, q.Type, q.Runner, q.Safety, q.Source} {
		if len(value) > 200 || strings.ContainsFunc(value, unicode.IsControl) {
			return q, errors.New("catalog filter values accept at most 200 bytes without control characters")
		}
	}
	if q.Uncategorized && q.Category != "" {
		return q, errors.New("choose a category path or uncategorized tests")
	}
	return q, nil
}

func matchesList(test Test, suiteName string, q ListQuery, category bool) bool {
	if q.State != "all" && (q.State == "active" && test.State == "archived" || q.State != "active" && test.State != q.State) {
		return false
	}
	for _, pair := range [][2]string{{q.Application, test.Application}, {q.Owner, test.Owner}, {q.Type, test.Type}, {q.Runner, test.Runner}, {q.Safety, test.Safety}} {
		if pair[0] != "" && pair[0] != pair[1] {
			return false
		}
	}
	if q.Tag != "" && !slices.Contains(test.Tags, q.Tag) {
		return false
	}
	if q.Source != "" && q.Source != test.Source && q.Source != test.DefinitionSource {
		return false
	}
	if category {
		actual := selectionCategory(test.Category)
		if q.Uncategorized && actual != "" {
			return false
		}
		if q.Category != "" && actual != q.Category && (!q.Descendants || !strings.HasPrefix(actual, q.Category+"/")) {
			return false
		}
	}
	text := strings.ToLower(strings.Join(append([]string{test.ID, test.Title, test.Description, test.Category, suiteName}, test.Tags...), " "))
	return strings.Contains(text, q.Query)
}

func compareList(a, b listItem, sort string) int {
	result := 0
	switch sort {
	case "title-desc":
		result = strings.Compare(strings.ToLower(b.Test.Title), strings.ToLower(a.Test.Title))
	case "id":
		result = strings.Compare(a.Test.ID, b.Test.ID)
	case "category":
		result = strings.Compare(selectionCategory(a.Test.Category), selectionCategory(b.Test.Category))
	case "updated":
		result = b.Entry.Updated.Compare(a.Entry.Updated)
	default:
		result = strings.Compare(strings.ToLower(a.Test.Title), strings.ToLower(b.Test.Title))
	}
	if result != 0 {
		return result
	}
	if result = strings.Compare(a.Test.Title, b.Test.Title); result != 0 {
		return result
	}
	if result = strings.Compare(a.Entry.Suite, b.Entry.Suite); result != 0 {
		return result
	}
	return strings.Compare(a.Test.ID, b.Test.ID)
}

func categoryNodes(items []listItem) []*CategoryNode {
	roots := []*CategoryNode{}
	for _, item := range items {
		value := selectionCategory(item.Test.Category)
		parts := strings.Split(value, "/")
		for _, part := range parts {
			if part == "" {
				parts = []string{value}
				break
			}
		}
		nodes := &roots
		path := ""
		for _, part := range parts {
			if path != "" {
				path += "/"
			}
			path += part
			key, label := path, part
			if value == "" {
				key, label = UncategorizedCategory, "Uncategorized"
			}
			var current *CategoryNode
			for _, node := range *nodes {
				if node.Path == key {
					current = node
					break
				}
			}
			if current == nil {
				current = &CategoryNode{Path: key, Label: label, Children: []*CategoryNode{}}
				*nodes = append(*nodes, current)
			}
			current.Count++
			nodes = &current.Children
		}
	}
	var order func([]*CategoryNode)
	order = func(nodes []*CategoryNode) {
		slices.SortFunc(nodes, func(a, b *CategoryNode) int { return strings.Compare(a.Label, b.Label) })
		for _, node := range nodes {
			order(node.Children)
		}
	}
	order(roots)
	return roots
}

func listOptions(items []listItem) (ListOptions, CatalogCounts) {
	options := ListOptions{Application: []string{}, Tag: []string{}, Owner: []string{}, Runner: []string{}, Safety: []string{}, Source: []string{}, Type: []string{}}
	counts := CatalogCounts{}
	suites := map[string]bool{}
	add := func(values *[]string, value string) {
		if value != "" && !slices.Contains(*values, value) {
			*values = append(*values, value)
		}
	}
	for _, item := range items {
		test := item.Test
		if test.State != "archived" {
			counts.Total++
			suites[item.Entry.Suite] = true
			if test.State == "enabled" {
				counts.Enabled++
			}
			if test.Safety == "safe" {
				counts.Safe++
			}
		}
		add(&options.Application, test.Application)
		add(&options.Owner, test.Owner)
		add(&options.Runner, test.Runner)
		add(&options.Safety, test.Safety)
		add(&options.Type, test.Type)
		add(&options.Source, test.Source)
		add(&options.Source, test.DefinitionSource)
		for _, tag := range test.Tags {
			add(&options.Tag, tag)
		}
	}
	for _, values := range []*[]string{&options.Application, &options.Tag, &options.Owner, &options.Runner, &options.Safety, &options.Source, &options.Type} {
		slices.Sort(*values)
	}
	counts.Suites = len(suites)
	return options, counts
}
