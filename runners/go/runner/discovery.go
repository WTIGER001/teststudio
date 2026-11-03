package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/gosuite"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func Discover(ctx context.Context, root string) (suitespec.Discovery, error) {
	discovery, err := suitespec.Load(root)
	if err != nil {
		return discovery, err
	}
	if discovery.Manifest.Runner != "go" {
		return discovery, errors.New("Go discovery requires a Go suite manifest")
	}
	module, err := suitespec.File(root, discovery.Manifest.ModuleRoot)
	if err != nil {
		return discovery, err
	}
	temp, err := os.MkdirTemp("", "teststudio-go-discovery-")
	if err != nil {
		return discovery, err
	}
	defer os.RemoveAll(temp)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	environment := append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=", "CGO_ENABLED=0")
	declarations := map[string]map[string][]suitespec.DiscoveredTest{}
	appendLog := func(value *output) {
		text := value.buffer.String()
		remaining := (128 << 10) - len(discovery.Log)
		if len(text) > remaining {
			text = text[:max(remaining, 0)]
			discovery.LogTruncated = true
		}
		discovery.Log += text
		discovery.LogTruncated = discovery.LogTruncated || value.truncated
	}
	for index, pkg := range discovery.Manifest.Tests.Packages {
		var metadata struct {
			Dir          string
			TestGoFiles  []string
			XTestGoFiles []string
		}
		list := command(ctx, "list", "-json", "-mod=readonly", pkg)
		list.Dir, list.Env = module, environment
		var listed, diagnostics output
		list.Stdout, list.Stderr = &listed, &diagnostics
		listErr := list.Run()
		appendLog(&diagnostics)
		if listErr != nil || listed.truncated || json.Unmarshal(listed.buffer.Bytes(), &metadata) != nil {
			return discovery, fmt.Errorf("Go package %s metadata is unavailable", pkg)
		}
		declarations[pkg] = map[string][]suitespec.DiscoveredTest{}
		for _, filename := range append(metadata.TestGoFiles, metadata.XTestGoFiles...) {
			relative := filepath.ToSlash(filepath.Join(discovery.Manifest.ModuleRoot, strings.TrimPrefix(pkg, "./"), filename))
			source, err := suitespec.File(root, relative)
			if err != nil {
				return discovery, errors.New("Go test source escapes the suite context")
			}
			files := token.NewFileSet()
			tree, err := parser.ParseFile(files, source, nil, 0)
			if err != nil {
				return discovery, fmt.Errorf("Go package %s test source cannot be parsed", pkg)
			}
			examples := map[string]bool{}
			for _, example := range doc.Examples(tree) {
				if example.Output != "" || example.EmptyOutput {
					examples["Example"+example.Name] = true
				}
			}
			for _, node := range tree.Decls {
				function, ok := node.(*ast.FuncDecl)
				if !ok || function.Recv != nil {
					continue
				}
				name := function.Name.Name
				if !runnableRoot(name) && !examples[name] {
					continue
				}
				declarations[pkg][name] = append(declarations[pkg][name], suitespec.DiscoveredTest{File: relative, Line: files.Position(function.Pos()).Line})
			}
		}
		build := command(ctx, "test", "-c", "-mod=readonly", "-o", filepath.Join(temp, fmt.Sprintf("package-%d.test", index)), pkg)
		build.Dir, build.Env = module, environment
		var compiled output
		build.Stdout, build.Stderr = &compiled, &compiled
		buildErr := build.Run()
		appendLog(&compiled)
		if buildErr != nil {
			return discovery, fmt.Errorf("Go package %s preparation failed", pkg)
		}
		if info, err := os.Stat(filepath.Join(temp, fmt.Sprintf("package-%d.test", index))); err != nil || !info.Mode().IsRegular() {
			return discovery, fmt.Errorf("Go package %s produced no test executable", pkg)
		}
	}
	for _, test := range discovery.Catalog.Tests {
		entry, _ := gosuite.Parse(test.Selector)
		parent, _, child := strings.Cut(entry.Test, "/")
		matches := declarations[entry.Package][parent]
		if len(matches) != 1 || matches[0].File != test.Source {
			return discovery, fmt.Errorf("Go test %s has no unique matching source declaration", test.ID)
		}
		found := matches[0]
		found.ID, found.Selector, found.ParentOnly = test.ID, test.Selector, child
		discovery.Tests = append(discovery.Tests, found)
	}
	return discovery, nil
}

func runnableRoot(name string) bool {
	if name == "TestMain" {
		return false
	}
	for _, prefix := range []string{"Test", "Fuzz"} {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if len(name) == len(prefix) {
			return true
		}
		next, _ := utf8.DecodeRuneInString(strings.TrimPrefix(name, prefix))
		return !unicode.IsLower(next)
	}
	return false
}
