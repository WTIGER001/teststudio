package localimage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

var immutable = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var runtimeLabel = regexp.MustCompile(`^(?:go|playwright)[0-9A-Za-z._-]+$`)

const logLimit = 256 << 10

type Builder struct {
	RuntimeImage  string
	TempDirectory string
	Command       func(context.Context, string, ...string) *exec.Cmd
}

func (b *Builder) command(ctx context.Context, args ...string) *exec.Cmd {
	if b.Command != nil {
		return b.Command(ctx, "docker", args...)
	}
	return exec.CommandContext(ctx, "docker", args...)
}
func (b *Builder) Build(ctx context.Context, input preparation.BuildRequest) (result preparation.Build, err error) {
	result.StartedAt = time.Now().UTC()
	defer func() { result.FinishedAt = time.Now().UTC() }()
	if !immutable.MatchString(b.RuntimeImage) || !validReference(input.Reference) {
		return result, errors.New("local source assembly requires an immutable runtime image and preparation ownership")
	}
	if !preparation.OfflineSourceProfile(input.Snapshot.Manifest.Runner, input.Snapshot.Manifest.BuildProfile) {
		return result, errors.New("the local source assembler requires a supported offline Go or Playwright profile")
	}
	if input.Origin.SnapshotDigest != input.Snapshot.Digest || suitespec.ValidateOrigin(input.Origin) != nil {
		return result, errors.New("build provenance must identify the stored source digest")
	}
	wall, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	metadata, err := b.metadata(wall, b.RuntimeImage)
	if err != nil {
		return result, err
	}
	if metadata.Kind != "source-base/v1" || !runtimeLabel.MatchString(metadata.Runtime) || len(metadata.OnBuild) > 0 {
		return result, errors.New("runtime image must be a trusted source base without ONBUILD instructions")
	}
	if !strings.HasPrefix(metadata.Runtime, input.Snapshot.Manifest.Runner) {
		return result, errors.New("source runner does not match the trusted runtime")
	}
	if input.Snapshot.Manifest.Runner == "playwright" {
		if err := runtimeDependencies(input.Snapshot, metadata.PlaywrightVersion); err != nil {
			return result, err
		}
	}
	if existing, err := b.metadata(wall, tag(input.Reference)); err == nil {
		if !matches(existing, input.Reference) || existing.Source != input.Snapshot.Digest {
			return result, errors.New("preparation image reference is already in use")
		}
		return result, errors.New("an existing build reference requires recovery before reuse")
	} else if !errors.Is(err, errImageAbsent) {
		return result, err
	}
	directory := b.contextPath(input.Reference)
	if err := os.MkdirAll(filepath.Dir(directory), 0700); err != nil {
		return result, err
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		return result, errors.New("build context already exists or cannot be reserved; recover its reference first")
	}
	ownership, err := json.Marshal(input.Reference)
	if err != nil {
		return result, err
	}
	if err := os.WriteFile(filepath.Join(directory, "owner.json"), ownership, 0600); err != nil {
		os.Remove(directory)
		return result, err
	}
	defer os.RemoveAll(directory)
	source := filepath.Join(directory, "suite")
	if err := os.Mkdir(source, 0700); err != nil {
		return result, err
	}
	for _, name := range input.Snapshot.Paths() {
		if !sourcebundle.SafePath(name) {
			return result, errors.New("build snapshot contains an unsafe path")
		}
		filename := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			return result, err
		}
		mode := os.FileMode(0600)
		if input.Snapshot.Executable[name] {
			mode = 0700
		}
		if err := os.WriteFile(filename, input.Snapshot.Files[name], mode); err != nil {
			return result, err
		}
	}
	for name, value := range map[string]any{suitespec.ManifestFile: input.Snapshot.Manifest, suitespec.OriginFile: input.Origin} {
		data, err := json.Marshal(value)
		if err != nil {
			return result, err
		}
		if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
			return result, err
		}
	}
	dockerfile := fmt.Sprintf("FROM %s\nLABEL teststudio.project=%q teststudio.installation=%q teststudio.preparation=%q teststudio.source=%q\nCOPY --chown=1000:1000 suite/ /suite/\nWORKDIR /suite\n", b.RuntimeImage, input.Reference.Project, input.Reference.Installation, input.Reference.ID, input.Snapshot.Digest)
	if err := os.WriteFile(filepath.Join(directory, "Dockerfile"), []byte(dockerfile), 0600); err != nil {
		return result, err
	}
	var log boundedLog
	command := b.command(wall, "build", "--pull=false", "--network=none", "--no-cache", "--rm", "--force-rm", "--tag", tag(input.Reference), "--file", filepath.Join(directory, "Dockerfile"), directory)
	command.Stdout, command.Stderr = &log, &log
	defer func() {
		if err == nil {
			return
		}
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer stop()
		if cleanupErr := b.CleanupBuild(cleanup, input.Reference); cleanupErr != nil {
			err = errors.Join(err, preparation.ErrCleanupIncomplete)
		}
	}()
	buildErr := command.Run()
	result.Log, result.LogTruncated = log.String(), log.truncated
	if buildErr != nil {
		if wall.Err() != nil {
			return result, wall.Err()
		}
		return result, errors.New("source image assembly failed; inspect its retained log")
	}
	built, err := b.metadata(wall, tag(input.Reference))
	if err != nil {
		return result, err
	}
	if !matches(built, input.Reference) || built.Source != input.Snapshot.Digest || !immutable.MatchString(built.ID) {
		return result, errors.New("assembled image does not match its preparation ownership")
	}
	result.Image, result.SourceDigest = built.ID, input.Snapshot.Digest
	return result, nil
}

