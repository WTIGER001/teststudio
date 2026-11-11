package suitespec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/definitions"
)

const DiscoveryLimit = 8 << 20

type ListedTest struct {
	TitlePath []string `json:"titlePath,omitempty"`
	Title     string   `json:"title,omitempty"`
	Tags      []string `json:"tags"`
	File      string   `json:"file"`
	Line      int      `json:"line"`
	Project   string   `json:"project"`
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
	if wire.Manifest.Discover {
		var err error
		document, err = DiscoverCatalog(wire.Manifest, document, wire.Tests)
		if err != nil {
			return discovery, err
		}
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
			if test.Framework != nil && actual.File == test.Source && actual.Project == test.Framework.Project && reflect.DeepEqual(actual.TitlePath, test.Framework.TitlePath) || test.Framework == nil && slices.Contains(actual.Tags, test.Selector) {
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

func DiscoverCatalog(manifest Manifest, seed catalog.Document, actual []ListedTest) (catalog.Document, error) {
	result := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{}}
	seen := map[string]bool{}
	matched := map[string]bool{}
	for _, entry := range actual {
		if !RelativePath(entry.File, false) || entry.Line < 1 || len(entry.TitlePath) < 1 || len(entry.TitlePath) > 40 || entry.Title == "" || len(entry.Title) > 200 || manifest.Tests.Project != "" && entry.Project != manifest.Tests.Project {
			return result, errors.New("discovery requires exact source, project, and bounded title paths")
		}
		for _, title := range entry.TitlePath {
			if len(title) < 1 || len(title) > 200 || strings.ContainsAny(title, "\r\n›>\x00") {
				return result, errors.New("test titles require a distinct single-line mapping")
			}
		}
		keyBytes, _ := json.Marshal([]any{manifest.Suite, entry.Project, entry.File, entry.TitlePath})
		sum := sha256.Sum256(keyBytes)
		key := hex.EncodeToString(sum[:])
		if seen[key] {
			return result, errors.New("discovery contains ambiguous test identities")
		}
		seen[key] = true
		test := catalog.Test{ID: "TEST-" + key[:24], Title: entry.Title, Category: "Imported", Runner: "playwright", Type: manifest.Type, State: "disabled", Safety: "needs-review", Rationale: "Review imported source, helpers, setup, and side effects before enabling this test.", TimeoutSeconds: 30, Source: entry.File, Selector: "@TEST-" + key[:24]}
		if len(manifest.Applications) == 1 {
			test.Application = manifest.Applications[0]
		}
		matchCount := 0
		for _, prior := range seed.Tests {
			if prior.Framework != nil && prior.Source == entry.File && prior.Framework.Project == entry.Project && reflect.DeepEqual(prior.Framework.TitlePath, entry.TitlePath) || prior.Framework == nil && slices.Contains(entry.Tags, prior.Selector) {
				if matched[prior.ID] {
					return result, errors.New("a permanent identity matches multiple discovered tests")
				}
				matchCount++
				if matchCount > 1 {
					return result, errors.New("multiple permanent identities match the same framework test")
				}
				test = prior
				matched[prior.ID] = true
			}
		}
		test.Framework = &catalog.FrameworkTest{Project: entry.Project, TitlePath: entry.TitlePath, Line: entry.Line}
		result.Tests = append(result.Tests, test)
	}
	for _, prior := range seed.Tests {
		if !matched[prior.ID] {
			return result, fmt.Errorf("test %s no longer matches source; review its rename or removal", prior.ID)
		}
	}
	if len(result.Tests) == 0 {
		return result, errors.New("no framework tests were discovered")
	}
	return result, nil
}
