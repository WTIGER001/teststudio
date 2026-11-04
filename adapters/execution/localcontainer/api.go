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
		if image == artifact.Image && labels["teststudio.work-item-bytes"] != "4194304" {
			return errors.New("API suites require an updated runtime with the bounded 4 MiB work-item contract")
		}
	}
	return nil
}

func (e *Executor) admitAPICredentials(ctx context.Context) error {
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	data, err := e.output(bounded, "image", "inspect", e.GatewayImage, "--format", "{{json .Config.Labels}}")
	var labels map[string]string
	if err != nil || json.Unmarshal(data, &labels) != nil || labels["teststudio.api-credentials"] != "gateway-injection/v1" {
		return errors.New("API credentials require a gateway-injection/v1 trusted gateway image")
	}
	return nil
}

func (e *Executor) admitGatewayConfiguration(ctx context.Context) error {
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	data, err := e.output(bounded, "image", "inspect", e.GatewayImage, "--format", "{{json .Config.Labels}}")
	var labels map[string]string
	if err != nil || json.Unmarshal(data, &labels) != nil || labels["teststudio.gateway-config"] != "private-config/v1" {
		return errors.New("environment execution requires a private-config/v1 trusted gateway image")
	}
	return nil
}
