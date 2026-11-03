package access

import (
	"context"
	"errors"
	"slices"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) SaveProvider(ctx context.Context, actor string, input ProviderInput, revision int64) (ProviderRecord, error) {
	if err := reason(input.Reason); err != nil {
		return ProviderRecord{}, err
	}
	value, err := normalizeProvider(input)
	if err != nil {
		return ProviderRecord{}, err
	}
	var result ProviderRecord
	err = s.Auth.WithPolicyTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth, LocalUser: s.LocalUser, LocalAccounts: s.LocalAccounts, SessionIdleTimeout: s.SessionIdleTimeout, ProviderSecretRefs: s.ProviderSecretRefs, AuthOrigin: s.AuthOrigin}
		if err := bound.providerAuthority(ctx, actor, value.ID, true); err != nil {
			return err
		}
		prior, row, err := bound.loadProvider(ctx, value.ID)
		if err != nil && (!errors.Is(err, storage.ErrNotFound) || row.Revision > 0) {
			return err
		}
		if row.Revision != revision {
			return storage.ErrConflict
		}
		if row.Revision > 0 {
			if err := bound.providerMappingAuthority(ctx, actor, prior, false); err != nil {
				return err
			}
		}
		if value.SecretRef != "" && !slices.Contains(bound.ProviderSecretRefs, value.SecretRef) && !(value.Disabled && row.Revision > 0 && value.SecretRef == prior.SecretRef) {
			return errors.New("select an operator-approved OIDC credential reference")
		}
		if err := bound.providerMappingAuthority(ctx, actor, value, true); err != nil {
			return err
		}
		hadAdmin, err := bound.hasUsableAdministrator(ctx)
		if err != nil {
			return err
		}
		change, err := storage.Put("providers", value.ID, "", revision, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(actor, "", "identity.provider.save", value.ID, prior, map[string]any{"value": value, "reason": input.Reason, "earlierAuthenticationInvalidated": true})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change, audit); err != nil {
			return err
		}
		if hadAdmin {
			remains, err := bound.hasUsableAdministrator(ctx)
			if err != nil {
				return err
			}
			if !remains {
				return errors.New("the change would remove the last usable installation administrator")
			}
		}
		row, err = bound.Store.Get(ctx, "providers", value.ID)
		if err != nil {
			return err
		}
		result, err = bound.providerRecord(ctx, actor, value, row)
		return err
	})
	return result, err
}
