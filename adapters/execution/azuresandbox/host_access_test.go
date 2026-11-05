package azuresandbox

import (
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxHostAccessBindsModeSpecificRolesAndResourceMetadata(t *testing.T) {
	for _, mode := range []string{"execution", "discovery"} {
		s := runtimeSettings(mode)
		w := newPlacementWorld(t, s)
		reader, err := NewSDKAuthority(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		expected, roles := 20, 2
		if mode == "execution" {
			expected, roles = 24, 3
		}
		if err := reader.Check(t.Context(), s.Worker); err != nil || len(w.calls) != expected {
			t.Fatal("selected dependency authority rejected", err, len(w.calls))
		}
		if len(s.hostAccessRoles()) != roles {
			t.Fatal("mode-specific dependency roles changed")
		}
		for _, role := range s.hostAccessRoles() {
			if strings.Contains(role.scope, "/secrets/") {
				if role.principal != s.Host.Startup.Principal || len(role.actions) != 0 || len(role.data) != 2 {
					t.Fatal("secret role gained runtime or management authority")
				}
				if w.calls[role.scope] != 0 {
					t.Fatal("authority reader retrieved a secret")
				}
			} else if role.scope != s.Host.Access.Blob.container(s.Host) || role.principal != s.Host.Runtime.Principal || len(role.actions) != 1 || len(role.data) != 3 {
				t.Fatal("evidence role escaped its container")
			}
		}
		if s.Host.Access.Blob != nil {
			s.Host.Access.Blob.Account += "-changed"
			s.Host.Runtime.Principal = "changed"
			reader.now = func() time.Time { return time.Now().Add(metadataRefresh) }
			if err := reader.Check(t.Context(), s.Worker); err != nil {
				t.Fatal("caller mutation changed admitted dependency pins", err)
			}
		}
	}
	for _, mode := range []string{"missing", "foreign-vault", "vault-name", "role-alias", "assignment-scope", "blob-account", "discovery-blob"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			switch mode {
			case "missing":
				s.Host.Access = HostAccessSettings{}
			case "foreign-vault":
				s.Host.Access.Database.Vault = strings.Replace(s.Host.Access.Database.Vault, s.Worker.Subscription, "foreign", 1)
			case "vault-name":
				s.Host.Access.Database.Vault += "-other"
			case "role-alias":
				s.Host.Access.Database.Grant.Role = s.Authority.MetadataGrant.Role
			case "assignment-scope":
				s.Host.Access.Database.Grant.Assignment = s.Host.Access.Configuration.Grant.Assignment
			case "blob-account":
				s.Host.Access.Blob.Account += "other"
			case "discovery-blob":
				s = runtimeSettings("discovery")
				s.Host.Access.Blob = &HostBlobAccess{}
			}
			s.Host.Profile = s.HostProfile()
			if _, err := NewSDKAuthority(s, &runtimeCredential{}, nil); err != hostedexecution.ErrConfiguration {
				t.Fatal("unsafe dependency settings admitted", err)
			}
		})
	}
}

