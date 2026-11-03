package suitespec

import (
	"errors"
	"path"
	"slices"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/definitions"
)

const DiscoveryLimit = 8 << 20

type ListedTest struct {
	Tags    []string `json:"tags"`
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Project string   `json:"project"`
}

type PlaywrightDiscovery struct {
	Schema      string       `json:"schemaVersion"`
	Manifest    Manifest     `json:"manifest"`
	Origin      Origin       `json:"origin"`
	CatalogText string       `json:"catalogText"`
	Tests       []ListedTest `json:"tests"`
}

func DecodeDiscovery(data []byte, runner string) (Discovery, error) {
	var discovery Discovery
	if len(data) == 0 || len(data) > DiscoveryLimit {
		return discovery, errors.New("discovery report exceeds its byte limit")
	}
	if runner == "go" {
		if err := decodeReport(data, &discovery); err != nil {
			return discovery, err
		}
		return discovery, nil
	}
	if runner != "playwright" {
		return discovery, errors.New("unsupported discovery runner")
	}
	var wire PlaywrightDiscovery
	if err := decodeReport(data, &wire); err != nil {
		return discovery, err
	}
	if wire.Schema != DiscoverySchema || wire.Manifest.Runner != "playwright" || len(wire.Tests) > 10000 {
		return discovery, errors.New("Playwright discovery report is invalid or oversized")
	}
	format := "json"
	if extension := path.Ext(wire.Manifest.Catalog); extension == ".yaml" || extension == ".yml" {
		format = "yaml"
	}
	var document catalog.Document
	if err := definitions.Decode(format, wire.CatalogText, &document); err != nil {
		return discovery, err
	}
	if err := ValidateCatalog(wire.Manifest, document); err != nil {
		return discovery, err
	}
	discovery = Discovery{Schema: wire.Schema, Manifest: wire.Manifest, Origin: wire.Origin, Catalog: document, Tests: []DiscoveredTest{}}
	for _, test := range document.Tests {
		matches := []ListedTest{}
		for _, actual := range wire.Tests {
			if !RelativePath(actual.File, false) || actual.Line < 1 {
				return discovery, errors.New("Playwright discovery source is outside the suite context")
			}
			if wire.Manifest.Tests.Project != "" && actual.Project != wire.Manifest.Tests.Project {
				return discovery, errors.New("discovery includes another Playwright project")
			}
			if slices.Contains(actual.Tags, test.Selector) {
				matches = append(matches, actual)
			}
		}
		if len(matches) != 1 {
			return discovery, errors.New("each permanent Playwright tag must select exactly one discovered test")
		}
		discovery.Tests = append(discovery.Tests, DiscoveredTest{ID: test.ID, Selector: test.Selector, File: matches[0].File, Line: matches[0].Line})
	}
	return discovery, nil
}

func decodeReport(data []byte, value any) error {
	return definitions.DecodeJSON(data, value, DiscoveryLimit, 40, 200000)
}
