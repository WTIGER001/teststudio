package ecsworker

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func environmentSettings(t *testing.T, settings Settings) Settings {
	t.Helper()
	guard, _, worker, gateway, _ := gatewayNetwork(t)
	network := guard.settings
	worker.NetworkProfile = ""
	worker.Subnets, worker.SecurityGroups = []string{"subnet-11111111", "subnet-22222222"}, []string{"sg-22222222"}
	network.VPC, network.ExchangeGroup, network.RegistryGroup = "vpc-11111111", "sg-33333333", "sg-44444444"
	network.ExchangeEndpoint, network.APIEndpoint, network.DKREndpoint, network.S3Endpoint = "vpce-77777777777777777", "vpce-88888888888888888", "vpce-99999999999999999", "vpce-aaaaaaaaaaaaaaaaa"
	gateway.Image = strings.TrimPrefix(registryOrigin(worker), "https://") + "/teststudio/gateway@sha256:" + strings.Repeat("b", 64)
	settings.Environment = &EnvironmentSettings{Worker: worker, Network: network, Gateway: gateway}
	bound, err := settings.bind()
	if err != nil {
		t.Fatal("valid separate environment settings rejected", err)
	}
	return bound
}

func TestECSEnvironmentSettingsBindSeparatePoolsAndPreserveOfflineIdentity(t *testing.T) {
	_, original, _, _ := sdkProtocol(t)
	settings := environmentSettings(t, original)
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSettings("", "TESTSTUDIO_ECS_CONFIGURATION", func(string) (string, bool) { return string(data), true })
	if err != nil || loaded.Environment == nil || loaded.Worker.NetworkProfile != original.Worker.NetworkProfile || loaded.Environment.Worker.NetworkProfile == original.Worker.NetworkProfile {
		t.Fatal("environment selection replaced the offline identity", err)
	}
	loaded.Environment.Gateway.Subnets[0] = "subnet-ffffffff"
	if settings.Environment.Gateway.Subnets[0] == loaded.Environment.Gateway.Subnets[0] {
		t.Fatal("loaded settings share mutable gateway configuration")
	}
	if _, err := loaded.bind(); !errors.Is(err, hostedexecution.ErrConfiguration) {
		t.Fatal("changed gateway scope retained its native profile", err)
	}
}

func TestECSEnvironmentSettingsRejectPartialSharedAndForeignProfiles(t *testing.T) {
	_, original, _, _ := sdkProtocol(t)
	valid := environmentSettings(t, original)
	cases := map[string]func(*Settings){
		"missing-network":          func(s *Settings) { s.Environment.Network.Gateway = nil },
		"missing-task":             func(s *Settings) { s.Environment.Gateway = GatewayConfig{} },
		"shared-worker-vpc":        func(s *Settings) { s.Environment.Network.VPC = s.Network.VPC },
		"shared-gateway-vpc":       func(s *Settings) { s.Environment.Network.Gateway.VPC = s.Network.VPC },
		"shared-worker-subnet":     func(s *Settings) { s.Environment.Worker.Subnets = s.Worker.Subnets },
		"shared-worker-group":      func(s *Settings) { s.Environment.Worker.SecurityGroups = s.Worker.SecurityGroups },
		"wrong-task-subnet":        func(s *Settings) { s.Environment.Gateway.Subnets = []string{"subnet-eeeeeeee"} },
		"wrong-registry-namespace": func(s *Settings) { s.Environment.Network.RepositoryPrefix = "teststudio-neighbor" },
		"foreign-gateway-image": func(s *Settings) {
			s.Environment.Gateway.Image = strings.Replace(s.Environment.Gateway.Image, "123456789012", "999999999999", 1)
		},
		"foreign-region":         func(s *Settings) { s.Environment.Worker.Region = "us-west-2" },
		"different-architecture": func(s *Settings) { s.Environment.Worker.Architecture = "amd64" },
		"different-exchange":     func(s *Settings) { s.Environment.Worker.ExchangeOrigin = "https://another.example.test" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			settings := copyValue(valid)
			settings.Environment.Worker.NetworkProfile = ""
			mutate(&settings)
			if _, err := settings.bind(); !errors.Is(err, hostedexecution.ErrConfiguration) {
				t.Fatal("unsupported environment pool retained production selection", err)
			}
		})
	}
}
