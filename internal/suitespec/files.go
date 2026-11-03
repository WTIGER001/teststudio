package suitespec

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wtiger001/teststudio/internal/definitions"
)

func File(root, relative string) (string, error) {
	if !RelativePath(relative, true) {
		return "", errors.New("suite file must stay within its source context")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(absolute, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	inside, err := filepath.Rel(realRoot, resolved)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return "", errors.New("suite file escapes its source context")
	}
	return resolved, nil
}

func Read(root, relative string, value any) error {
	filename, err := File(root, relative)
	if err != nil {
		return errors.New("declared suite file is unavailable or escapes its source context")
	}
	info, err := os.Stat(filename)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("suite declarations require regular files")
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("suite declarations require regular files")
	}
	data, err := io.ReadAll(io.LimitReader(file, definitions.ContentLimit+1))
	if err != nil {
		return err
	}
	format := "json"
	if extension := filepath.Ext(relative); extension == ".yaml" || extension == ".yml" {
		format = "yaml"
	}
	return definitions.Decode(format, string(data), value)
}

func Load(root string) (Discovery, error) {
	discovery := Discovery{Schema: DiscoverySchema, Tests: []DiscoveredTest{}}
	if err := Read(root, ManifestFile, &discovery.Manifest); err != nil {
		return discovery, err
	}
	if err := Validate(discovery.Manifest); err != nil {
		return discovery, err
	}
	if err := Read(root, OriginFile, &discovery.Origin); err != nil {
		return discovery, err
	}
	if err := ValidateOrigin(discovery.Origin); err != nil {
		return discovery, err
	}
	if err := Read(root, discovery.Manifest.Catalog, &discovery.Catalog); err != nil {
		return discovery, err
	}
	return discovery, ValidateCatalog(discovery.Manifest, discovery.Catalog)
}
