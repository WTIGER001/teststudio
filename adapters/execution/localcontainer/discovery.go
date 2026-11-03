package localcontainer

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

var _ preparation.ArtifactInspector = (*Executor)(nil)

func (e *Executor) Inspect(ctx context.Context, input preparation.InspectionRequest) (inspection preparation.Inspection, resultErr error) {
	inspection.Image, inspection.StartedAt = input.Image, time.Now().UTC()
	defer func() { inspection.FinishedAt = time.Now().UTC() }()
	ref := input.Reference
	if err := e.validateInspection(input); err != nil {
		return inspection, err
	}
	if !digest.MatchString(input.Image) || !slices.Contains(e.Images, input.Image) || (e.runner() != "go" && e.runner() != "playwright") {
		return inspection, errors.New("discovery image or runner is not enabled in this installation")
	}
	wall, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	metadata, err := e.output(wall, "image", "inspect", input.Image, "--format", `{"id":{{json .Id}},"runtime":{{json (index .Config.Labels "teststudio.runtime")}}}`)
	if err != nil {
		return inspection, errors.New("approved discovery image is unavailable locally")
	}
	var image struct {
		ID      string `json:"id"`
		Runtime string `json:"runtime"`
	}
	if json.Unmarshal(metadata, &image) != nil || !strings.HasPrefix(image.ID, "sha256:") || !digest.MatchString(image.ID) || !runtimeLabel.MatchString(image.Runtime) {
		return inspection, errors.New("prepared images require valid immutable image and runtime metadata")
	}
	inspection.ImageID, inspection.Runtime = image.ID, image.Runtime

	if info, exists, err := e.container(wall, ref); err != nil {
		return inspection, err
	} else if exists || info.ID != "" {
		return inspection, errors.New("preparation reference already has a container; recover it before retrying")
	}
	if err := e.prepareVolume(wall, ref); err != nil {
		return inspection, err
	}
	args := []string{"create", "--pull=never", "--name", ref.Container, "--init", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=256", "--memory=2g", "--cpus=2", "--shm-size=256m", "--log-driver=none", "--tmpfs", "/tmp:rw,exec,nosuid,nodev,size=1g,mode=1777", "--user", "1000:1000", "--env", "HOME=/tmp", "--mount", "type=volume,src=" + ref.Volume + ",dst=/output"}
	for name, value := range labels(ref) {
		args = append(args, "--label", name+"="+value)
	}
	args = append(args, input.Image, "--discover")
	if _, err := e.output(wall, args...); err != nil {
		return inspection, errors.New("preparation container reservation failed; recover its reference before retrying")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := e.Cleanup(cleanup, ref); err != nil {
			resultErr = errors.Join(resultErr, errors.New("preparation resource cleanup is incomplete"))
		}
	}()
	command := exec.CommandContext(wall, e.binary(), "start", "--attach", ref.Container)
	var log boundedOutput
	command.Stdout, command.Stderr = &log, &log
	command.WaitDelay = 3 * time.Second
	err = command.Run()
	inspection.Log, inspection.LogTruncated = string(log.Bytes()), log.Truncated
	if err != nil {
		if ctx.Err() != nil {
			return inspection, ctx.Err()
		}
		if wall.Err() != nil {
			return inspection, errors.New("artifact discovery exceeded its preparation deadline")
		}
		return inspection, errors.New("artifact discovery failed; inspect the bounded preparation log")
	}
	info, exists, err := e.container(wall, ref)
	if err != nil || !exists || info.State.Running || info.State.ExitCode != 0 || info.Config.Image != input.Image {
		return inspection, errors.New("discovery container state does not prove successful completion")
	}
	copyCtx, cancelCopy := context.WithTimeout(wall, 10*time.Second)
	defer cancelCopy()
	report, err := e.discoveryFile(copyCtx, ref)
	if err != nil {
		return inspection, err
	}
	discovery, err := suitespec.DecodeDiscovery(report, e.runner())
	if err != nil {
		return inspection, err
	}
	if discovery.Manifest.Runner != e.runner() || discovery.Manifest.Suite != input.Suite {
		return inspection, errors.New("discovered suite or runner differs from its registration destination")
	}
	profile := "go-module"
	if e.runner() == "playwright" {
		profile = "playwright-node"
	}
	if discovery.Manifest.RequestPolicy != "" || len(discovery.Manifest.RequiredBindings) != 0 {
		return inspection, errors.New("this local inspection profile does not support request policies or execution bindings")
	}
	if discovery.Manifest.BuildProfile != profile {
		return inspection, errors.New("manifest build profile is unsupported by this local inspector")
	}
	if err := suitespec.ValidateDiscovery(discovery, input.Image); err != nil {
		return inspection, err
	}
	inspection.Discovery = discovery
	return inspection, nil
}

func (e *Executor) discoveryFile(ctx context.Context, ref execution.Reference) ([]byte, error) {
	command := exec.CommandContext(ctx, e.binary(), "cp", ref.Container+":/output", "-")
	command.WaitDelay = 2 * time.Second
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr boundedOutput
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, errors.New("discovery report could not be read")
	}
	data, readErr := io.ReadAll(io.LimitReader(stdout, suitespec.DiscoveryLimit+(64<<10)+1))
	if len(data) > suitespec.DiscoveryLimit+(64<<10) || readErr != nil {
		_ = command.Process.Kill()
	}
	waitErr := command.Wait()
	if readErr != nil || waitErr != nil || len(data) > suitespec.DiscoveryLimit+(64<<10) {
		return nil, errors.New("discovery report is unavailable or exceeds its byte limit")
	}
	reader := tar.NewReader(bytes.NewReader(data))
	var report []byte
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("discovery output archive is invalid")
		}
		name := strings.TrimSuffix(header.Name, "/")
		if name == "output" && header.Typeflag == tar.TypeDir {
			continue
		}
		if name != "output/discovery.json" || header.Typeflag != tar.TypeReg || header.Size < 1 || header.Size > suitespec.DiscoveryLimit || report != nil {
			return nil, errors.New("discovery output must contain only one bounded regular report file")
		}
		report, err = io.ReadAll(io.LimitReader(reader, suitespec.DiscoveryLimit+1))
		if err != nil || int64(len(report)) != header.Size {
			return nil, errors.New("discovery report is incomplete")
		}
	}
	if report == nil {
		return nil, errors.New("discovery report is unavailable")
	}
	return report, nil
}

func (e *Executor) validateInspection(input preparation.InspectionRequest) error {
	ref := input.Reference
	expected := e.Reference(ref)
	if !domain.ValidID(input.Suite) || !domain.ValidID(ref.ID) || !domain.ValidID(ref.Project) || !domain.ValidID(ref.Worker) || !strings.HasPrefix(ref.Installation, "preparation-") || !domain.ValidID(ref.Installation) || !domain.ValidID(ref.Run) || ref.Test != "catalog" || ref.Attempt < 1 || ref.Attempt > 1000 || ref.Container != expected.Container || ref.Volume != expected.Volume || ref.Profile != e.Profile() {
		return errors.New("discovery requires its own preparation resource reference")
	}

	return nil
}

func (e *Executor) CleanupInspection(ctx context.Context, input preparation.InspectionRequest) error {
	if err := e.validateInspection(input); err != nil {
		return err
	}
	return e.Cleanup(ctx, input.Reference)
}
