package azureworker

import (
	"slices"
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
)

const workloadIdentityItems = 64

func inventoryWithIdentities(placement WorkloadPlacement, identity *a.ManagedServiceIdentity) WorkloadPlacement {
	placement.IdentityKnown, placement.Identities, placement.SystemPrincipalID = false, nil, ""
	if identity == nil || identity.Type == nil || len(identity.UserAssignedIdentities) > workloadIdentityItems {
		return placement
	}
	kind := value(identity.Type)
	system := kind == a.ManagedServiceIdentityTypeSystemAssigned || kind == a.ManagedServiceIdentityTypeSystemAssignedUserAssigned
	user := kind == a.ManagedServiceIdentityTypeUserAssigned || kind == a.ManagedServiceIdentityTypeSystemAssignedUserAssigned
	if kind != a.ManagedServiceIdentityTypeNone && !system && !user {
		return placement
	}
	principal, tenant := value(identity.PrincipalID), value(identity.TenantID)
	if system && !observerGUID(strings.ToLower(principal)) || !system && (principal != "" || tenant != "") || tenant != "" && !observerGUID(strings.ToLower(tenant)) || user && len(identity.UserAssignedIdentities) == 0 || !user && len(identity.UserAssignedIdentities) != 0 {
		return placement
	}
	identities := make([]WorkloadIdentity, 0, len(identity.UserAssignedIdentities))
	seen := map[string]bool{}
	for id, metadata := range identity.UserAssignedIdentities {
		key := inventoryIdentityKey(id)
		if !workloadIdentityReference(id) || metadata == nil || seen[key] {
			return placement
		}
		client, principal := value(metadata.ClientID), value(metadata.PrincipalID)
		if client != "" && !observerGUID(strings.ToLower(client)) || principal != "" && !observerGUID(strings.ToLower(principal)) || client != "" && strings.EqualFold(client, principal) {
			return placement
		}
		identities = append(identities, WorkloadIdentity{ResourceID: id, ClientID: client, PrincipalID: principal})
		seen[key] = true
	}
	slices.SortFunc(identities, func(a, b WorkloadIdentity) int {
		return strings.Compare(inventoryIdentityKey(a.ResourceID), inventoryIdentityKey(b.ResourceID))
	})
	placement.IdentityKnown, placement.Identities, placement.SystemPrincipalID = true, identities, principal
	return placement
}

func workloadIdentityReference(id string) bool {
	if len(id) > 2048 {
		return false
	}
	parts := strings.Split(id, "/")
	return len(parts) == 9 && parts[0] == "" && strings.EqualFold(parts[1], "subscriptions") && guid.MatchString(strings.ToLower(parts[2])) && strings.EqualFold(parts[3], "resourceGroups") && inventoryGroup(parts[4]) && strings.EqualFold(parts[5], "providers") && strings.EqualFold(parts[6], "Microsoft.ManagedIdentity") && strings.EqualFold(parts[7], "userAssignedIdentities") && endpointChildName.MatchString(parts[8])
}
