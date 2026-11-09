package access

import (
	"context"
	"errors"
	"slices"
	"time"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type ProviderInput struct {
	HiddenFromSignIn     bool              `json:"hiddenFromSignIn,omitempty"`
	LoginBypassEnabled   bool              `json:"loginBypassEnabled,omitempty"`
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Issuer               string            `json:"issuer"`
	ClientID             string            `json:"clientId"`
	SecretRef            string            `json:"secretRef,omitempty"`
	Scopes               []string          `json:"scopes"`
	Disabled             *bool             `json:"disabled"`
	AutoProvision        bool              `json:"autoProvision"`
	GroupClaim           string            `json:"groupClaim,omitempty"`
	GroupMappings        map[string]string `json:"groupMappings"`
	LogoutEndpoint       string            `json:"logoutEndpoint,omitempty"`
	MembershipTTLSeconds int               `json:"membershipTtlSeconds"`
	Reason               string            `json:"reason"`
}

type ProviderView struct {
	authoidc.Provider
	CanManage         bool   `json:"canManage"`
	LogoutCallbackURL string `json:"logoutCallbackUrl"`
	CallbackURL       string `json:"callbackUrl"`
}
type ProviderRecord struct {
	ID       string       `json:"id"`
	Revision int64        `json:"revision"`
	Updated  time.Time    `json:"updatedAt"`
	Data     ProviderView `json:"data"`
}
type ProviderPage struct {
	Items          []ProviderRecord `json:"items"`
	Cursor         string           `json:"cursor,omitempty"`
	Partial        bool             `json:"partial"`
	CanCreate      bool             `json:"canCreate"`
	CredentialRefs []string         `json:"credentialRefs"`
}

func (s *Service) providerAuthority(ctx context.Context, actor, id string, manage bool) error {
	for _, object := range []string{"", "provider/" + id} {
		if err := s.Auth.Check(ctx, actor, "", object, "providers.read"); err != nil {
			return err
		}
		if manage {
			if err := s.Auth.Check(ctx, actor, "", object, "providers.manage"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) loadProvider(ctx context.Context, id string) (authoidc.Provider, storage.Record, error) {
	if !domain.ValidID(id) {
		return authoidc.Provider{}, storage.Record{}, errors.New("a valid provider identifier is required")
	}
	row, err := s.Store.Get(ctx, "providers", id)
	if err != nil {
		return authoidc.Provider{}, row, err
	}
	provider, err := storage.Decode[authoidc.Provider](row)
	if err != nil {
		return provider, row, err
	}
	if row.Project != "" || provider.ID != id {
		return provider, row, storage.ErrNotFound
	}
	return provider, row, nil
}

func (s *Service) providerRecord(ctx context.Context, actor string, provider authoidc.Provider, row storage.Record) (ProviderRecord, error) {
	if provider.Scopes == nil {
		provider.Scopes = []string{}
	}
	if provider.GroupMappings == nil {
		provider.GroupMappings = map[string]string{}
	}
	canManage, err := roleCapability(func() error {
		if err := s.providerAuthority(ctx, actor, provider.ID, true); err != nil {
			return err
		}
		return s.providerMappingAuthority(ctx, actor, provider, false)
	})
	return ProviderRecord{ID: row.ID, Revision: row.Revision, Updated: row.Updated, Data: ProviderView{Provider: provider, CanManage: canManage, CallbackURL: s.AuthOrigin + "/auth/callback/" + provider.ID, LogoutCallbackURL: s.AuthOrigin + "/auth/logout/callback/" + provider.ID}}, err
}

func (s *Service) ReadProvider(ctx context.Context, actor, id string) (ProviderRecord, error) {
	if err := s.providerAuthority(ctx, actor, id, false); err != nil {
		return ProviderRecord{}, err
	}
	provider, row, err := s.loadProvider(ctx, id)
	if err != nil {
		return ProviderRecord{}, err
	}
	return s.providerRecord(ctx, actor, provider, row)
}

func (s *Service) Providers(ctx context.Context, actor, cursor string, limit int) (ProviderPage, error) {
	if err := s.Auth.Check(ctx, actor, "", "", "providers.read"); err != nil {
		return ProviderPage{}, err
	}
	page, err := s.page(ctx, "providers", "", queryScope(actor, "providers", "", false), cursor, limit)
	if err != nil {
		return ProviderPage{}, err
	}
	result := ProviderPage{Items: []ProviderRecord{}, Cursor: page.Cursor, Partial: page.Cursor != "", CredentialRefs: slices.Clone(s.ProviderSecretRefs)}
	if result.CredentialRefs == nil {
		result.CredentialRefs = []string{}
	}
	result.CanCreate, err = roleCapability(func() error { return s.Auth.Check(ctx, actor, "", "", "providers.manage") })
	if err != nil {
		return result, err
	}
	for _, row := range page.Items {
		if err := s.providerAuthority(ctx, actor, row.ID, false); errors.Is(err, authorization.ErrForbidden) {
			continue
		} else if err != nil {
			return result, err
		}
		provider, err := storage.Decode[authoidc.Provider](row)
		if err != nil {
			return result, err
		}
		if provider.ID != row.ID || row.Project != "" {
			continue
		}
		value, err := s.providerRecord(ctx, actor, provider, row)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, value)
	}
	return result, nil
}
