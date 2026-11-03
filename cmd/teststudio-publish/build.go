package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/wtiger001/teststudio/adapters/preparation/localimage"
	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type savedBuildReference struct {
	Schema           string              `json:"schemaVersion"`
	Reference        execution.Reference `json:"reference"`
	ContextDirectory string              `json:"contextDirectory"`
}

type buildFile struct {
	Schema        string            `json:"schemaVersion"`
	Status        string            `json:"status"`
	Project       string            `json:"projectId"`
	Suite         string            `json:"suiteId"`
	Manifest      string            `json:"manifestPath"`
	ReferenceFile string            `json:"referenceFile"`
	Build         preparation.Build `json:"build"`
	Error         string            `json:"error,omitempty"`
}

func buildBundle(ctx context.Context, args []string, writer io.Writer) error {
	flags := flag.NewFlagSet("build-bundle", flag.ContinueOnError)
	flags.SetOutput(writer)
	archivePath := flags.String("archive", "", "Complete ZIP source context")
	manifestPath := flags.String("manifest", suitespec.ManifestFile, "Manifest path relative to the ZIP root")
	runtimeImage := flags.String("runtime-image", "", "Trusted locally available Go or Playwright source runtime image ID")
	project := flags.String("project", "", "Destination project identifier")
	suite := flags.String("suite", "", "Destination suite identifier")
	output := flags.String("output", "", "New private build result JSON file")
	stateDirectory := flags.String("state-directory", "", "Private build reference directory")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *archivePath == "" || *runtimeImage == "" || !domain.ValidID(*project) || !domain.ValidID(*suite) || *output == "" {
		return errors.New("build-bundle requires archive, runtime-image, project, suite, and a new output path")
	}
	if _, err := os.Lstat(*output); !os.IsNotExist(err) {
		return errors.New("build output must be a new file")
	}
	file, err := os.Open(*archivePath)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return errors.New("source archive must be a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, sourcebundle.CompressedLimit+1))
	file.Close()
	if err != nil {
		return err
	}
	snapshot, err := sourcebundle.Validate(data, *manifestPath, *suite)
	if err != nil {
		return err
	}
	if *stateDirectory == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		*stateDirectory = filepath.Join(cache, "TestStudio", "preparation")
	}
	ref := execution.Reference{ID: domain.NewID(), Project: *project, Installation: "preparation-" + domain.NewID()}
	stateRoot, err := filepath.Abs(*stateDirectory)
	if err != nil {
		return err
	}
	stateFile := filepath.Join(stateRoot, ref.ID+".json")
	contextDirectory := filepath.Join(stateRoot, "context-"+ref.ID)
	if err := writeNew(stateFile, savedBuildReference{Schema: "teststudio.build-reference/v1", Reference: ref, ContextDirectory: contextDirectory}); err != nil {
		return err
	}
	if err := os.Mkdir(contextDirectory, 0700); err != nil {
		return err
	}
	builder := &localimage.Builder{RuntimeImage: *runtimeImage, TempDirectory: contextDirectory}
	fmt.Fprintln(writer, "Assembling the validated source bundle with a trusted local runtime. No tests are started.")
	result, buildErr := builder.Build(ctx, preparation.BuildRequest{Snapshot: snapshot, Origin: suitespec.Origin{SnapshotDigest: snapshot.Digest}, Reference: ref})
	if err := os.RemoveAll(contextDirectory); err != nil {
		return errors.Join(buildErr, err)
	}
	document := buildFile{Schema: "teststudio.build/v1", Status: "assembled", Project: *project, Suite: *suite, Manifest: *manifestPath, ReferenceFile: stateFile, Build: result}
	if buildErr != nil {
		document.Status, document.Error = "failed", buildErr.Error()
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		cleanupErr := builder.CleanupBuild(cleanup, ref)
		stop()
		if cleanupErr == nil {
			if err := os.Remove(stateFile); err != nil {
				return err
			}
			document.ReferenceFile = ""
		}
	}
	if err := writeNew(*output, document); err != nil {
		return errors.Join(buildErr, err)
	}
	if buildErr != nil {
		return fmt.Errorf("source assembly failed; private diagnostics: %s", *output)
	}
	fmt.Fprintln(writer, "Assembled immutable image:", result.Image)
	fmt.Fprintln(writer, "Use prepare to request server inspection and review before publication.")
	fmt.Fprintln(writer, "Retain the image while releases need it. Its scoped removal reference is:", stateFile)
	return nil
}

func cleanupBuildReference(ctx context.Context, data []byte, filename string, writer io.Writer) error {
	var state savedBuildReference
	if err := definitions.Decode("json", string(data), &state); err != nil {
		return err
	}
	if state.Schema != "teststudio.build-reference/v1" {
		return errors.New("unsupported build reference")
	}
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	expected := filepath.Join(filepath.Dir(absolute), "context-"+state.Reference.ID)
	if state.ContextDirectory != expected {
		return errors.New("build context does not match its private reference directory")
	}
	if info, err := os.Lstat(expected); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0) {
		return errors.New("build context must be an owned private directory")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	cleanup, stop := context.WithTimeout(ctx, 15*time.Second)
	defer stop()
	if err := (&localimage.Builder{TempDirectory: state.ContextDirectory}).CleanupBuild(cleanup, state.Reference); err != nil {
		return err
	}
	if err := os.RemoveAll(expected); err != nil {
		return err
	}
	if err := os.Remove(filename); err != nil {
		return err
	}
	fmt.Fprintln(writer, "Owned build image reference is absent.")
	return nil
}
