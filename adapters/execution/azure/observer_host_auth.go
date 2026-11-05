package azureworker

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s ObserverHostSettings) AuthDefinition(c Config) (a.AuthConfig, error) {
	if _, err := s.canonical(c); err != nil {
		return a.AuthConfig{}, err
	}
	return a.AuthConfig{Properties: &a.AuthConfigProperties{Platform: &a.AuthPlatform{Enabled: to.Ptr(false)}}}, nil
}

func verifyObserverHostAuthentication(s ObserverHostSettings, auth a.AuthConfig) error {
	if value(auth.ID) == "" || value(auth.Name) == "" || value(auth.Type) == "" || auth.Properties == nil || auth.Properties.Platform == nil || auth.Properties.Platform.Enabled == nil {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(auth.ID), s.Application+"/authConfigs/current") || value(auth.Name) != "current" || !strings.EqualFold(value(auth.Type), "Microsoft.App/containerApps/authConfigs") || value(auth.Properties.Platform.Enabled) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
