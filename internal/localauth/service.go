package localauth

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrInvalid = errors.New("user identifier or password is incorrect")
var ErrLimited = errors.New("too many sign-in attempts; try again later")

type Service struct {
	Store     *storage.Store
	Directory *identity.Directory
	Sessions  *sessions.Service
	key       []byte
}

func New(ctx context.Context, sessionService *sessions.Service) (*Service, error) {
	if !sessionService.LocalAccounts {
		return nil, errors.New("local accounts must be explicitly enabled")
	}
	if err := Configure(ctx, sessionService.Directory, true); err != nil {
		return nil, err
	}
	s := &Service{Store: sessionService.Store, Directory: sessionService.Directory, Sessions: sessionService}
	for range 8 {
		row, err := s.Store.Get(ctx, "local-auth-settings", "limiter-key")
		if errors.Is(err, storage.ErrNotFound) {
			key := make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				return nil, err
			}
			c, err := storage.Put("local-auth-settings", "limiter-key", "", 0, key)
			if err != nil {
				return nil, err
			}
			if err := s.Store.Commit(ctx, c); errors.Is(err, storage.ErrConflict) {
				continue
			} else if err != nil {
				return nil, err
			}
			s.key = key
			return s, nil
		}
		if err != nil {
			return nil, err
		}
		s.key, err = storage.Decode[[]byte](row)
		if err != nil {
			return nil, err
		}
		if len(s.key) != 32 {
			return nil, errors.New("invalid local sign-in limiter configuration")
		}
		return s, nil
	}
	return nil, storage.ErrConflict
}

func (s *Service) verify(ctx context.Context, user, password, peer string) (storage.Record, error) {
	if !domain.ValidID(user) || !utf8.ValidString(password) || len(password) > 1024 {
		return storage.Record{}, ErrInvalid
	}
	if err := s.reserve(ctx, user, peer); err != nil {
		return storage.Record{}, err
	}
	row, err := s.Store.Get(ctx, "local-credentials", user)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.Record{}, err
	}
	valid := false
	var credential identity.LocalCredential
	if row.Revision > 0 {
		credential, err = storage.Decode[identity.LocalCredential](row)
		if err != nil {
			return storage.Record{}, err
		}
	}
	if passwords.ValidHash(credential.Hash) {
		valid, err = passwords.Verify(ctx, credential.Hash, password)
	} else {
		err = passwords.DummyVerify(ctx, password)
	}
	if err != nil {
		return storage.Record{}, err
	}
	if !valid || credential.Disabled || credential.UserID != user {
		return storage.Record{}, ErrInvalid
	}
	profile, err := s.Directory.User(ctx, user)
	if errors.Is(err, storage.ErrNotFound) {
		return storage.Record{}, ErrInvalid
	}
	if err != nil {
		return storage.Record{}, err
	}
	if profile.Disabled {
		return storage.Record{}, ErrInvalid
	}
	return row, nil
}

func (s *Service) Login(ctx context.Context, user, password, peer, previous string) (sessions.Session, *http.Cookie, error) {
	row, err := s.verify(ctx, user, password, peer)
	if err != nil {
		return sessions.Session{}, nil, err
	}
	session, cookie, err := s.Sessions.CreatePassword(ctx, user, row.Revision, previous)
	if errors.Is(err, sessions.ErrUnauthenticated) {
		return sessions.Session{}, nil, ErrInvalid
	}
	return session, cookie, err
}

func (s *Service) VerifyCurrent(ctx context.Context, current sessions.Session, password, peer string) error {
	active, err := s.Sessions.Validate(ctx, current.ID)
	if err != nil {
		return err
	}
	if active.UserID != current.UserID || active.AuthenticationMethod != "password" || active.MustChangePassword {
		return sessions.ErrUnauthenticated
	}
	row, err := s.verify(ctx, active.UserID, password, peer)
	if err != nil {
		return err
	}
	if row.Revision != active.CredentialRevision {
		return sessions.ErrUnauthenticated
	}
	_, err = s.Sessions.Validate(ctx, active.ID)
	return err
}
