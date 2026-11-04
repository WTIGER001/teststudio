package azureworker

import (
	"slices"
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type serviceBinding struct {
	application, environment, location, domain, certificate string
	identities                                              []string
	templateDocument                                        func(*a.Template) map[string]any
}

func (b serviceBinding) applicationDocument(actual a.ContainerApp) (map[string]any, error) {
	if !perimeterIdentity(value(actual.ID), value(actual.Name), value(actual.Type), b.application, "Microsoft.App/containerApps", value(actual.Location), b.location) || actual.Properties == nil {
		return nil, hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != a.ContainerAppProvisioningStateSucceeded || value(p.RunningStatus) != a.ContainerAppRunningStatusRunning && value(p.RunningStatus) != a.ContainerAppRunningStatusReady {
		return nil, hostedexecution.ErrUnavailable
	}
	environment := value(p.ManagedEnvironmentID)
	if environment == "" {
		environment = value(p.EnvironmentID)
	}
	if !strings.EqualFold(environment, b.environment) || p.EnvironmentID != nil && !strings.EqualFold(value(p.EnvironmentID), b.environment) || p.Configuration == nil || p.Configuration.Ingress == nil || actual.Identity == nil || len(actual.Identity.UserAssignedIdentities) != len(b.identities) {
		return nil, hostedexecution.ErrOwnership
	}
	_, name := resourceParts(b.application)
	if value(p.Configuration.Ingress.Fqdn) == "" {
		return nil, hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(p.Configuration.Ingress.Fqdn), name+"."+b.domain) {
		return nil, hostedexecution.ErrOwnership
	}
	doc := document(actual)
	if doc == nil {
		return nil, hostedexecution.ErrOwnership
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
	properties["managedEnvironmentId"] = b.environment
	properties["template"] = b.templateDocument(p.Template)
	identity, _ := doc["identity"].(map[string]any)
	delete(identity, "principalId")
	delete(identity, "tenantId")
	owned := map[string]any{}
	for id := range actual.Identity.UserAssignedIdentities {
		match := ""
		for _, expected := range b.identities {
			if strings.EqualFold(id, expected) {
				match = expected
				break
			}
		}
		if match == "" || owned[match] != nil {
			return nil, hostedexecution.ErrOwnership
		}
		owned[match] = map[string]any{}
	}
	identity["userAssignedIdentities"] = owned
	config, _ := properties["configuration"].(map[string]any)
	if fingerprint(config["dapr"]) == fingerprint(map[string]any{"enabled": false}) {
		delete(config, "dapr")
	}
	for _, field := range []string{"identitySettings", "registries", "secrets"} {
		entries, _ := config[field].([]any)
		for _, entry := range entries {
			item, _ := entry.(map[string]any)
			for _, id := range b.identities {
				if strings.EqualFold(valueString(item["identity"]), id) {
					item["identity"] = id
				}
			}
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
			return nil, hostedexecution.ErrOwnership
		}
		item["name"] = strings.ToLower(valueString(item["name"]))
		if strings.EqualFold(valueString(item["certificateId"]), b.certificate) {
			item["certificateId"] = b.certificate
		}
	}
	normalizeServiceConfiguration(config)
	return doc, nil
}

func sortServiceObjects(value any, key string) {
	entries, _ := value.([]any)
	slices.SortFunc(entries, func(a, b any) int {
		left, _ := a.(map[string]any)
		right, _ := b.(map[string]any)
		return strings.Compare(valueString(left[key]), valueString(right[key]))
	})
}

func normalizeServiceConfiguration(config map[string]any) {
	for field, key := range map[string]string{"identitySettings": "identity", "registries": "server", "secrets": "name"} {
		sortServiceObjects(config[field], key)
	}
}

func expectedServiceDocument(expected a.ContainerApp, normalize func(*a.Template) map[string]any) map[string]any {
	doc := document(expected)
	properties, _ := doc["properties"].(map[string]any)
	properties["template"] = normalize(expected.Properties.Template)
	config, _ := properties["configuration"].(map[string]any)
	normalizeServiceConfiguration(config)
	return doc
}

func verifyServiceRevisions(app a.ContainerApp, revisions []a.Revision, application string, expected *a.Template, normalize func(*a.Template) map[string]any) error {
	if len(revisions) == 0 || value(app.Properties.LatestRevisionName) == "" || value(app.Properties.LatestReadyRevisionName) == "" {
		return hostedexecution.ErrUnavailable
	}
	if len(revisions) > 4 {
		return hostedexecution.ErrOwnership
	}
	if value(app.Properties.LatestRevisionName) != value(app.Properties.LatestReadyRevisionName) {
		return hostedexecution.ErrUnavailable
	}
	active := 0
	seen := map[string]bool{}
	_, appName := resourceParts(application)
	for _, revision := range revisions {
		name := value(revision.Name)
		if !strings.HasPrefix(name, appName+"--") || !endpointChild(value(revision.ID), name, application, "revisions") || !strings.EqualFold(value(revision.Type), "Microsoft.App/containerApps/revisions") || seen[strings.ToLower(name)] || revision.Properties == nil {
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
		if fingerprint(normalize(p.Template)) != fingerprint(normalize(expected)) {
			return hostedexecution.ErrOwnership
		}
	}
	if active != 1 {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
