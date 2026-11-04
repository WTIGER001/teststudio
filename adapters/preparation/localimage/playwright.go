package localimage

import (
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
)

func runtimeDependencies(snapshot sourcebundle.Snapshot, version string) error {
	return preparation.ValidateBrowserDependencies(snapshot, version)
}
