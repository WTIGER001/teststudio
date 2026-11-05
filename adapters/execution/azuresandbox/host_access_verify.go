package azuresandbox

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (s Settings) verifyHostAccessResources(resources map[string]placementResource) error {
	a := s.Host.Access
	for _, vault := range []string{a.Database.Vault, a.Configuration.Vault} {
		actual := resources[vault]
		if !namedResource(actual, vault, "Microsoft.KeyVault/vaults") || !strings.EqualFold(actual.Location, s.Worker.Region) {
			return hostedexecution.ErrOwnership
		}
		var p struct {
			Tenant string `json:"tenantId"`
			URI    string `json:"vaultUri"`
			RBAC   *bool  `json:"enableRbacAuthorization"`
			Public string `json:"publicNetworkAccess"`
		}
		if err := properties(actual, &p); err != nil {
			return err
		}
		if p.Tenant == "" || p.URI == "" || p.RBAC == nil || p.Public == "" {
			return hostedexecution.ErrUnavailable
		}
		if !strings.EqualFold(p.Tenant, s.TenantID) || p.URI != "https://"+vault[strings.LastIndex(vault, "/")+1:]+".vault.azure.net/" || !*p.RBAC || p.Public != "Enabled" {
			return hostedexecution.ErrOwnership
		}
	}
	if a.Blob == nil {
		return nil
	}
	account := resources[a.Blob.Account]
	if !namedResource(account, a.Blob.Account, "Microsoft.Storage/storageAccounts") || !strings.EqualFold(account.Location, s.Worker.Region) {
		return hostedexecution.ErrOwnership
	}
	var p struct {
		State     string `json:"provisioningState"`
		HTTPS     *bool  `json:"supportsHttpsTrafficOnly"`
		TLS       string `json:"minimumTlsVersion"`
		Anonymous *bool  `json:"allowBlobPublicAccess"`
		Keys      *bool  `json:"allowSharedKeyAccess"`
		Public    string `json:"publicNetworkAccess"`
		Endpoints *struct {
			Blob string `json:"blob"`
		} `json:"primaryEndpoints"`
	}
	if err := properties(account, &p); err != nil {
		return err
	}
	if p.State != "Succeeded" || p.HTTPS == nil || p.Anonymous == nil || p.Keys == nil || p.Endpoints == nil {
		return hostedexecution.ErrUnavailable
	}
	if !*p.HTTPS || *p.Anonymous || *p.Keys || p.TLS != "TLS1_2" || p.Public != "Enabled" || p.Endpoints.Blob != s.Host.BlobServiceURL {
		return hostedexecution.ErrOwnership
	}
	container := a.Blob.container(s.Host)
	if !namedResource(resources[container], container, "Microsoft.Storage/storageAccounts/blobServices/containers") {
		return hostedexecution.ErrOwnership
	}
	var cp struct {
		Public  string `json:"publicAccess"`
		Deleted bool   `json:"deleted"`
	}
	if err := properties(resources[container], &cp); err != nil {
		return err
	}
	if cp.Public != "None" || cp.Deleted {
		return hostedexecution.ErrOwnership
	}
	return nil
}
