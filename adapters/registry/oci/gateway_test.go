package oci

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/browsergateway"
)

func TestRegistryGatewayRequiresPinnedPlatformAndPrivateDeliveryCapabilities(t *testing.T) {
	for _, mode := range []string{"valid", "default path", "private certificate", "old protocol", "source entrypoint", "root", "environment", "custom path", "extra environment", "source ownership"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t, "go", false, func(c *configuration, _ *manifest) {
				c.Config.WorkingDir = ""
				c.Config.Entrypoint = []string{"/gateway"}
				c.Config.Labels = map[string]string{"teststudio.gateway-remote": browsergateway.RemoteProtocol, "teststudio.api-policy": "request-policy/v1", "teststudio.api-credentials": "gateway-injection/v1", "teststudio.gateway-config": "private-config/v1"}
				switch mode {
				case "private certificate":
					c.Config.Labels["teststudio.gateway-certificate"] = browsergateway.RemoteCertificateProtocol
				case "default path":
					c.Config.Env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
				case "old protocol":
					c.Config.Labels["teststudio.gateway-remote"] = "unsupported/v0"
				case "source entrypoint":
					c.Config.Entrypoint = []string{"/runner/teststudio-go"}
				case "root":
					c.Config.User = "0:0"
				case "environment":
					c.Config.Env = []string{"PRIVATE_VALUE=fixture-canary"}
				case "custom path":
					c.Config.Env = []string{"PATH=/untrusted"}
				case "extra environment":
					c.Config.Env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "SSL_CERT_FILE=/untrusted/ca.crt"}
				case "source ownership":
					c.Config.Labels["teststudio.project"] = "untrusted-project"
				}
			})
			image, err := f.client.InspectGateway(t.Context(), f.reference)
			if mode != "valid" && mode != "default path" && mode != "private certificate" {
				if err == nil {
					t.Fatal("unsafe gateway image admitted", mode)
				}
				return
			}
			if err != nil || image.Reference != f.reference || image.Architecture != "arm64" || image.Protocol != browsergateway.RemoteProtocol {
				t.Fatal("trusted gateway metadata rejected", image, err)
			}
			wantCertificate := ""
			if mode == "private certificate" {
				wantCertificate = browsergateway.RemoteCertificateProtocol
			}
			if image.CertificateConfiguration != wantCertificate {
				t.Fatal("registry lost the optional private certificate configuration capability")
			}
			if _, err := f.client.Inspect(t.Context(), f.reference); err == nil {
				t.Fatal("gateway image became an uploaded source runtime")
			}
		})
	}
}
