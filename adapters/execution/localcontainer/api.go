package localcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
)

func (e *Executor) admitAPIPolicy(ctx context.Context, artifact catalog.Artifact) error {
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, image := range []string{e.GatewayImage, artifact.Image} {
		if !digest.MatchString(image) {
			return errors.New("API request policy requires immutable capable runner and gateway images")
		}
		data, err := e.output(bounded, "image", "inspect", image, "--format", "{{json .Config.Labels}}")
		var labels map[string]string
		if err != nil || json.Unmarshal(data, &labels) != nil || labels["teststudio.api-policy"] != "request-policy/v1" {
			return errors.New("API request policy requires updated request-policy/v1 runner and gateway images")
		}
	}
	return nil
}
