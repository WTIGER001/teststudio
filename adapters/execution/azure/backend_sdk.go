package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKBackendReader struct{ *sdkServiceReader }

func NewSDKBackendReader(c Config, s BackendSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKBackendReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	core, err := newSDKServiceReader(c, s.Environment, s.Application, nil, credential, transport)
	if err != nil {
		return nil, err
	}
	return &SDKBackendReader{core}, nil
}

var _ BackendReader = (*SDKBackendReader)(nil)
