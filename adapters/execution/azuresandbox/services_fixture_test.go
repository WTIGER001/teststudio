package azuresandbox

import (
	"encoding/json"
	"strings"
	"time"
)

func runtimeServiceSettings(c Config, p PlacementSettings) ServicesSettings {
	identity := "/subscriptions/" + c.Subscription + "/resourceGroups/studio-identities/providers/Microsoft.ManagedIdentity/userAssignedIdentities/"
	base := "/subscriptions/" + c.Subscription + "/resourceGroups/"
	pin := func(name, digit string) ServiceIdentitySettings {
		return ServiceIdentitySettings{Resource: identity + name, Client: "d" + strings.Repeat(digit, 7) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 12), Principal: "e" + strings.Repeat(digit, 7) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 12)}
	}
	return ServicesSettings{
		Relay:           RelayServiceSettings{Application: base + "studio-relay/providers/Microsoft.App/containerApps/studio-exchange", Certificate: p.Private.ExchangeEnvironment + "/certificates/studio-exchange", Subnet: base + "studio-relay/providers/Microsoft.Network/virtualNetworks/studio-relay/subnets/services", InfrastructureGroup: "studio-relay-infra", Image: c.Registry + "/studio/services/exchange@sha256:" + strings.Repeat("b", 64), Pull: pin("studio-relay-pull", "1")},
		Backend:         BackendServiceSettings{Application: base + "studio-backend/providers/Microsoft.App/containerApps/studio-backend", Environment: base + "studio-backend/providers/Microsoft.App/managedEnvironments/studio-backend", Subnet: base + "studio-backend/providers/Microsoft.Network/virtualNetworks/studio-backend/subnets/services", InfrastructureGroup: "studio-backend-infra", Origin: "https://studio-backend.backendzone.westus2.azurecontainerapps.io", Address: "10.60.2.4", Image: c.Registry + "/studio/services/server@sha256:" + strings.Repeat("c", 64), Pull: pin("studio-backend-pull", "2"), Startup: pin("studio-backend-startup", "3"), Runtime: pin("studio-backend-runtime", "4"), DatabaseSecretURL: "https://studiotest.vault.azure.net/secrets/database", BlobServiceURL: "https://studiotest.blob.core.windows.net/", EvidenceContainer: "evidence", Replicas: 1},
		BackendEndpoint: BackendEndpointSettings{Endpoint: base + "studio-relay/providers/Microsoft.Network/privateEndpoints/studio-backend", Interface: base + "studio-relay/providers/Microsoft.Network/networkInterfaces/studio-backend", Subnet: base + "studio-relay/providers/Microsoft.Network/virtualNetworks/studio-relay/subnets/backend-endpoint", SubnetPrefix: "10.60.2.0/24", RelaySubnetPrefix: "10.60.0.0/23", SecurityGroup: base + "studio-relay/providers/Microsoft.Network/networkSecurityGroups/studio-backend"},
	}
}

