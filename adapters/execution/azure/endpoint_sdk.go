package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKEndpointReader struct{ *sdkEndpointReader }

func NewSDKEndpointReader(c Config, s EndpointSettings, credential azcore.TokenCredential, transport policy.Transporter) (*SDKEndpointReader, error) {
	s, err := s.canonical(c)
	if err != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	core, err := newSDKEndpointReader(c, []string{s.RegistryEndpoint, s.ExchangeEndpoint}, []string{s.RegistryNIC, s.ExchangeNIC}, []string{s.Subnet}, []string{s.SecurityGroup}, credential, transport)
	if err != nil {
		return nil, err
	}
	return &SDKEndpointReader{core}, nil
}

var _ EndpointReader = (*SDKEndpointReader)(nil)
