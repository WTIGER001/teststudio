package registryimage

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func buildInput(t *testing.T, runner string) preparation.BuildRequest {
	t.Helper()
	manifest := suitespec.Manifest{Schema: 1, Suite: "invoices", Type: "unit", Runner: runner, ModuleRoot: "checks", Tests: suitespec.Tests{Packages: []string{"./invoice"}}, Catalog: "catalog.json", BuildProfile: "go-module"}
	test := catalog.Test{ID: "INVOICE-7", Title: "Invoice arithmetic", Type: "unit", Runner: runner, State: "enabled", Safety: "safe", Rationale: "Uses in-memory values only.", TimeoutSeconds: 10, Source: "checks/invoice/invoice_test.go", Selector: "./invoice::TestInvoice"}
	files := map[string][]byte{"checks/go.mod": []byte("module invoicefixture\n\ngo 1.26.2\n"), "checks/invoice/invoice_test.go": []byte("package invoice\nimport \"testing\"\nfunc TestInvoice(t *testing.T) { if 7*3 != 21 { t.Fatal(\"invoice\") } }\n"), "Dockerfile": []byte("RUN must-never-execute"), ".dockerignore": []byte("*"), "teststudio-origin.json": []byte(`{"untrusted":"replaced"}`)}
	if runner == "playwright" {
		manifest.Type, manifest.BuildProfile, manifest.Tests = "browser", "playwright-node", suitespec.Tests{Config: "playwright.config.ts"}
		test.Type, test.Source, test.Selector = "browser", "checks/invoice.spec.ts", "@INVOICE-7"
		files = map[string][]byte{"checks/invoice.spec.ts": []byte("import { test, expect } from '@playwright/test'; test('invoice @INVOICE-7', () => { expect(7*3).toBe(21); });"), "checks/playwright.config.ts": []byte("export default { testDir: '.' };"), "Dockerfile": []byte("RUN must-never-execute")}
		dependencies := map[string]string{"@playwright/test": "1.64.1"}
		files["checks/package.json"], _ = json.Marshal(map[string]any{"devDependencies": dependencies})
		packages := map[string]any{"": map[string]any{"devDependencies": dependencies}}
		for _, name := range []string{"@playwright/test", "playwright", "playwright-core"} {
			packages["node_modules/"+name] = map[string]any{"version": "1.64.1"}
		}
		files["checks/package-lock.json"], _ = json.Marshal(map[string]any{"lockfileVersion": 3, "packages": packages})
	}
	files["suite.json"], _ = json.Marshal(manifest)
	files["catalog.json"], _ = json.Marshal(catalog.Document{Schema: catalog.SchemaVersion, Suite: "invoices", Tests: []catalog.Test{test}})
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		file.Write(files[name])
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := sourcebundle.Validate(output.Bytes(), "suite.json", "invoices")
	if err != nil {
		t.Fatal(err)
	}
	return preparation.BuildRequest{Snapshot: snapshot, Origin: suitespec.Origin{SnapshotDigest: snapshot.Digest}, Reference: execution.Reference{ID: "source-job", Installation: "preparation-registry", Project: "invoicing"}}
}

