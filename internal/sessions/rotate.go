package sessions

import (
	"context"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Rotate(ctx context.Context, id string) (Session, *http.Cookie, error) {
	version, err := s.Directory.Revision(ctx)
	if err != nil {
		return Session{}, nil, err
	}
	current, err := s.Validate(ctx, id)
	if err != nil {
		return Session{}, nil, err
	}
	row, err := s.Store.Get(ctx, "users", current.UserID)
	if err != nil {
		return Session{}, nil, err
	}
	u, err := storage.Decode[identity.User](row)
	if err != nil {
		return Session{}, nil, err
	}
	if u.Disabled || u.SessionEpoch != current.UserEpoch {
		return Session{}, nil, ErrUnauthenticated
	}
	u.SessionEpoch++
	now := time.Now().UTC()
	token := Random()
	current.ID, current.CSRF, current.UserEpoch = Digest(token), Random(), u.SessionEpoch
	current.CreatedAt, current.LastSeen = now, now
	profile, err := storage.Put("users", u.ID, "", row.Revision, u)
	if err != nil {
		return Session{}, nil, err
	}
	next, err := storage.Put("sessions", current.ID, "", 0, current)
	if err != nil {
		return Session{}, nil, err
	}
	prior, err := s.Store.Get(ctx, "sessions", id)
	if err != nil {
		return Session{}, nil, err
	}
	if err := s.Directory.CommitAt(ctx, version, profile, next, storage.Change{Record: prior, Delete: true}); err != nil {
		return Session{}, nil, err
	}
	cookie := &http.Cookie{Name: "teststudio_session", Value: token, Path: "/", HttpOnly: true, Secure: s.Secure, SameSite: http.SameSiteLaxMode, Expires: current.ExpiresAt}
	return current, cookie, nil
}
