package trends

import (
	"cmp"
	"slices"

	"github.com/wtiger001/teststudio/internal/runs"
)

func options(input []runs.Run, f Filter) Options {
	suites, environments := map[string]string{}, map[string]string{}
	categories, locations := map[string]bool{}, map[string]bool{}
	for _, run := range input {
		if !inPeriod(run, f) || !runs.InApplication(run, f.Application) {
			continue
		}
		suites[run.Request.Suite] = run.Request.Suite
		environments["none"] = "No target"
		if run.Environment != nil {
			environments[run.Request.Environment] = run.Environment.Name
		}
		locations[run.Request.Location] = true
		for _, test := range run.Tests {
			if runs.TestInApplication(run, test, f.Application) && test.Category != "" {
				categories[test.Category] = true
			}
		}
	}
	result := Options{Suites: []Option{}, Environments: []Option{}, Categories: []string{}, Locations: []string{}}
	for id, name := range suites {
		result.Suites = append(result.Suites, Option{ID: id, Name: name})
	}
	for id, name := range environments {
		result.Environments = append(result.Environments, Option{ID: id, Name: name})
	}
	for category := range categories {
		result.Categories = append(result.Categories, category)
	}
	for location := range locations {
		result.Locations = append(result.Locations, location)
	}
	slices.SortFunc(result.Suites, func(a, b Option) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(result.Environments, func(a, b Option) int { return cmp.Compare(a.ID, b.ID) })
	slices.Sort(result.Categories)
	slices.Sort(result.Locations)
	result.Partial = len(result.Suites) > 200 || len(result.Environments) > 200 || len(result.Categories) > 200 || len(result.Locations) > 200
	result.Suites = result.Suites[:min(200, len(result.Suites))]
	result.Environments = result.Environments[:min(200, len(result.Environments))]
	result.Categories = result.Categories[:min(200, len(result.Categories))]
	result.Locations = result.Locations[:min(200, len(result.Locations))]
	return result
}
