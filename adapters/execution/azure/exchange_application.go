package azureworker

import (
	"slices"
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func exchangeTemplateDocument(template *a.Template) map[string]any {
	if template == nil {
		return nil
	}
	doc := document(template)
	delete(doc, "revisionSuffix")
	clearEphemeral(doc["containers"])
	containers, _ := doc["containers"].([]any)
	for _, entry := range containers {
		container, _ := entry.(map[string]any)
		probes, _ := container["probes"].([]any)
		slices.SortFunc(probes, func(left, right any) int {
			a, _ := left.(map[string]any)
			b, _ := right.(map[string]any)
			return strings.Compare(valueString(a["type"]), valueString(b["type"]))
		})
	}
	return doc
}

func valueString(value any) string { text, _ := value.(string); return text }

func (g *ExchangeGuard) verifyApplication(actual a.ContainerApp, domain string) error {
	s, c := g.settings, g.worker
	binding := serviceBinding{application: s.Application, environment: s.environment(), location: c.Region, domain: domain, certificate: s.Certificate, identities: []string{c.PullIdentity}, templateDocument: exchangeTemplateDocument}
	doc, err := binding.applicationDocument(actual)
	if err != nil {
		return err
	}
	expected, _ := s.ApplicationDefinition(c)
	if fingerprint(doc) != fingerprint(expectedServiceDocument(expected, exchangeTemplateDocument)) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (g *ExchangeGuard) verifyRevisions(app a.ContainerApp, revisions []a.Revision) error {
	expected, _ := g.settings.ApplicationDefinition(g.worker)
	return verifyServiceRevisions(app, revisions, g.settings.Application, expected.Properties.Template, exchangeTemplateDocument)
}
