package azureworker

import (
	"net/url"
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyObserverHostRuntime(c Config, s ObserverHostSettings, actual a.ContainerApp) (a.ContainerApp, error) {
	expected, err := s.ApplicationDefinition(c)
	if err != nil {
		return a.ContainerApp{}, err
	}
	if actual.Properties.Configuration.Ingress == nil || actual.Properties.Template.Scale == nil || len(actual.Properties.Configuration.Registries) == 0 || len(actual.Properties.Configuration.Secrets) == 0 {
		return a.ContainerApp{}, hostedexecution.ErrUnavailable
	}
	ui, _ := url.Parse(s.Runtime.UIOrigin)
	_, name := resourceParts(s.Application)
	binding := serviceBinding{application: s.Application, environment: c.ControllerEnvironment, location: c.Region, domain: strings.TrimPrefix(ui.Hostname(), name+"."), identities: []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity, s.Identity.Identity}, templateDocument: backendTemplateDocument}
	doc, err := binding.applicationDocument(actual)
	if err != nil {
		return a.ContainerApp{}, err
	}
	properties, _ := doc["properties"].(map[string]any)
	if properties["workloadProfileName"] == "Consumption" {
		delete(properties, "workloadProfileName")
	}
	configuration, _ := properties["configuration"].(map[string]any)
	if maximum := actual.Properties.Configuration.MaxInactiveRevisions; maximum != nil {
		if *maximum < 1 || *maximum > 100 {
			return a.ContainerApp{}, hostedexecution.ErrOwnership
		}
		delete(configuration, "maxInactiveRevisions")
	}
	if fingerprint(doc) != fingerprint(expectedServiceDocument(expected, backendTemplateDocument)) {
		return a.ContainerApp{}, hostedexecution.ErrOwnership
	}
	return expected, nil
}
