package ecsworker

import (
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

type Settings struct {
	Worker  Config          `json:"worker"`
	Network NetworkSettings `json:"network"`
}

func LoadSettings(file, variable string, lookup func(string) (string, bool)) (*Settings, error) {
	if variable != "" && lookup == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	var settings Settings
	found, err := runtimeconfig.LoadJSON(file, variable, "ECS worker", lookup, &settings)
	if err != nil || !found {
		return nil, err
	}
	settings.Worker, err = settings.Network.Bind(settings.Worker)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}
