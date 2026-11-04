package azureworker

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

func publicARMOptions(transport policy.Transporter) *arm.ClientOptions {
	return &arm.ClientOptions{DisableRPRegistration: true, ClientOptions: policy.ClientOptions{
		Transport: transport, Retry: policy.RetryOptions{MaxRetries: -1},
		Cloud: cloud.Configuration{ActiveDirectoryAuthorityHost: "https://login.microsoftonline.com/", Services: map[cloud.ServiceName]cloud.ServiceConfiguration{cloud.ResourceManager: {Endpoint: "https://management.azure.com/", Audience: "https://management.core.windows.net/"}}},
	}}
}
