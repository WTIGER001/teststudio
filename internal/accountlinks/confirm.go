package accountlinks

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Confirm(ctx context.Context, sessionID, id string, revision int64) (View, *http.Cookie, error) {
	var result View
	var cookie *http.Cookie
	err := s.transaction(ctx, func(bound *Service) error {
		row, op, err := bound.load(ctx, sessionID, id)
		if err != nil {
			return err
		}
		if revision < 1 || row.Revision != revision {
			return storage.ErrConflict
		}
		if err := bound.current(ctx, op); err != nil {
			return err
		}
		now := time.Now().UTC()
		if op.State != "review" || op.Identity == nil || op.Identity.Issuer != op.TargetProvider.Issuer || op.Identity.Subject == "" || op.SourceVerifiedAt.IsZero() || op.TargetVerifiedAt.IsZero() || now.Sub(op.SourceVerifiedAt) > 5*time.Minute || now.Sub(op.TargetVerifiedAt) > 5*time.Minute {
			return ErrInvalid
		}
		_, err = bound.Sessions.Directory.ExternalLink(ctx, op.Identity.Issuer, op.Identity.Subject)
		if err == nil {
			return ErrClaimed
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		link := identity.ExternalIdentity{Issuer: op.Identity.Issuer, Subject: op.Identity.Subject, UserID: op.UserID}
		change, err := storage.Put("external-identities", identity.ExternalIdentityID(link.Issuer, link.Subject), "", 0, link)
		if err != nil {
			return err
		}
		session, nextCookie, err := bound.Sessions.Rotate(ctx, sessionID)
		if err != nil {
			return err
		}
		op.State, op.SessionID, op.Revision = "complete", session.ID, row.Revision+1
		completed, err := storage.Put(bucket, op.ID, "", row.Revision, op)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(op.UserID, "", "identity.link.confirm", op.ID, nil, map[string]any{"identity": link, "reason": op.Reason})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change, completed, audit); err != nil {
			return err
		}
		result, cookie = op.View, nextCookie
		return nil
	})
	if err != nil {
		return View{}, nil, err
	}
	return result, cookie, nil
}
