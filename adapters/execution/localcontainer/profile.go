package localcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
)

var runtimeLabel = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$`)

func (e *Executor) TimingProfile(ctx context.Context, artifact catalog.Artifact) (string, error) {
	if !digest.MatchString(artifact.Image) {
		return "", errors.New("timing requires an immutable coded runner image")
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	data, err := e.output(checkCtx, "image", "inspect", artifact.Image, "--format", `{"labels":{{json .Config.Labels}},"id":{{json .Id}},"os":{{json .Os}},"architecture":{{json .Architecture}}}`)
	if err != nil {
		return "", errors.New("coded runtime metadata is unavailable")
	}
	var metadata struct {
		Labels       map[string]string `json:"labels"`
		ID           string            `json:"id"`
		OS           string            `json:"os"`
		Architecture string            `json:"architecture"`
	}
	if json.Unmarshal(data, &metadata) != nil || !strings.HasPrefix(metadata.ID, "sha256:") || !digest.MatchString(metadata.ID) || !runtimeLabel.MatchString(metadata.OS) || !runtimeLabel.MatchString(metadata.Architecture) {
		return "", errors.New("coded runtime metadata requires an immutable local image, operating system, and architecture")
	}
	platform := metadata.OS + "-" + metadata.Architecture
	label := strings.TrimSpace(metadata.Labels["teststudio.runtime"])
	if label == "" && e.runner() == "playwright" {
		return e.Profile() + ":unlabelled-" + strings.TrimPrefix(metadata.ID, "sha256:") + ":" + platform, nil
	}
	if !runtimeLabel.MatchString(label) {
		return "", errors.New("coded image requires a teststudio.runtime label for its toolchain, platform, and execution flags")
	}
	return e.Profile() + ":" + label + ":" + platform, nil
}
