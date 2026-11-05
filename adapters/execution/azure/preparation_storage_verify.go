package azureworker

import (
	"strings"

	s "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyPreparationStorage(settings PreparationSettings, account s.Account, share s.FileShare) error {
	c, n, h := settings.Worker, settings.Network, settings.Host
	if !perimeterIdentity(value(account.ID), value(account.Name), value(account.Type), n.StorageAccount, "Microsoft.Storage/storageAccounts", value(account.Location), c.Region) || account.ExtendedLocation != nil {
		return hostedexecution.ErrOwnership
	}
	p := account.Properties
	if p == nil || p.NetworkRuleSet == nil || p.PrimaryEndpoints == nil || p.PrimaryEndpoints.File == nil || p.ProvisioningState == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(p.ProvisioningState) != s.ProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if value(account.Kind) != s.KindStorageV2 && value(account.Kind) != s.KindFileStorage || value(p.PublicNetworkAccess) != s.PublicNetworkAccessDisabled || value(p.NetworkRuleSet.DefaultAction) != s.DefaultActionDeny || value(p.NetworkRuleSet.Bypass) != s.BypassNone || len(p.NetworkRuleSet.IPRules)+len(p.NetworkRuleSet.VirtualNetworkRules)+len(p.NetworkRuleSet.ResourceAccessRules) != 0 || value(p.PrimaryEndpoints.File) != "https://"+h.JournalAccount+".file.core.windows.net/" || !value(p.EnableHTTPSTrafficOnly) || value(p.MinimumTLSVersion) != s.MinimumTLSVersionTLS12 || value(p.AllowBlobPublicAccess) || p.AllowSharedKeyAccess != nil && !value(p.AllowSharedKeyAccess) || value(p.IsSftpEnabled) || value(p.IsLocalUserEnabled) || value(p.EnableNfsV3) {
		return hostedexecution.ErrOwnership
	}
	if len(p.PrivateEndpointConnections) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if len(p.PrivateEndpointConnections) != 1 || p.PrivateEndpointConnections[0] == nil || p.PrivateEndpointConnections[0].Properties == nil {
		return hostedexecution.ErrOwnership
	}
	connection := p.PrivateEndpointConnections[0]
	if !endpointChild(value(connection.ID), value(connection.Name), n.StorageAccount, "privateEndpointConnections") || !strings.EqualFold(value(connection.Type), "Microsoft.Storage/storageAccounts/privateEndpointConnections") || connection.Properties.PrivateEndpoint == nil || !strings.EqualFold(value(connection.Properties.PrivateEndpoint.ID), n.FileEndpoint) {
		return hostedexecution.ErrOwnership
	}
	cp := connection.Properties
	if cp.PrivateLinkServiceConnectionState == nil || value(cp.ProvisioningState) != s.PrivateEndpointConnectionProvisioningStateSucceeded {
		return hostedexecution.ErrUnavailable
	}
	if value(cp.PrivateLinkServiceConnectionState.Status) != s.PrivateEndpointServiceConnectionStatusApproved {
		if value(cp.PrivateLinkServiceConnectionState.Status) == "" || value(cp.PrivateLinkServiceConnectionState.Status) == s.PrivateEndpointServiceConnectionStatusPending {
			return hostedexecution.ErrUnavailable
		}
		return hostedexecution.ErrOwnership
	}
	if action := value(cp.PrivateLinkServiceConnectionState.ActionRequired); action != "" && action != "None" {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(value(share.ID), n.share(settings)) || value(share.Name) != h.JournalShare || !strings.EqualFold(value(share.Type), "Microsoft.Storage/storageAccounts/fileServices/shares") {
		return hostedexecution.ErrOwnership
	}
	sp := share.FileShareProperties
	if sp == nil || sp.EnabledProtocols == nil || sp.ShareQuota == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(sp.EnabledProtocols) != s.EnabledProtocolsSMB || value(sp.ShareQuota) < 1 || value(sp.Deleted) || sp.SnapshotTime != nil {
		return hostedexecution.ErrOwnership
	}
	return nil
}