func TestRegistryAssemblyRejectsRuntimeWithoutHostedDiscoveryBeforePublication(t *testing.T) {
	f := fixture(t, "go", func(base map[string]any) {
		delete(base["config"].(map[string]any)["Labels"].(map[string]string), "teststudio.discovery-supervisor")
	})
	input := buildInput(t, "go")
	builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
	if _, err := builder.Build(t.Context(), input); err == nil {
		t.Fatal("runtime without hosted discovery was published")
	}
	if f.posts != 0 || f.manifests != 0 {
		t.Fatal("unsupported runtime created registry uploads")
	}
	if err := builder.CleanupBuild(t.Context(), input.Reference); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryAssemblyPublishesBothRunnersWithFrozenRuntimeAndSource(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f := fixture(t, runner, nil)
			input := buildInput(t, runner)
			builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
			result, err := builder.Build(t.Context(), input)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(result.Image, strings.Split(f.base, "@")[0]+"@sha256:") || result.SourceDigest != input.Snapshot.Digest || result.FinishedAt.Before(result.StartedAt) {
				t.Fatal(result)
			}
			metadata, err := f.client.Inspect(t.Context(), result.Image)
			if err != nil || metadata.Project != "invoicing" || metadata.Installation != input.Reference.Installation || metadata.Preparation != input.Reference.ID || metadata.Source != input.Snapshot.Digest || metadata.Image.Runner != runner {
				t.Fatal(metadata, err)
			}
			f.mu.Lock()
			var configuration map[string]any
			json.Unmarshal(f.objects[metadata.Configuration], &configuration)
			config := configuration["config"].(map[string]any)
			if config["StopSignal"] != "SIGTERM" || len(config["Env"].([]any)) != 2 || len(configuration["rootfs"].(map[string]any)["diff_ids"].([]any)) != 2 || f.posts != 2 || len(f.sessions) != 0 || !slices.Contains(f.scopes, "repository:teststudio/source:pull,push") {
				t.Fatal("source assembly changed the fixed runtime or registry scope")
			}
			f.mu.Unlock()
			if _, err := builder.Build(t.Context(), input); err == nil {
				t.Fatal("a reference was reused without recovery")
			}
			if err := builder.CleanupBuild(t.Context(), input.Reference); err != nil {
				t.Fatal(err)
			}
			repeated, err := builder.Build(t.Context(), input)
			if err != nil || repeated.Image != result.Image {
				t.Fatal("same preparation input produced another artifact", repeated, err)
			}
			f.mu.Lock()
			if f.posts != 2 {
				t.Error("identical content was uploaded again")
			}
			f.mu.Unlock()
			if err := builder.CleanupBuild(t.Context(), input.Reference); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRegistryAssemblyRecoversLostBlobAndManifestAcknowledgements(t *testing.T) {
	for _, mode := range []string{"lost-blob-ack", "lost-manifest-ack", "concurrent-blob"} {
		t.Run(mode, func(t *testing.T) {
			f := fixture(t, "go", nil)
			f.mode = mode
			input := buildInput(t, "go")
			builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
			if result, err := builder.Build(t.Context(), input); err != nil || result.Image == "" {
				t.Fatal(result, err)
			}
			f.mu.Lock()
			pending := len(f.sessions)
			f.mu.Unlock()
			if pending != 0 {
				t.Fatal("acknowledgement recovery left an upload session active")
			}
			if err := builder.CleanupBuild(t.Context(), input.Reference); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRegistryAssemblyCancellationAndRestartCleanupUseOnlyOwnedSessions(t *testing.T) {
	for _, mode := range []string{"cancel", "cleanup-failure"} {
		t.Run(mode, func(t *testing.T) {
			f := fixture(t, "go", nil)
			input := buildInput(t, "go")
			builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if mode == "cancel" {
				f.cancel = cancel
			} else {
				f.mode = mode
			}
			result, err := builder.Build(ctx, input)
			if err == nil || result.Image != "" || strings.Contains(err.Error(), "private-") {
				t.Fatal(result, err)
			}
			f.mu.Lock()
			pending, published := len(f.sessions), f.manifests
			f.mode, f.password = "", "rotated-private-password"
			f.mu.Unlock()
			if published != 0 || mode == "cancel" && pending != 0 || mode == "cleanup-failure" && (pending != 1 || !errors.Is(err, preparation.ErrCleanupIncomplete)) {
				t.Fatal("cleanup lost pending-session ownership", pending, published, err)
			}
			restarted := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: builder.Directory}
			if err := restarted.CleanupBuild(t.Context(), input.Reference); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(builder.Directory)
			if err != nil || len(entries) != 0 {
				t.Fatal("recovery left private journal state", entries, err)
			}
		})
	}
}

func TestRegistryAssemblyDoesNotInventCleanupAfterUnknownCreation(t *testing.T) {
	for _, mode := range []string{"lost-opening", "foreign-upload"} {
		t.Run(mode, func(t *testing.T) {
			f := fixture(t, "go", nil)
			f.mode = mode
			input := buildInput(t, "go")
			builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
			if result, err := builder.Build(t.Context(), input); result.Image != "" || !errors.Is(err, preparation.ErrCleanupIncomplete) || strings.Contains(err.Error(), "private-") || strings.Contains(err.Error(), "credential-canary") {
				t.Fatal(result, err)
			}
			if err := builder.CleanupBuild(t.Context(), input.Reference); !errors.Is(err, preparation.ErrCleanupIncomplete) {
				t.Fatal("unknown registry session became cleaned", err)
			}
			if _, err := builder.Build(t.Context(), input); err == nil {
				t.Fatal("uncertain preparation was automatically repeated")
			}
		})
	}
}

func TestSourceLayerIsDeterministicConfinedAndReplacesUploadedMetadata(t *testing.T) {
	input := buildInput(t, "go")
	input.Snapshot.Executable["checks/invoice/invoice_test.go"] = true
	one, diff, err := sourceLayer(t.Context(), input.Snapshot, input.Origin)
	if err != nil {
		t.Fatal(err)
	}
	two, second, err := sourceLayer(t.Context(), input.Snapshot, input.Origin)
	if err != nil || !bytes.Equal(one, two) || diff != second {
		t.Fatal("layer generation is not deterministic", err)
	}
	compressed, err := gzip.NewReader(bytes.NewReader(one))
	if err != nil {
		t.Fatal(err)
	}
	defer compressed.Close()
	uncompressed, _ := io.ReadAll(compressed)
	if checksum(uncompressed) != diff {
		t.Fatal("uncompressed layer digest differs")
	}
	archive := tar.NewReader(bytes.NewReader(uncompressed))
	seen := map[string]bool{}
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil || seen[header.Name] || header.Uid != 1000 || header.Gid != 1000 || header.Mode&0077 != 0 || header.Name != "suite" && !strings.HasPrefix(header.Name, "suite/") || header.Typeflag != tar.TypeDir && header.Typeflag != tar.TypeReg {
			t.Fatal(header, err)
		}
		seen[header.Name] = true
		data, _ := io.ReadAll(archive)
		if header.Name == "suite/teststudio-origin.json" {
			var origin suitespec.Origin
			if json.Unmarshal(data, &origin) != nil || origin != input.Origin {
				t.Fatal("uploaded origin overrode retained provenance")
			}
		}
		if header.Name == "suite/checks/invoice/invoice_test.go" && header.Mode != 0700 {
			t.Fatal("executable source mode was lost")
		}
	}
	if !seen["suite/Dockerfile"] || !seen["suite/teststudio-suite.json"] || !seen["suite/teststudio-origin.json"] {
		t.Fatal("assembly lost source or server metadata")
	}
	for _, name := range []string{"../escape", "checks/.wh.runner", "teststudio-suite.json/child"} {
		input := buildInput(t, "go")
		input.Snapshot.Files[name] = []byte("reject")
		if _, _, err := sourceLayer(t.Context(), input.Snapshot, input.Origin); err == nil {
			t.Fatal("unsafe source layer was accepted", name)
		}
	}
}

func TestRegistryAssemblyRejectsAnExpandedRegistryWriteScope(t *testing.T) {
	f := fixture(t, "go", nil)
	f.mode = "extra-scope"
	builder := &Builder{Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "private")}
	input := buildInput(t, "go")
	if result, err := builder.Build(t.Context(), input); err == nil || result.Image != "" || errors.Is(err, preparation.ErrCleanupIncomplete) {
		t.Fatal(result, err)
	}
	f.mu.Lock()
	posts, scopes := f.posts, slices.Clone(f.scopes)
	f.mu.Unlock()
	if posts != 0 {
		t.Fatal("unapproved write scope opened a registry session")
	}
	for _, scope := range scopes {
		if strings.Contains(scope, "delete") {
			t.Fatal("credential exchange escalated its scope", scope)
		}
	}
}
