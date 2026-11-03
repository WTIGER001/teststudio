package suitespec

import "github.com/wtiger001/teststudio/internal/catalog"

const SchemaVersion = 1
const DiscoverySchema = "teststudio.discovery/v1"
const ManifestFile = "teststudio-suite.json"
const OriginFile = "teststudio-origin.json"

type Tests struct {
	Config   string   `json:"config,omitempty"`
	Packages []string `json:"packages,omitempty"`
	Project  string   `json:"project,omitempty"`
}

type Manifest struct {
	Schema           int      `json:"schemaVersion"`
	Suite            string   `json:"suiteId"`
	Type             string   `json:"type"`
	Runner           string   `json:"runner"`
	Applications     []string `json:"applications"`
	ModuleRoot       string   `json:"moduleRoot"`
	Tests            Tests    `json:"tests"`
	Catalog          string   `json:"catalog"`
	RequestPolicy    string   `json:"requestPolicy,omitempty"`
	RequiredBindings []string `json:"requiredBindings,omitempty"`
	BuildProfile     string   `json:"buildProfile"`
}

type Origin = catalog.Provenance

type DiscoveredTest struct {
	ID         string `json:"id"`
	Selector   string `json:"selector"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	ParentOnly bool   `json:"parentOnly,omitempty"`
}

type Discovery struct {
	Schema       string           `json:"schemaVersion"`
	Manifest     Manifest         `json:"manifest"`
	Origin       Origin           `json:"origin"`
	Catalog      catalog.Document `json:"catalog"`
	Tests        []DiscoveredTest `json:"tests"`
	Log          string           `json:"log,omitempty"`
	LogTruncated bool             `json:"logTruncated,omitempty"`
}
