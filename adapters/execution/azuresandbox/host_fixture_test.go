package azuresandbox

import "strings"

func runtimeHostSettings(s Settings) HostSettings {
	base := "/subscriptions/" + s.Worker.Subscription + "/resourceGroups/studio-control/providers/"
	identities := "/subscriptions/" + s.Worker.Subscription + "/resourceGroups/studio-identities/providers/Microsoft.ManagedIdentity/userAssignedIdentities/"
	pin := func(name, digit string) ServiceIdentitySettings {
		return ServiceIdentitySettings{Resource: identities + name, Client: "d" + strings.Repeat(digit, 7) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 12), Principal: "e" + strings.Repeat(digit, 7) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 4) + "-" + strings.Repeat(digit, 12)}
	}
	h := HostSettings{DatabaseHost: "studiotest.postgres.database.azure.com", Application: base + "Microsoft.App/containerApps/studio-controller", Environment: base + "Microsoft.App/managedEnvironments/studio-control", Subnet: base + "Microsoft.Network/virtualNetworks/studio-control/subnets/services", InfrastructureGroup: "studio-control-infra", Image: s.Worker.Registry + "/studio/services/server@sha256:" + strings.Repeat("d", 64), Pull: pin("studio-controller-pull", "5"), Startup: pin("studio-controller-startup", "6"), DatabaseSecretURL: "https://studiotest.vault.azure.net/secrets/database", ConfigurationSecretURL: "https://studiotest.vault.azure.net/secrets/sandbox", Replicas: 1}
	network := base + "Microsoft.Network/"
	h.SubnetPrefix = "10.72.0.0/23"
	h.Network = HostNetworkSettings{Topology: TopologySettings{Prefix: "10.72.0.0/16", Firewall: network + "azureFirewalls/controller", Policy: network + "firewallPolicies/controller", PublicIP: network + "publicIPAddresses/controller", FirewallPrefix: "10.72.4.0/26"}, FirewallIP: "10.72.4.4", SecurityGroup: network + "networkSecurityGroups/controller", RouteTable: network + "routeTables/controller", Resolver: ResolverSettings{Policy: network + "dnsResolverPolicies/controller", AllowList: network + "dnsResolverDomainLists/controller-allow", BlockList: network + "dnsResolverDomainLists/controller-block"}}
	h.Registry = RelayRegistrySettings{Endpoint: network + "privateEndpoints/controller-registry", Interface: network + "networkInterfaces/controller-registry", Subnet: strings.TrimSuffix(h.Subnet, "services") + "registry", SubnetPrefix: "10.72.3.0/24", SecurityGroup: network + "networkSecurityGroups/controller-registry", LoginIP: "10.72.3.4", DataIP: "10.72.3.5", Zone: network + "privateDnsZones/privatelink.azurecr.io", PullAssignment: s.Placement.Registry + "/providers/Microsoft.Authorization/roleAssignments/97777777-7777-7777-7777-777777777777", Connection: s.Placement.Registry + "/privateEndpointConnections/controller"}
	if s.Mode == "execution" {
		runtime := pin("studio-controller-runtime", "7")
		h.Runtime = &runtime
		h.BlobServiceURL = "https://studiotest.blob.core.windows.net/"
		h.EvidenceContainer = "evidence"
		h.PublicOrigin = "https://studio.teststudio.example"
	}
	return h
}

func (w *placementWorld) addHost() {
	w.t.Helper()
	s, h := w.s, w.s.Host
	defs, err := s.HostDefinitions()
	if err != nil {
		w.t.Fatal(err)
	}
	for id, body := range defs {
		value := objectMap(body)
		value["id"], value["name"], value["type"] = id, id[strings.LastIndex(id, "/")+1:], "Microsoft.App/containerApps"
		if strings.HasSuffix(id, "/authConfigs/current") {
			value["type"] = "Microsoft.App/containerApps/authConfigs"
		} else {
			props := value["properties"].(map[string]any)
			revision := "studio-controller--verified"
			props["provisioningState"], props["runningStatus"] = "Succeeded", "Running"
			props["latestRevisionName"], props["latestReadyRevisionName"] = revision, revision
			for resource, pin := range s.hostWorkloadBinding().Identities {
				value["identity"].(map[string]any)["userAssignedIdentities"].(map[string]any)[resource] = map[string]string{"clientId": pin.Client, "principalId": pin.Principal}
			}
			w.pages[id+"/revisions"] = map[string]any{"value": []any{map[string]any{"id": id + "/revisions/" + revision, "name": revision, "type": "Microsoft.App/containerApps/revisions", "properties": map[string]any{"active": true, "provisioningState": "Provisioned", "healthState": "Healthy", "runningState": "Running", "replicas": 1, "trafficWeight": 0, "template": props["template"]}}}}
		}
		w.resources[id] = value
	}
	w.resources[h.Environment] = map[string]any{"id": h.Environment, "name": "studio-control", "type": "Microsoft.App/managedEnvironments", "location": s.Worker.Region, "properties": map[string]any{"provisioningState": "Succeeded", "vnetConfiguration": map[string]any{"internal": true, "infrastructureSubnetId": h.Subnet}, "infrastructureResourceGroup": h.InfrastructureGroup, "publicNetworkAccess": "Disabled", "appLogsConfiguration": map[string]string{"destination": "none"}, "workloadProfiles": []any{map[string]any{"name": "Consumption", "workloadProfileType": "Consumption", "minimumCount": 0, "maximumCount": 0}}, "defaultDomain": "controlzone.westus2.azurecontainerapps.io"}}
	w.pages[h.Environment+"/httpRouteConfigs"] = map[string]any{"value": []any{}}
	w.pages[h.Environment+"/privateEndpointConnections"] = map[string]any{"value": []any{}}
}
