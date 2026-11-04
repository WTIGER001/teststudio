package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func backendTemplateDocument(template *a.Template) map[string]any {
	doc := exchangeTemplateDocument(template)
	if doc == nil {
		return nil
	}
	containers, _ := doc["containers"].([]any)
	for _, item := range containers {
		container, _ := item.(map[string]any)
		env, _ := container["env"].([]any)
		for _, entry := range env {
			variable, _ := entry.(map[string]any)
			if valueString(variable["name"]) == "TESTSTUDIO_ARTIFACT_CONFIGURATION" {
				var configuration map[string]any
				if definitions.DecodeJSON([]byte(valueString(variable["value"])), &configuration, 8192, 4, 64) != nil || configuration == nil {
					return nil
				}
				variable["value"] = configuration
			}
		}
		sortServiceObjects(container["env"], "name")
		probes, _ := container["probes"].([]any)
		for _, entry := range probes {
			probe, _ := entry.(map[string]any)
			if probe["successThreshold"] == float64(1) {
				delete(probe, "successThreshold")
			}
			if probe["initialDelaySeconds"] == float64(0) {
				delete(probe, "initialDelaySeconds")
			}
		}
	}
	scale, _ := doc["scale"].(map[string]any)
	if scale["cooldownPeriod"] == float64(300) {
		delete(scale, "cooldownPeriod")
	}
	if scale["pollingInterval"] == float64(30) {
		delete(scale, "pollingInterval")
	}
	return doc
}

func (g *BackendGuard) verifyApplication(actual a.ContainerApp, domain string) error {
	s := g.settings
	if actual.Identity == nil {
		return hostedexecution.ErrOwnership
	}
	clients, principals := map[string]bool{}, map[string]bool{}
	for id, identity := range actual.Identity.UserAssignedIdentities {
		if identity == nil || !guid.MatchString(strings.ToLower(value(identity.ClientID))) || !guid.MatchString(strings.ToLower(value(identity.PrincipalID))) {
			return hostedexecution.ErrUnavailable
		}
		client, principal := strings.ToLower(value(identity.ClientID)), strings.ToLower(value(identity.PrincipalID))
		if clients[client] || principals[principal] || strings.EqualFold(id, s.RuntimeIdentity) && client != s.RuntimeClientID {
			return hostedexecution.ErrOwnership
		}
		clients[client], principals[principal] = true, true
	}
	binding := serviceBinding{application: s.Application, environment: s.Environment, location: g.worker.Region, domain: domain, identities: []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity}, templateDocument: backendTemplateDocument}
	doc, err := binding.applicationDocument(actual)
	if err != nil {
		return err
	}
	expected, _ := s.ApplicationDefinition(g.worker)
	if fingerprint(doc) != fingerprint(expectedServiceDocument(expected, backendTemplateDocument)) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
