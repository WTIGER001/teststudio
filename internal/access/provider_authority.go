package access

import (
	"context"
	"errors"
	"slices"
	"strings"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) providerMappingAuthority(ctx context.Context, actor string, provider authoidc.Provider, validate bool) error {
	groups := []string{}
	for _, id := range provider.GroupMappings {
		if !slices.Contains(groups, id) {
			groups = append(groups, id)
		}
	}
	slices.Sort(groups)
	for _, id := range groups {
		group, _, err := s.group(ctx, actor, id, "groups.read")
		if errors.Is(err, storage.ErrNotFound) && !validate {
			continue
		}
		if err != nil {
			return err
		}
		for _, permission := range []string{"groups.manage", "members.manage"} {
			if err := s.Auth.Check(ctx, actor, group.Project, "group/"+id, permission); err != nil {
				return err
			}
		}
		if !validate {
			continue
		}
		if group.ManagedBy != "oidc:"+provider.ID {
			return errors.New("mapping destinations must be groups owned by this provider")
		}
		if !provider.Disabled && group.Archived {
			return errors.New("restore mapped groups before enabling this provider")
		}
		if group.Project != "" {
			row, err := s.Store.Get(ctx, "projects", group.Project)
			if err != nil {
				return err
			}
			project, err := storage.Decode[struct {
				ID       string `json:"id"`
				Archived bool   `json:"archived"`
			}](row)
			if err != nil {
				return err
			}
			if row.Project != group.Project || project.ID != group.Project {
				return storage.ErrNotFound
			}
			if !provider.Disabled && project.Archived {
				return errors.New("restore mapped projects before enabling this provider")
			}
		}
	}
	return nil
}

func normalizeProvider(input ProviderInput) (authoidc.Provider, error) {
	if input.Disabled == nil {
		return authoidc.Provider{}, errors.New("provider state must include an explicit disabled value")
	}
	provider := authoidc.Provider{ID: input.ID, Name: strings.TrimSpace(input.Name), Issuer: strings.TrimSpace(input.Issuer), ClientID: strings.TrimSpace(input.ClientID), SecretRef: input.SecretRef, Scopes: slices.Clone(input.Scopes), Disabled: *input.Disabled, AutoProvision: input.AutoProvision, GroupClaim: input.GroupClaim, GroupMappings: input.GroupMappings, MembershipTTLSeconds: input.MembershipTTLSeconds, LogoutEndpoint: strings.TrimSpace(input.LogoutEndpoint)}
	if err := authoidc.ValidateProvider(provider); err != nil {
		return provider, err
	}
	if provider.Scopes == nil {
		provider.Scopes = []string{}
	}
	if provider.GroupMappings == nil {
		provider.GroupMappings = map[string]string{}
	}
	return provider, nil
}

func (s *Service) providerGroupOwner(ctx context.Context, actor, source string) (storage.Record, error) {
	id := strings.TrimPrefix(source, "oidc:")
	if !strings.HasPrefix(source, "oidc:") || !domain.ValidID(id) {
		return storage.Record{}, errors.New("external groups require oidc:PROVIDER ownership")
	}
	if err := s.providerAuthority(ctx, actor, id, true); err != nil {
		return storage.Record{}, err
	}
	_, row, err := s.loadProvider(ctx, id)
	return row, err
}
