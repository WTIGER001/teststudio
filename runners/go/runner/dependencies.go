package runner

import (
	"os"
	"path/filepath"
)

func moduleDependencyMode(directory string) string {
	if info, err := os.Stat(filepath.Join(directory, "vendor", "modules.txt")); err == nil && info.Mode().IsRegular() {
		return "-mod=vendor"
	}
	return "-mod=readonly"
}
