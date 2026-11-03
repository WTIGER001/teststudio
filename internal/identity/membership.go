package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (d *Directory) MembershipCurrent(ctx context.Context, membership Membership) (bool, error) {
	if membership.ExpiresAt != nil && !time.Now().Before(*membership.ExpiresAt) {
		return false, nil
	}
	if !strings.HasPrefix(membership.Source, "oidc:") {
		return membership.ProviderRevision == 0, nil
	}
	if membership.ProviderRevision < 1 {
		return false, nil
	}
	_, err := d.providerAt(ctx, strings.TrimPrefix(membership.Source, "oidc:"), membership.ProviderRevision)
	if errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrConflict) {
		return false, nil
	}
	return err == nil, err
}
