package preparation

import (
	"encoding/json"
	"errors"
	"path"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/sourcebundle"
)

var playwrightVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type browserPackages struct {
	Dependencies map[string]string `json:"dependencies"`
	Development  map[string]string `json:"devDependencies"`
	Optional     map[string]string `json:"optionalDependencies"`
	Peer         map[string]string `json:"peerDependencies"`
	Workspaces   json.RawMessage   `json:"workspaces"`
}

func ValidateBrowserDependencies(snapshot sourcebundle.Snapshot, version string) error {
	if !playwrightVersion.MatchString(version) {
		return errors.New("trusted browser runtime must declare its exact Playwright version")
	}
	var manifest browserPackages
	if json.Unmarshal(snapshot.Files[path.Join(snapshot.Manifest.ModuleRoot, "package.json")], &manifest) != nil {
		return errors.New("Playwright source requires a valid package.json")
	}
	if len(manifest.Optional) > 0 || len(manifest.Peer) > 0 || len(manifest.Workspaces) > 0 {
		return errors.New("the offline browser runtime does not install optional, peer, or workspace dependencies")
	}
	dependencies := map[string]string{}
	for _, group := range []map[string]string{manifest.Dependencies, manifest.Development} {
		for name, value := range group {
			if name != "@playwright/test" || value != version {
				return errors.New("offline browser source must use only the trusted runtime's exact @playwright/test version")
			}
			dependencies[name] = value
		}
	}
	if len(dependencies) != 1 {
		return errors.New("offline browser source requires the trusted runtime's @playwright/test dependency")
	}
	var lock struct {
		Version  int `json:"lockfileVersion"`
		Packages map[string]struct {
			browserPackages
			Version string `json:"version"`
			Link    bool   `json:"link"`
		} `json:"packages"`
	}
	if json.Unmarshal(snapshot.Files[path.Join(snapshot.Manifest.ModuleRoot, "package-lock.json")], &lock) != nil || lock.Version != 3 || len(lock.Packages) != 4 {
		return errors.New("offline browser source requires a version 3 lockfile for the runtime's three Playwright packages")
	}
	root, ok := lock.Packages[""]
	if !ok || root.Link || len(root.Optional) > 0 || len(root.Peer) > 0 || len(root.Workspaces) > 0 || !samePackages(root.Dependencies, manifest.Dependencies) || !samePackages(root.Development, manifest.Development) {
		return errors.New("browser lockfile root does not match package.json")
	}
	for _, name := range []string{"@playwright/test", "playwright", "playwright-core"} {
		entry, ok := lock.Packages["node_modules/"+name]
		if !ok || entry.Version != version || entry.Link {
			return errors.New("browser lockfile does not match the trusted runtime")
		}
	}
	for name := range snapshot.Files {
		for _, part := range strings.Split(name, "/") {
			if part == "node_modules" {
				return errors.New("offline browser bundles must omit node_modules and use the trusted runtime dependencies")
			}
		}
	}
	return nil
}
func samePackages(first, second map[string]string) bool {
	if len(first) != len(second) {
		return false
	}
	for name, value := range first {
		if second[name] != value {
			return false
		}
	}
	return true
}