type imageMetadata struct {
	PlaywrightVersion string   `json:"playwrightVersion"`
	ID                string   `json:"id"`
	Runtime           string   `json:"runtime"`
	Kind              string   `json:"kind"`
	Project           string   `json:"project"`
	Installation      string   `json:"installation"`
	Preparation       string   `json:"preparation"`
	Source            string   `json:"source"`
	OnBuild           []string `json:"onBuild"`
}

var errImageAbsent = errors.New("image is absent")

func (b *Builder) metadata(ctx context.Context, image string) (imageMetadata, error) {
	var result imageMetadata
	var output boundedLog
	command := b.command(ctx, "image", "inspect", image, "--format", `{"playwrightVersion":{{json (index .Config.Labels "teststudio.playwright-version")}},"id":{{json .Id}},"runtime":{{json (index .Config.Labels "teststudio.runtime")}},"kind":{{json (index .Config.Labels "teststudio.runtime-kind")}},"project":{{json (index .Config.Labels "teststudio.project")}},"installation":{{json (index .Config.Labels "teststudio.installation")}},"preparation":{{json (index .Config.Labels "teststudio.preparation")}},"source":{{json (index .Config.Labels "teststudio.source")}},"onBuild":{{json (index .Config "OnBuild")}}}`)
	command.Stdout, command.Stderr = &output, &output
	if err := command.Run(); err != nil {
		if strings.Contains(strings.ToLower(output.String()), "no such image") {
			return result, errImageAbsent
		}
		return result, errors.New("local image metadata is unavailable")
	}
	if output.truncated || json.Unmarshal(output.Bytes(), &result) != nil {
		return result, errors.New("invalid local image metadata")
	}
	return result, nil
}
func validReference(ref execution.Reference) bool {
	return domain.ValidID(ref.ID) && domain.ValidID(ref.Project) && domain.ValidID(ref.Installation) && strings.HasPrefix(ref.Installation, "preparation-")
}
func tag(ref execution.Reference) string {
	sum := sha256.Sum256([]byte(ref.Installation + "\x00" + ref.Project + "\x00" + ref.ID))
	return "teststudio-preparation:" + hex.EncodeToString(sum[:])
}
func matches(image imageMetadata, ref execution.Reference) bool {
	return image.Project == ref.Project && image.Installation == ref.Installation && image.Preparation == ref.ID
}
func (b *Builder) CleanupBuild(ctx context.Context, ref execution.Reference) error {
	if !validReference(ref) {
		return errors.New("invalid build cleanup ownership")
	}
	image, err := b.metadata(ctx, tag(ref))
	if err != nil && !errors.Is(err, errImageAbsent) {
		return err
	}
	if err == nil {
		if !matches(image, ref) {
			return errors.New("build cleanup refuses another preparation image")
		}
		var output boundedLog
		command := b.command(ctx, "image", "rm", tag(ref))
		command.Stdout, command.Stderr = &output, &output
		if err := command.Run(); err != nil {
			return errors.New("preparation image reference cleanup failed")
		}
	}
	directory := b.contextPath(ref)
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return errors.New("build cleanup refuses an unowned context directory")
	}
	data, err := os.ReadFile(filepath.Join(directory, "owner.json"))
	if err != nil {
		return err
	}
	var owner execution.Reference
	if len(data) > 8192 || json.Unmarshal(data, &owner) != nil || owner != ref {
		return errors.New("build context ownership does not match its reference")
	}
	return os.RemoveAll(directory)
}
func (b *Builder) contextPath(ref execution.Reference) string {
	root := b.TempDirectory
	if root == "" {
		root = os.TempDir()
	}
	return filepath.Join(root, "teststudio-build-"+strings.TrimPrefix(tag(ref), "teststudio-preparation:"))
}

type boundedLog struct {
	buffer    bytes.Buffer
	truncated bool
}

func (b *boundedLog) Write(data []byte) (int, error) {
	size := len(data)
	remaining := logLimit - b.buffer.Len()
	if len(data) > remaining {
		data = data[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(data)
	return size, nil
}

func (b *boundedLog) String() string { return b.buffer.String() }
func (b *boundedLog) Bytes() []byte  { return b.buffer.Bytes() }