func TestSandboxHostAccessRejectsResourceScopeAndPermissionDrift(t *testing.T) {
	for _, mode := range []string{"tenant", "vault-uri", "rbac", "vault-access", "secret-principal", "secret-scope", "secret-write", "secret-management", "secret-exclusion", "delegation", "blob-principal", "blob-write-role", "blob-endpoint", "shared-key", "anonymous", "tls", "https", "container-public", "container-deleted"} {
		t.Run(mode, func(t *testing.T) {
			s := runtimeSettings()
			w := newPlacementWorld(t, s)
			a := s.Host.Access
			props := func(id string) map[string]any { return w.resources[id]["properties"].(map[string]any) }
			switch mode {
			case "tenant":
				props(a.Database.Vault)["tenantId"] = "foreign"
			case "vault-uri":
				props(a.Database.Vault)["vaultUri"] = "https://foreign.vault.azure.net/"
			case "rbac":
				props(a.Database.Vault)["enableRbacAuthorization"] = false
			case "vault-access":
				props(a.Database.Vault)["publicNetworkAccess"] = "Disabled"
			case "secret-principal":
				props(a.Database.Grant.Assignment)["principalId"] = s.Authority.Control.PrincipalID
			case "secret-scope":
				props(a.Database.Grant.Assignment)["scope"] = a.Database.Vault
			case "secret-write":
				rolePermission(w, a.Database.Grant.Role)["dataActions"] = []string{"*"}
			case "secret-management":
				rolePermission(w, a.Database.Grant.Role)["actions"] = []string{"Microsoft.KeyVault/vaults/write"}
			case "secret-exclusion":
				rolePermission(w, a.Database.Grant.Role)["notDataActions"] = []string{"Microsoft.KeyVault/vaults/secrets/getSecret/action"}
			case "delegation":
				props(a.Database.Grant.Assignment)["delegatedManagedIdentityResourceId"] = s.Authority.Control.Resource
			case "blob-principal":
				props(a.Blob.Grant.Assignment)["principalId"] = s.Host.Startup.Principal
			case "blob-write-role":
				rolePermission(w, a.Blob.Grant.Role)["actions"] = []string{"Microsoft.Storage/storageAccounts/listKeys/action"}
			case "blob-endpoint":
				props(a.Blob.Account)["primaryEndpoints"].(map[string]any)["blob"] = "https://foreign.blob.core.windows.net/"
			case "shared-key":
				props(a.Blob.Account)["allowSharedKeyAccess"] = true
			case "anonymous":
				props(a.Blob.Account)["allowBlobPublicAccess"] = true
			case "tls":
				props(a.Blob.Account)["minimumTlsVersion"] = "TLS1_0"
			case "https":
				props(a.Blob.Account)["supportsHttpsTrafficOnly"] = false
			case "container-public":
				props(a.Blob.container(s.Host))["publicAccess"] = "Blob"
			case "container-deleted":
				props(a.Blob.container(s.Host))["deleted"] = true
			}
			reader, err := NewSDKAuthority(s, &runtimeCredential{token: "placement-metadata-canary"}, w)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if err := reader.Check(t.Context(), s.Worker); err != hostedexecution.ErrOwnership {
				t.Fatal("unsafe dependency authority admitted", err)
			}
		})
	}
}

func TestSandboxRuntimeHostAccessExpiredEvidenceBlocksAllAdmissionsAndCloses(t *testing.T) {
	f := newControllerFixture(t, "go", "")
	s := runtimeSettings("discovery")
	w := newPlacementWorld(t, s)
	p := &placementProtection{}
	runtime, err := OpenRuntime(RuntimeOptions{Settings: s, Exchange: f.exchange, Admission: p, ImageAdmission: p, ARMTransport: w, Credentials: func(string) (azcore.TokenCredential, error) {
		return &runtimeCredential{token: "placement-metadata-canary"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != nil {
		t.Fatal(err)
	}
	rolePermission(w, s.Host.Access.Configuration.Grant.Role)["dataActions"] = []string{"*"}
	runtime.authority.now = func() time.Time { return time.Now().Add(metadataRefresh) }
	if err := runtime.Provider.images.admission.BeforeImport(t.Context(), s.Worker, s.PullIdentity); err != hostedexecution.ErrOwnership || p.imports.Load() != 0 {
		t.Fatal("dependency drift reached import", err)
	}
	if err := runtime.Provider.controller.admission.BeforeCreate(t.Context(), s.Worker); err != hostedexecution.ErrOwnership || p.creates.Load() != 1 {
		t.Fatal("dependency drift reached creation", err)
	}
	if err := runtime.Provider.controller.admission.VerifyInstance(t.Context(), s.Worker, Snapshot{ID: "owned", State: "Running", DefinitionHash: hash("owned")}); err != hostedexecution.ErrOwnership || p.verifies.Load() != 0 {
		t.Fatal("failed authority observation retained earlier evidence", err)
	}
	runtime.Close()
	if err := runtime.authority.Check(t.Context(), s.Worker); err != hostedexecution.ErrUnavailable {
		t.Fatal("closed authority reader retained safe evidence", err)
	}
}
