package azuresandbox

import azureworker "github.com/wtiger001/teststudio/adapters/execution/azure"

func (s Settings) serviceWorkloadBinding(backend bool) azureworker.ServiceWorkloadBinding {
	v := s.Placement.Services
	if backend {
		b := v.Backend
		return azureworker.ServiceWorkloadBinding{Application: b.Application, Environment: b.Environment, Identities: map[string]azureworker.ServiceIdentityPin{b.Pull.Resource: {Client: b.Pull.Client, Principal: b.Pull.Principal, Lifecycle: "None"}, b.Startup.Resource: {Client: b.Startup.Client, Principal: b.Startup.Principal, Lifecycle: "None"}, b.Runtime.Resource: {Client: b.Runtime.Client, Principal: b.Runtime.Principal, Lifecycle: "Main"}}}
	}
	r := v.Relay
	return azureworker.ServiceWorkloadBinding{Application: r.Application, Environment: s.Placement.Private.ExchangeEnvironment, Identities: map[string]azureworker.ServiceIdentityPin{r.Pull.Resource: {Client: r.Pull.Client, Principal: r.Pull.Principal, Lifecycle: "None"}}}
}
