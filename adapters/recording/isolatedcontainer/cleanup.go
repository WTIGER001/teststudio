package isolatedcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wtiger001/teststudio/internal/recordingworker"
)

type resource struct {
	Labels map[string]string `json:"Labels"`
	Config struct {
		Labels map[string]string `json:"Labels"`
		Image  string            `json:"Image"`
	} `json:"Config"`
}

func (r *Runtime) inspect(ctx context.Context, kind, name string) (resource, bool, error) {
	output, err := exec.CommandContext(ctx, r.binary(), "inspect", "--type", kind, name).CombinedOutput()
	if err != nil {
		value := strings.ToLower(string(output))
		if strings.Contains(value, "no such object") || strings.Contains(value, "no such container") || strings.Contains(value, "no such volume") {
			return resource{}, false, nil
		}
		return resource{}, false, errors.New("recording resource inspection is unavailable")
	}
	var items []resource
	if len(output) > 1<<20 || json.Unmarshal(output, &items) != nil || len(items) != 1 {
		return resource{}, false, errors.New("recording resource metadata is invalid")
	}
	return items[0], true, nil
}

func (r *Runtime) Cleanup(ctx context.Context, b recordingworker.Binding, ref recordingworker.Reference) error {
	if !validReference(b, ref) {
		return errors.New("recording cleanup does not match its reserved ownership")
	}
	for _, item := range []struct{ kind, name, image string }{{"container", ref.Container, ref.RuntimeImage}, {"container", ref.Gateway, ref.GatewayImage}, {"volume", ref.Volume, ""}} {
		actual, exists, err := r.inspect(ctx, item.kind, item.name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		actualLabels := actual.Labels
		if item.kind == "container" {
			actualLabels = actual.Config.Labels
		}
		for key, value := range labels(b, ref) {
			if actualLabels[key] != value {
				return errors.New("recording cleanup rejected another resource's ownership")
			}
		}
		if item.image != "" && actual.Config.Image != item.image {
			return errors.New("recording cleanup rejected an unexpected image")
		}
		args := []string{item.kind, "rm"}
		if item.kind == "container" {
			args = append(args, "--force")
		}
		if err := r.run(ctx, append(args, item.name)...); err != nil {
			return err
		}
		if _, exists, err := r.inspect(ctx, item.kind, item.name); err != nil || exists {
			return errors.New("recording resource cleanup is still pending")
		}
	}
	if r.Directory == "" {
		return errors.New("recording cleanup requires its private transport directory")
	}
	return os.RemoveAll(filepath.Join(r.Directory, ref.Fingerprint))
}
