package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/adapters/artifacts/config"
)

func TestArtifactConfigurationSelectsOneBackendWithoutReadingCredentials(t *testing.T) {
	root := t.TempDir()
	for _, fixture := range []struct{ name, document, driver string }{
		{"local", `{"driver":"filesystem","directory":"private"}`, "filesystem"},
		{"s3", `{"driver":"s3","s3":{"region":"us-east-1","bucket":"independent-evidence","prefix":"teststudio/install","expectedOwner":"123456789012"}}`, "s3"},
		{"azure", `{"driver":"azure","azure":{"serviceUrl":"https://independent.blob.core.windows.net","container":"evidence","prefix":"teststudio/install"}}`, "azure"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			file := filepath.Join(root, fixture.name+".json")
			if err := os.WriteFile(file, []byte(fixture.document), 0600); err != nil {
				t.Fatal(err)
			}
			s, err := config.Load(file, "unused-default")
			if err != nil || s.Driver != fixture.driver {
				t.Fatal(s, err)
			}
			if s.Driver == "filesystem" && s.Directory != filepath.Join(root, "private") {
				t.Fatal("relative directory was not resolved beside configuration", s.Directory)
			}
			if s.Driver != "filesystem" && s.Directory != "" {
				t.Fatal("local storage settings leaked into the selected provider")
			}
		})
	}
	defaults, err := config.Load("", filepath.Join(root, "default"))
	if err != nil || defaults.Driver != "filesystem" {
		t.Fatal("local default changed", defaults, err)
	}
	store, close, err := config.Open(t.Context(), defaults)
	if err != nil || store == nil {
		t.Fatal("local storage requires a provider account", err)
	}
	defer close()
}

func TestArtifactConfigurationRejectsAmbiguousOrCredentialBearingDocuments(t *testing.T) {
	for _, document := range []string{
		`null`, `{}`, `{"driver":"unsupported"}`, `{"driver":"filesystem"}`,
		`{"driver":"filesystem","directory":"private","s3":{"region":"us-east-1","bucket":"evidence"}}`,
		`{"driver":"s3","s3":{"region":"us-east-1","bucket":"evidence","secretAccessKey":"must-not-echo"}}`,
		`{"driver":"s3","s3":{"region":"","bucket":"evidence"}}`,
		`{"driver":"s3","s3":{"region":"us-east-1","bucket":"evidence","prefix":"../escape"}}`,
		`{"driver":"s3","s3":{"region":"us-east-1","bucket":"evidence","expectedOwner":"wrong"}}`,
		`{"driver":"azure","azure":{"serviceUrl":"http://example.com","container":"evidence"}}`,
		`{"driver":"azure","azure":{"serviceUrl":"https://user:must-not-echo@example.com","container":"evidence"}}`,
		`{"driver":"azure","azure":{"serviceUrl":"https://example.com?sig=must-not-echo","container":"evidence"}}`,
		`{"driver":"azure","azure":{"serviceUrl":"https://example.com/path","container":"evidence"}}`,
		`{"driver":"azure","azure":{"serviceUrl":"https://example.com","container":"bad--container"}}`,
		`{"driver":"filesystem","directory":"private"} {}`,
	} {
		file := filepath.Join(t.TempDir(), "store.json")
		if err := os.WriteFile(file, []byte(document), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := config.Load(file, "unused")
		if err == nil {
			t.Fatal("invalid storage configuration was accepted", document)
		}
		if strings.Contains(err.Error(), "must-not-echo") {
			t.Fatal("configuration errors exposed a credential")
		}
	}
}
