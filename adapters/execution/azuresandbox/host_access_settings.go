package azuresandbox

import (
	"net/url"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type HostSecretAccess struct {
	Vault string         `json:"vault"`
	Grant AuthorityGrant `json:"grant"`
}

type HostBlobAccess struct {
	Account string         `json:"account"`
	Grant   AuthorityGrant `json:"grant"`
}

type HostAccessSettings struct {
	Database      HostSecretAccess `json:"database"`
	Configuration HostSecretAccess `json:"configuration"`
	Blob          *HostBlobAccess  `json:"blob,omitempty"`
}

func secretResource(vault, reference string) string {
	u, err := url.Parse(reference)
	if err != nil {
		return ""
	}
	return vault + u.Path
}

func (a HostBlobAccess) container(h HostSettings) string {
	return a.Account + "/blobServices/default/containers/" + h.EvidenceContainer
}

func (h HostSettings) validateAccess(s Settings) error {
	a, c := h.Access, s.Worker
	for _, entry := range []struct {
		pin       HostSecretAccess
		reference string
	}{{a.Database, h.DatabaseSecretURL}, {a.Configuration, h.ConfigurationSecretURL}} {
		u, err := url.Parse(entry.reference)
		if err != nil || !resourceID(c, entry.pin.Vault, "Microsoft.KeyVault", "vaults") || u.Hostname() != entry.pin.Vault[strings.LastIndex(entry.pin.Vault, "/")+1:]+".vault.azure.net" {
			return hostedexecution.ErrConfiguration
		}
	}
	if s.Mode == "execution" {
		if a.Blob == nil || !resourceID(c, a.Blob.Account, "Microsoft.Storage", "storageAccounts") || h.BlobServiceURL != "https://"+a.Blob.Account[strings.LastIndex(a.Blob.Account, "/")+1:]+".blob.core.windows.net/" {
			return hostedexecution.ErrConfiguration
		}
	} else if a.Blob != nil {
		return hostedexecution.ErrConfiguration
	}
	roles, assignments := map[string]bool{}, map[string]bool{}
	for _, role := range s.authorityRoles() {
		roles[strings.ToLower(role.grant.Role)], assignments[strings.ToLower(role.grant.Assignment)] = true, true
	}
	for _, id := range []string{s.Authority.RegistryAssignment, s.Placement.PullAssignment, s.Placement.Services.RelayNetwork.Registry.PullAssignment, h.Registry.PullAssignment} {
		assignments[strings.ToLower(id)] = true
	}
	for _, role := range s.hostAccessRoles() {
		grant := role.grant
		if !customRoleID(c, grant.Role) || !assignmentID(role.scope, grant.Assignment) || roles[strings.ToLower(grant.Role)] || assignments[strings.ToLower(grant.Assignment)] {
			return hostedexecution.ErrConfiguration
		}
		roles[strings.ToLower(grant.Role)], assignments[strings.ToLower(grant.Assignment)] = true, true
	}
	return nil
}

func (s Settings) hostAccessRoles() []authorityRole {
	h, a := s.Host, s.Host.Access
	name := "TestStudio " + hash(h.Application)[:12] + " "
	secretData := []string{"Microsoft.KeyVault/vaults/secrets/getSecret/action", "Microsoft.KeyVault/vaults/secrets/readMetadata/action"}
	result := []authorityRole{
		{a.Database.Grant, secretResource(a.Database.Vault, h.DatabaseSecretURL), resourceGroupID(a.Database.Vault), h.Startup.Principal, name + "database secret", nil, secretData},
		{a.Configuration.Grant, secretResource(a.Configuration.Vault, h.ConfigurationSecretURL), resourceGroupID(a.Configuration.Vault), h.Startup.Principal, name + "runtime configuration", nil, secretData},
	}
	if a.Blob != nil && h.Runtime != nil {
		result = append(result, authorityRole{a.Blob.Grant, a.Blob.container(h), resourceGroupID(a.Blob.Account), h.Runtime.Principal, name + "evidence", []string{"Microsoft.Storage/storageAccounts/blobServices/containers/read"}, []string{"Microsoft.Storage/storageAccounts/blobServices/containers/blobs/read", "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/write", "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/delete"}})
	}
	return result
}

func (s Settings) allAuthorityRoles() []authorityRole {
	return append(s.authorityRoles(), s.hostAccessRoles()...)
}

func (s Settings) hostAccessVersions() map[string]string {
	a := s.Host.Access
	result := map[string]string{a.Database.Vault: "2023-07-01", a.Configuration.Vault: "2023-07-01"}
	if a.Blob != nil {
		result[a.Blob.Account], result[a.Blob.container(s.Host)] = "2024-01-01", "2024-01-01"
	}
	return result
}
