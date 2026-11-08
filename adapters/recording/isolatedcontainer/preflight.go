package isolatedcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"time"

	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/recordingworker"
)

func (r *Runtime) Preflight(ctx context.Context) error {
	if _, err := r.Plan(recordingworker.Binding{}); err != nil {
		return err
	}
	if r.Directory == "" {
		return errors.New("recording worker requires a private transport directory")
	}
	if err := os.MkdirAll(r.Directory, 0700); err != nil {
		return errors.New("recording worker transport storage is unavailable")
	}
	info, err := os.Lstat(r.Directory)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.New("recording worker transport storage must be a private regular directory")
	}
	wall, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for _, image := range []string{r.Image, r.GatewayImage} {
		output, err := exec.CommandContext(wall, r.binary(), "image", "inspect", image, "--format", "{{json .Config}}").Output()
		var config struct {
			User   string
			Labels map[string]string
		}
		if err != nil || len(output) > 64<<10 || json.Unmarshal(output, &config) != nil || config.User != "1000:1000" {
			return errors.New("recording requires current trusted non-root runtime images installed on this worker")
		}
		if image == r.Image && (config.Labels["teststudio.recording-runtime"] != "isolated-browser/v1" || config.Labels["teststudio.playwright-version"] != recording.PlaywrightVersion) {
			return errors.New("the installed recording runtime protocol does not match")
		}
		if image == r.GatewayImage && config.Labels["teststudio.gateway-config"] != "private-config/v1" {
			return errors.New("the installed recording gateway protocol does not match")
		}
	}
	return nil
}
