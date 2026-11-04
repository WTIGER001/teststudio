package oidc

import (
	"errors"

	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

func InitialProviders(filename, variable string, lookup func(string) (string, bool)) ([]Provider, error) {
	var providers []Provider
	configured, err := runtimeconfig.LoadJSON(filename, variable, "OIDC", lookup, &providers)
	if err != nil || !configured {
		return nil, err
	}
	if providers == nil || len(providers) == 0 || len(providers) > 100 {
		return nil, errors.New("OIDC configuration requires between one and one hundred providers")
	}
	seen := map[string]bool{}
	for _, provider := range providers {
		if err := ValidateProvider(provider); err != nil {
			return nil, err
		}
		if seen[provider.ID] {
			return nil, errors.New("OIDC configuration requires unique provider identifiers")
		}
		seen[provider.ID] = true
	}
	return providers, nil
}
