package ecsworker

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func gatewaySourceImage(t *testing.T, fixture *sdkFixture) {
	t.Helper()
	var config struct {
		Configuration map[string]any `json:"config"`
	}
	if err := json.Unmarshal(fixture.config, &config); err != nil {
		t.Fatal(err)
	}
	labels := config.Configuration["Labels"].(map[string]any)
	labels["teststudio.job-gateway"], labels["teststudio.job-gateway-address"] = browsergateway.RemoteProtocol, browsergateway.RemoteAddressProtocol
	labels["teststudio.job-source-key"] = strings.Repeat("d", 64)
	var complete map[string]any
	json.Unmarshal(fixture.config, &complete)
	complete["config"] = config.Configuration
	fixture.config, _ = json.Marshal(complete)
	fixture.configID = contentID(fixture.config)
	var manifest map[string]any
	json.Unmarshal(fixture.manifest, &manifest)
	manifest["config"].(map[string]any)["digest"] = fixture.configID
	manifest["config"].(map[string]any)["size"] = len(fixture.config)
	fixture.manifest, _ = json.Marshal(manifest)
	fixture.imageID = contentID(fixture.manifest)
}

func TestECSEnvironmentSDKSelectsDistinctNativeProviderWithSharedRegistry(t *testing.T) {
	for _, role := range []string{"controller", "preparation"} {
		t.Run(role, func(t *testing.T) {
			fixture, settings, store, credentials := sdkProtocol(t)
			settings = environmentSettings(t, settings)
			gatewaySourceImage(t, fixture)
			var authority EnvironmentAuthority
			if role == "controller" {
				authority = &workerexchange.Service{Store: store}
			}
			runtime, err := settings.open(t.Context(), store, nil, credentials, fixture, fixture, authority)
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Close()
			if len(fixture.calls) != 1 || fixture.calls["sts.us-east-1.amazonaws.com"] != 1 || runtime.Provider.environment != nil || runtime.Provider.config.NetworkProfile != settings.Worker.NetworkProfile {
				t.Fatal("factory replaced the offline provider or created native resources")
			}
			if role == "preparation" {
				if runtime.Environment != nil {
					t.Fatal("preparation gained application execution authority")
				}
				return
			}
			if runtime.Environment == nil || runtime.Environment.worker.inspector != runtime.Registry || runtime.Environment.gateway.control.inspector != runtime.Registry || runtime.Environment.authority != authority {
				t.Fatal("environment provider lost its inspected registry or private authority")
			}
			if runtime.Environment.worker.config.NetworkProfile == runtime.Provider.config.NetworkProfile || runtime.Environment.worker.config.NetworkProfile != settings.Environment.Worker.NetworkProfile {
				t.Fatal("native environment and offline execution share a network profile")
			}
			reference := strings.TrimPrefix(registryOrigin(settings.Worker), "https://") + "/teststudio/worker@" + fixture.imageID
			image, err := runtime.Environment.InspectImage(t.Context(), reference)
			if err != nil || image.GatewayAddress != browsergateway.RemoteAddressProtocol || !strings.HasPrefix(image.ExecutionProfile, "fargate-environment-v1-") {
				t.Fatal("environment source capabilities did not reach normal registry admission", err)
			}
		})
	}
}
