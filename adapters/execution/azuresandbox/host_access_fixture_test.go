package azuresandbox

import "strings"

func runtimeHostAccess(s Settings, h HostSettings) HostAccessSettings {
	base := "/subscriptions/" + s.Worker.Subscription + "/resourceGroups/studio-data/providers/"
	vault := base + "Microsoft.KeyVault/vaults/studiotest"
	grant := func(scope, digit string) AuthorityGrant {
		return AuthorityGrant{Role: "/subscriptions/" + s.Worker.Subscription + "/providers/Microsoft.Authorization/roleDefinitions/bbbbbbb" + digit + "-bbbb-bbbb-bbbb-bbbbbbbbbbb" + digit, Assignment: scope + "/providers/Microsoft.Authorization/roleAssignments/ccccccc" + digit + "-cccc-cccc-cccc-ccccccccccc" + digit}
	}
	a := HostAccessSettings{Database: HostSecretAccess{Vault: vault, Grant: grant(secretResource(vault, h.DatabaseSecretURL), "7")}, Configuration: HostSecretAccess{Vault: vault, Grant: grant(secretResource(vault, h.ConfigurationSecretURL), "8")}}
	if s.Mode == "execution" {
		blob := HostBlobAccess{Account: base + "Microsoft.Storage/storageAccounts/studiotest"}
		blob.Grant = grant(blob.container(h), "9")
		a.Blob = &blob
	}
	return a
}

func (w *placementWorld) addHostAccess() {
	s, h := w.s, w.s.Host
	put := func(id, kind string, body any) {
		value := objectMap(body)
		value["id"], value["name"], value["type"] = id, id[strings.LastIndex(id, "/")+1:], kind
		w.resources[id] = value
	}
	for _, vault := range []string{h.Access.Database.Vault, h.Access.Configuration.Vault} {
		put(vault, "Microsoft.KeyVault/vaults", map[string]any{"location": s.Worker.Region, "properties": map[string]any{"tenantId": s.TenantID, "vaultUri": "https://" + vault[strings.LastIndex(vault, "/")+1:] + ".vault.azure.net/", "enableRbacAuthorization": true, "publicNetworkAccess": "Enabled"}})
	}
	if h.Access.Blob != nil {
		blob := h.Access.Blob
		put(blob.Account, "Microsoft.Storage/storageAccounts", map[string]any{"location": s.Worker.Region, "kind": "StorageV2", "properties": map[string]any{"provisioningState": "Succeeded", "supportsHttpsTrafficOnly": true, "minimumTlsVersion": "TLS1_2", "allowBlobPublicAccess": false, "allowSharedKeyAccess": false, "publicNetworkAccess": "Enabled", "primaryEndpoints": map[string]string{"blob": h.BlobServiceURL}}})
		put(blob.container(h), "Microsoft.Storage/storageAccounts/blobServices/containers", map[string]any{"properties": map[string]any{"publicAccess": "None", "deleted": false}})
	}
}
