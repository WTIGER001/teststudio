package oci

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/browsergateway"
)

func TestRegistryGatewayRequiresPinnedPlatformAndPrivateDeliveryCapabilities(t *testing.T) {
	for _, mode := range []string{"valid", "old protocol", "source entrypoint", "root", "environment", "source ownership"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t, "go", false, func(c *configuration, _ *manifest) {
				c.Config.WorkingDir = ""
				c.Config.Entrypoint = []string{"/gateway"}
				c.Config.Labels = map[string]string{"teststudio.gateway-remote": browsergateway.RemoteProtocol, "teststudio.api-policy": "request-policy/v1", "teststudio.api-credentials": "gateway-injection/v1", "teststudio.gateway-config": "private-config/v1"}
				switch mode {
				case "old protocol":
					c.Config.Labels["teststudio.gateway-remote"] = "unsupported/v0"
				case "source entrypoint":
					c.Config.Entrypoint = []string{"/runner/teststudio-go"}
				case "root":
					c.Config.User = "0:0"
				case "environment":
					c.Config.Env = []string{"PRIVATE_VALUE=fixture-canary"}
				case "source ownership":
					c.Config.Labels["teststudio.project"] = "untrusted-project"
				}
			})
			image, err := f.client.InspectGateway(t.Context(), f.reference)
			if mode != "valid" {
				if err == nil {
					t.Fatal("unsafe gateway image admitted", mode)
				}
				return
			}
			if err != nil || image.Reference != f.reference || image.Architecture != "arm64" || image.Protocol != browsergateway.RemoteProtocol {
				t.Fatal("trusted gateway metadata rejected", image, err)
			}
			if _, err := f.client.Inspect(t.Context(), f.reference); err == nil {
				t.Fatal("gateway image became an uploaded source runtime")
			}
		})
	}
}
