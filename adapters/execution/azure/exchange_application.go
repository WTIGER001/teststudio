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
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), s.Application, "Microsoft.App/containerApps", value(actual.Location), c.Region) || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != a.ContainerAppProvisioningStateSucceeded || value(p.RunningStatus) != a.ContainerAppRunningStatusRunning && value(p.RunningStatus) != a.ContainerAppRunningStatusReady {
		return hostedexecution.ErrUnavailable
	}
	environment := value(p.ManagedEnvironmentID)
	if environment == "" {
		environment = value(p.EnvironmentID)
	}
	if !strings.EqualFold(environment, s.environment()) || p.EnvironmentID != nil && !strings.EqualFold(value(p.EnvironmentID), s.environment()) || p.Configuration == nil || p.Configuration.Ingress == nil || actual.Identity == nil || len(actual.Identity.UserAssignedIdentities) != 1 {
		return hostedexecution.ErrOwnership
	}
	_, name := resourceParts(s.Application)
	if value(p.Configuration.Ingress.Fqdn) == "" {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(p.Configuration.Ingress.Fqdn), name+"."+domain) {
		return hostedexecution.ErrOwnership
	}
	doc := document(actual)
	if doc == nil {
		return hostedexecution.ErrOwnership
	}
	for _, key := range []string{"id", "name", "type", "tags", "systemData"} {
		delete(doc, key)
	}
	if value(actual.Kind) == "containerapps" {
		delete(doc, "kind")
	}
	properties, _ := doc["properties"].(map[string]any)
	for _, key := range []string{"environmentId", "customDomainVerificationId", "eventStreamEndpoint", "latestReadyRevisionName", "latestRevisionFqdn", "latestRevisionName", "outboundIpAddresses", "provisioningState", "runningStatus"} {
		delete(properties, key)
	}
	properties["managedEnvironmentId"] = s.environment()
	properties["template"] = exchangeTemplateDocument(p.Template)
	identity, _ := doc["identity"].(map[string]any)
	delete(identity, "principalId")
	delete(identity, "tenantId")
	for id := range actual.Identity.UserAssignedIdentities {
		if !strings.EqualFold(id, c.PullIdentity) {
			return hostedexecution.ErrOwnership
		}
	}
	identity["userAssignedIdentities"] = map[string]any{c.PullIdentity: map[string]any{}}
	config, _ := properties["configuration"].(map[string]any)
	if fingerprint(config["dapr"]) == fingerprint(map[string]any{"enabled": false}) {
		delete(config, "dapr")
	}
	settings, _ := config["identitySettings"].([]any)
	for _, entry := range settings {
		item, _ := entry.(map[string]any)
		if strings.EqualFold(valueString(item["identity"]), c.PullIdentity) {
			item["identity"] = c.PullIdentity
		}
	}
	registries, _ := config["registries"].([]any)
	for _, entry := range registries {
		item, _ := entry.(map[string]any)
		if strings.EqualFold(valueString(item["identity"]), c.PullIdentity) {
			item["identity"] = c.PullIdentity
		}
	}
	ingress, _ := config["ingress"].(map[string]any)
	delete(ingress, "fqdn")
	if ingress["exposedPort"] == float64(0) {
		delete(ingress, "exposedPort")
	}
	if ingress["clientCertificateMode"] == string(a.IngressClientCertificateModeIgnore) {
		delete(ingress, "clientCertificateMode")
	}
	if fingerprint(ingress["stickySessions"]) == fingerprint(map[string]any{"affinity": "none"}) {
		delete(ingress, "stickySessions")
	}
	domains, _ := ingress["customDomains"].([]any)
	for _, entry := range domains {
		item, valid := entry.(map[string]any)
		if !valid || item == nil {
			return hostedexecution.ErrOwnership
		}
		item["name"] = strings.ToLower(valueString(item["name"]))
		if strings.EqualFold(valueString(item["certificateId"]), s.Certificate) {
			item["certificateId"] = s.Certificate
		}
	}
	expected, _ := s.ApplicationDefinition(c)
	wanted := document(expected)
	wantedProperties, _ := wanted["properties"].(map[string]any)
	wantedProperties["template"] = exchangeTemplateDocument(expected.Properties.Template)
	if fingerprint(doc) != fingerprint(wanted) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (g *ExchangeGuard) verifyRevisions(app a.ContainerApp, revisions []a.Revision) error {
	if len(revisions) == 0 || value(app.Properties.LatestRevisionName) == "" || value(app.Properties.LatestReadyRevisionName) == "" {
		return hostedexecution.ErrUnavailable
	}
	if len(revisions) > 4 {
		return hostedexecution.ErrOwnership
	}
	if value(app.Properties.LatestRevisionName) != value(app.Properties.LatestReadyRevisionName) {
		return hostedexecution.ErrUnavailable
	}
	expected, _ := g.settings.ApplicationDefinition(g.worker)
	active := 0
	seen := map[string]bool{}
	_, appName := resourceParts(g.settings.Application)
	for _, revision := range revisions {
		name := value(revision.Name)
		if !strings.HasPrefix(name, appName+"--") || !endpointChild(value(revision.ID), name, g.settings.Application, "revisions") || !strings.EqualFold(value(revision.Type), "Microsoft.App/containerApps/revisions") || seen[strings.ToLower(name)] || revision.Properties == nil {
			return hostedexecution.ErrOwnership
		}
		seen[strings.ToLower(name)] = true
		p := revision.Properties
		if p.Active == nil {
			return hostedexecution.ErrUnavailable
		}
		if !value(p.Active) {
			if p.TrafficWeight == nil || p.Replicas == nil || value(p.TrafficWeight) != 0 || value(p.Replicas) != 0 {
				return hostedexecution.ErrUnavailable
			}
			continue
		}
		active++
		if active > 1 || name != value(app.Properties.LatestReadyRevisionName) {
			return hostedexecution.ErrOwnership
		}
		if value(p.ProvisioningState) != a.RevisionProvisioningStateProvisioned || value(p.HealthState) != a.RevisionHealthStateHealthy || value(p.RunningState) != a.RevisionRunningStateRunning || value(p.Replicas) < 1 || value(p.TrafficWeight) != 100 {
			return hostedexecution.ErrUnavailable
		}
		if fingerprint(exchangeTemplateDocument(p.Template)) != fingerprint(exchangeTemplateDocument(expected.Properties.Template)) {
			return hostedexecution.ErrOwnership
		}
	}
	if active != 1 {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
