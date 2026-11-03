package runner

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/wtiger001/teststudio/internal/gosuite"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func loadCatalog(root string) (string, gosuite.Manifest, error) {
	if _, err := os.Stat(filepath.Join(root, suitespec.ManifestFile)); !os.IsNotExist(err) {
		if err != nil {
			return root, gosuite.Manifest{}, err
		}
		discovery, err := suitespec.Load(root)
		if err != nil {
			return root, gosuite.Manifest{}, err
		}
		if discovery.Manifest.Runner != "go" {
			return root, gosuite.Manifest{}, errors.New("prepared suite declares another runner")
		}
		module, err := suitespec.File(root, discovery.Manifest.ModuleRoot)
		if err != nil {
			return root, gosuite.Manifest{}, err
		}
		manifest := gosuite.Manifest{Schema: "teststudio.go.catalog/v1", Tests: []gosuite.Entry{}}
		for _, test := range discovery.Catalog.Tests {
			entry, err := gosuite.Parse(test.Selector)
			if err != nil {
				return root, manifest, err
			}
			entry.ID = test.ID
			manifest.Tests = append(manifest.Tests, entry)
		}
		return module, manifest, gosuite.Validate(manifest.Tests)
	}
	file, err := os.Open(filepath.Join(root, "teststudio-go.json"))
	if err != nil {
		return root, gosuite.Manifest{}, errors.New("Go catalog manifest is unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (512<<10)+1))
	if err != nil || len(data) > 512<<10 {
		return root, gosuite.Manifest{}, errors.New("Go catalog manifest is unavailable or exceeds its byte limit")
	}
	var manifest gosuite.Manifest
	if json.Unmarshal(data, &manifest) != nil || manifest.Schema != "teststudio.go.catalog/v1" {
		return root, manifest, errors.New("Go catalog manifest is invalid")
	}
	return root, manifest, gosuite.Validate(manifest.Tests)
}