func (w *placementWorld) addServices() {
	w.t.Helper()
	s, v := w.s, w.s.Placement.Services
	put := func(id, kind, location string, props map[string]any) map[string]any {
		r := map[string]any{"id": id, "name": id[strings.LastIndex(id, "/")+1:], "type": kind, "location": location, "properties": props}
		w.resources[id] = r
		return r
	}
	connections := []any{map[string]any{"id": s.Placement.Private.ExchangeEnvironment + "/privateEndpointConnections/studio-exchange", "name": "studio-exchange", "type": "Microsoft.App/managedEnvironments/privateEndpointConnections", "properties": map[string]any{"provisioningState": "Succeeded", "privateEndpoint": map[string]string{"id": s.Placement.Private.ExchangeEndpoint}, "groupIds": []string{"managedEnvironment"}, "privateLinkServiceConnectionState": map[string]string{"status": "Approved", "actionsRequired": "None"}}}}
	for _, item := range []struct{ id, subnet, infrastructure, domain string }{{s.Placement.Private.ExchangeEnvironment, v.Relay.Subnet, v.Relay.InfrastructureGroup, "relayzone.westus2.azurecontainerapps.io"}, {v.Backend.Environment, v.Backend.Subnet, v.Backend.InfrastructureGroup, "backendzone.westus2.azurecontainerapps.io"}} {
		props := map[string]any{"provisioningState": "Succeeded", "vnetConfiguration": map[string]any{"internal": true, "infrastructureSubnetId": item.subnet}, "infrastructureResourceGroup": item.infrastructure, "publicNetworkAccess": "Disabled", "appLogsConfiguration": map[string]string{"destination": "none"}, "workloadProfiles": []any{map[string]any{"name": "Consumption", "workloadProfileType": "Consumption", "minimumCount": 0, "maximumCount": 0}}, "defaultDomain": item.domain}
		if item.id == s.Placement.Private.ExchangeEnvironment {
			props["privateEndpointConnections"] = connections
		}
		put(item.id, "Microsoft.App/managedEnvironments", s.Worker.Region, props)
		w.pages[item.id+"/httpRouteConfigs"] = map[string]any{"value": []any{}}
	}
	w.pages[s.Placement.Private.ExchangeEnvironment+"/privateEndpointConnections"] = map[string]any{"value": connections}
	w.pages[s.Placement.Private.ExchangeEnvironment+"/privateLinkResources"] = map[string]any{"value": []any{map[string]any{"id": s.Placement.Private.ExchangeEnvironment + "/privateLinkResources/managedEnvironment", "name": "managedEnvironment", "type": "Microsoft.App/managedEnvironments/privateLinkResources", "properties": map[string]any{"groupId": "managedEnvironment", "requiredMembers": []string{"managedEnvironment"}, "requiredZoneNames": []string{"privatelink.westus2.azurecontainerapps.io"}}}}}
	now := time.Now().UTC()
	put(v.Relay.Certificate, "Microsoft.App/managedEnvironments/certificates", s.Worker.Region, map[string]any{"provisioningState": "Succeeded", "valid": true, "issueDate": now.Add(-time.Hour), "expirationDate": now.Add(6 * time.Hour), "subjectAlternativeNames": []string{strings.TrimPrefix(s.Worker.ExchangeOrigin, "https://")}})
	defs, err := s.ServiceDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	for id, definition := range defs {
		data, _ := json.Marshal(definition)
		var value map[string]any
		if json.Unmarshal(data, &value) != nil {
			w.t.Fatal("invalid service definition")
		}
		value["id"], value["name"] = id, id[strings.LastIndex(id, "/")+1:]
		kind := "Microsoft.App/containerApps"
		if strings.HasSuffix(id, "/authConfigs/current") {
			kind += "/authConfigs"
		} else {
			props := value["properties"].(map[string]any)
			revision := value["name"].(string) + "--verified"
			props["provisioningState"], props["runningStatus"] = "Succeeded", "Running"
			props["latestRevisionName"], props["latestReadyRevisionName"] = revision, revision
			domain, identities := "relayzone.westus2.azurecontainerapps.io", []ServiceIdentitySettings{v.Relay.Pull}
			if id == v.Backend.Application {
				domain, identities = "backendzone.westus2.azurecontainerapps.io", []ServiceIdentitySettings{v.Backend.Pull, v.Backend.Startup, v.Backend.Runtime}
			}
			props["configuration"].(map[string]any)["ingress"].(map[string]any)["fqdn"] = value["name"].(string) + "." + domain
			for _, pin := range identities {
				value["identity"].(map[string]any)["userAssignedIdentities"].(map[string]any)[pin.Resource] = map[string]string{"clientId": pin.Client, "principalId": pin.Principal}
			}
			w.pages[id+"/revisions"] = map[string]any{"value": []any{map[string]any{"id": id + "/revisions/" + revision, "name": revision, "type": "Microsoft.App/containerApps/revisions", "properties": map[string]any{"active": true, "provisioningState": "Provisioned", "healthState": "Healthy", "runningState": "Running", "replicas": 1, "trafficWeight": 100, "template": props["template"]}}}}
		}
		value["type"] = kind
		w.resources[id] = value
	}
}
