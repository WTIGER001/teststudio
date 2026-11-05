package registryimage

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func TestRegistryEnvironmentAssemblyRequiresTransportAndRequestPolicyCapabilities(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, mode := range []string{"approved", "old runtime", "missing API policy", "ECS approved", "ECS missing address"} {
			t.Run(runner+"/"+mode, func(t *testing.T) {
				f := fixture(t, runner, func(base map[string]any) {
					labels := base["config"].(map[string]any)["Labels"].(map[string]string)
					labels["teststudio.job-source-key"] = strings.Repeat("c", 64)
					if mode != "old runtime" {
						labels["teststudio.job-gateway"] = browsergateway.RemoteProtocol
					}
					if mode != "missing API policy" {
						labels["teststudio.api-policy"] = "request-policy/v1"
					}
					if mode == "ECS approved" {
						labels["teststudio.job-gateway-address"] = browsergateway.RemoteAddressProtocol
					}
				})
				input := buildInput(t, runner, func(manifest *suitespec.Manifest, test *catalog.Test) {
					manifest.Type, test.Type = "api", "api"
					manifest.BuildProfile = runner + "-environment"
					manifest.RequiredBindings, test.RequiredBindings = []string{"reader"}, []string{"reader"}
					manifest.Browser = &browserpolicy.Spec{Endpoint: "api", Requests: &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "reader", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/status", ReadOnly: true}}, MaxRequests: 2, MaxResponseBytes: 4096}}
				})
				builder := &Builder{RequireGatewayAddress: strings.HasPrefix(mode, "ECS"), Registry: f.client, RuntimeImage: f.base, Directory: filepath.Join(t.TempDir(), "journals")}
				result, err := builder.Build(t.Context(), input)
				if mode != "approved" && mode != "ECS approved" {
					if err == nil || f.posts != 0 || f.manifests != 0 {
						t.Fatal("unsupported environment runtime published source", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				metadata, err := f.client.Inspect(t.Context(), result.Image)
				if err != nil || result.SourceDigest != input.Snapshot.Digest || metadata.Image.GatewayTransport != browsergateway.RemoteProtocol || metadata.Image.APIPolicy != "request-policy/v1" {
					t.Fatal("environment source publication lost its immutable capabilities", err)
				}
				if mode == "ECS approved" && metadata.Image.GatewayAddress != browsergateway.RemoteAddressProtocol {
					t.Fatal("source publication lost the verified-address capability")
				}
				if err := builder.CompleteAssembly(t.Context(), input.Reference); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
