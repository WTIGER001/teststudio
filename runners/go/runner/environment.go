package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func validateEnvironment(root string, input Input) error {
	var manifest suitespec.Manifest
	if _, err := os.Stat(filepath.Join(root, suitespec.ManifestFile)); os.IsNotExist(err) {
		if input.Browser != nil {
			return errors.New("Go API execution requires its prepared environment manifest")
		}
		return nil
	}
	if err := suitespec.Read(root, suitespec.ManifestFile, &manifest); err != nil {
		return err
	}
	if (manifest.Browser != nil) != (input.Browser != nil) {
		return errors.New("Go execution target does not match the prepared source profile")
	}
	if manifest.Browser == nil {
		return nil
	}
	if manifest.BuildProfile != "go-environment" || manifest.Type != "api" || input.Test.Type != "api" || input.Browser.Requests == nil {
		return errors.New("Go environment execution requires a request-restricted API profile")
	}
	if err := browserpolicy.ValidateTarget(*input.Browser); err != nil {
		return err
	}
	prepared, _ := json.Marshal(manifest.Browser.Requests)
	frozen, _ := json.Marshal(input.Browser.Requests)
	if !bytes.Equal(prepared, frozen) {
		return errors.New("Go API request policy does not match its prepared source")
	}
	var document catalog.Document
	if err := suitespec.Read(root, manifest.Catalog, &document); err != nil {
		return err
	}
	for _, test := range document.Tests {
		if test.ID == input.Test.ID && test.Selector == input.Test.Selector {
			if !slices.Equal(test.RequiredBindings, input.Test.RequiredBindings) || input.Browser.Authenticated != (len(test.RequiredBindings) > 0) {
				return errors.New("Go API authentication does not match the prepared test")
			}
			return apipolicy.ValidateBindings(manifest.Browser.Requests, test.RequiredBindings)
		}
	}
	return errors.New("Go API test is absent from the prepared catalog")
}

func requestEnvironment(base []string, address string) []string {
	return append(slices.Clone(base), "TESTSTUDIO_BASE_URL="+address, "TESTSTUDIO_API_BASE_URL="+address, "HTTP_PROXY=", "HTTPS_PROXY=", "ALL_PROXY=", "http_proxy=", "https_proxy=", "all_proxy=", "NO_PROXY=", "no_proxy=")
}
