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
	data, err := e.output(checkCtx, "image", "inspect", artifact.Image, "--format", "{{json .Config.Labels}}")
	if err != nil {
		return "", errors.New("coded runtime metadata is unavailable")
	}
	var labels map[string]string
	if json.Unmarshal(data, &labels) != nil {
		return "", errors.New("coded runtime metadata is invalid")
	}
	label := strings.TrimSpace(labels["teststudio.runtime"])
	if label == "" && e.runner() == "playwright" {
		return e.Profile(), nil
	}
	if !runtimeLabel.MatchString(label) {
		return "", errors.New("coded image requires a teststudio.runtime label for its toolchain, platform, and execution flags")
	}
	return e.Profile() + ":" + label, nil
}
