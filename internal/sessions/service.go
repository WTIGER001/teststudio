package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrPasswordChangeRequired = errors.New("change your password before continuing")

var ErrUnauthenticated = errors.New("sign-in is required")

type Session struct {
	ID                   string    `json:"-"`
	UserID               string    `json:"userId"`
	Provider             string    `json:"provider"`
	AuthenticationMethod string    `json:"authenticationMethod,omitempty"`
	LocalModeRevision    int64     `json:"localModeRevision,omitempty"`
	CredentialRevision   int64     `json:"credentialRevision,omitempty"`
	MustChangePassword   bool      `json:"mustChangePassword,omitempty"`
	UserEpoch            int64     `json:"userEpoch"`
	CSRF                 string    `json:"csrfToken"`
	CreatedAt            time.Time `json:"createdAt"`
	LastSeen             time.Time `json:"lastSeen"`
	ExpiresAt            time.Time `json:"expiresAt"`
	AuthenticationUntil  time.Time `json:"authenticationUntil"`
}

type Service struct {
	Store         *storage.Store
	Directory     *identity.Directory
	Origin        string
	Secure        bool
	IdleTimeout   time.Duration
	Lifetime      time.Duration
	LocalAccounts bool
}

func Random() string {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data[:])
}

func Digest(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}

func (s *Service) Create(ctx context.Context, w http.ResponseWriter, user, provider string, authUntil time.Time) (Session, error) {
	session, cookie, err := s.create(ctx, user, provider, authUntil, 0, "")
	if err == nil {
		http.SetCookie(w, cookie)
	}
	return session, err
}

func (s *Service) CreatePassword(ctx context.Context, user string, credentialRevision int64, previous string) (Session, *http.Cookie, error) {
	if !s.LocalAccounts || credentialRevision < 1 {
		return Session{}, nil, ErrUnauthenticated
	}
	return s.create(ctx, user, "password", time.Time{}, credentialRevision, previous)
}

func (s *Service) create(ctx context.Context, user, provider string, authUntil time.Time, credentialRevision int64, previous string) (Session, *http.Cookie, error) {
	lifetime := s.Lifetime
	if lifetime == 0 {
		lifetime = 8 * time.Hour
	}
	for range 8 {
		version, err := s.Directory.Revision(ctx)
		if err != nil {
			return Session{}, nil, err
		}
		row, err := s.Store.Get(ctx, "users", user)
		if err != nil {
			return Session{}, nil, err
		}
		u, err := storage.Decode[identity.User](row)
		if err != nil {
			return Session{}, nil, err
		}
		if u.Disabled {
			return Session{}, nil, ErrUnauthenticated
		}
		mustChange := false
		localModeRevision := int64(0)
		if credentialRevision > 0 {
			mode, err := s.Store.Get(ctx, "local-auth-mode", "installation")
			if errors.Is(err, storage.ErrNotFound) {
				return Session{}, nil, ErrUnauthenticated
			}
			if err != nil {
				return Session{}, nil, err
			}
			state, err := storage.Decode[identity.LocalAuthMode](mode)
			if err != nil {
				return Session{}, nil, err
			}
			if !state.Enabled {
				return Session{}, nil, ErrUnauthenticated
			}
			localModeRevision = mode.Revision
			credential, err := s.Store.Get(ctx, "local-credentials", user)
			if errors.Is(err, storage.ErrNotFound) {
				return Session{}, nil, ErrUnauthenticated
			}
			if err != nil {
				return Session{}, nil, err
			}
			value, err := storage.Decode[identity.LocalCredential](credential)
			if err != nil {
				return Session{}, nil, err
			}
			if value.Disabled || !passwords.ValidHash(value.Hash) || value.UserID != user || credential.Revision != credentialRevision {
				return Session{}, nil, ErrUnauthenticated
			}
			mustChange = value.MustChange
		}
		now := time.Now().UTC()
		until := authUntil
		if until.IsZero() {
			until = now.Add(lifetime)
		}
		token := Random()
		method := ""
		if credentialRevision > 0 {
			method = "password"
		}
		v := Session{LocalModeRevision: localModeRevision, AuthenticationMethod: method, ID: Digest(token), UserID: user, Provider: provider, CredentialRevision: credentialRevision, MustChangePassword: mustChange, UserEpoch: u.SessionEpoch, CSRF: Random(), CreatedAt: now, LastSeen: now, ExpiresAt: now.Add(lifetime), AuthenticationUntil: until}
		c, err := storage.Put("sessions", v.ID, "", 0, v)
		if err != nil {
			return Session{}, nil, err
		}
		u.LastSignIn = &now
		profile, err := storage.Put("users", user, "", row.Revision, u)
		if err != nil {
			return Session{}, nil, err
		}
		changes := []storage.Change{profile, c}
		if previous != "" {
			prior, err := s.Store.Get(ctx, "sessions", previous)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return Session{}, nil, err
			}
			if prior.Revision > 0 {
				changes = append(changes, storage.Change{Record: prior, Delete: true})
			}
		}
		if err := s.Directory.CommitAt(ctx, version, changes...); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return Session{}, nil, err
		}
		cookie := &http.Cookie{Name: "teststudio_session", Value: token, Path: "/", HttpOnly: true, Secure: s.Secure, SameSite: http.SameSiteLaxMode, Expires: v.ExpiresAt}
		return v, cookie, nil
	}
	return Session{}, nil, storage.ErrConflict
}

func (s *Service) Read(ctx context.Context, r *http.Request) (Session, error) {
	cookie, err := r.Cookie("teststudio_session")
	if err != nil || len(cookie.Value) != 43 {
		return Session{}, ErrUnauthenticated
	}
	for range 3 {
		record, err := s.Store.Get(ctx, "sessions", Digest(cookie.Value))
		if errors.Is(err, storage.ErrNotFound) {
			return Session{}, ErrUnauthenticated
		}
		if err != nil {
			return Session{}, err
		}
		v, err := storage.Decode[Session](record)
		if err != nil {
			return Session{}, err
		}
		v.ID = record.ID
		now := time.Now().UTC()
		idle := s.IdleTimeout
		if idle == 0 {
			idle = time.Hour
		}
		if !now.Before(v.ExpiresAt) || !now.Before(v.AuthenticationUntil) || now.Sub(v.LastSeen) > idle {
			return Session{}, ErrUnauthenticated
		}
		u, err := s.Directory.User(ctx, v.UserID)
		if err != nil {
			return Session{}, err
		}
		if u.Disabled || u.SessionEpoch != v.UserEpoch {
			return Session{}, ErrUnauthenticated
		}
		if err := s.checkCredential(ctx, v); err != nil {
			return Session{}, err
		}
		if now.Sub(v.LastSeen) < time.Minute {
			return v, nil
		}
		v.LastSeen = now
		change, err := storage.Put(record.Bucket, record.ID, "", record.Revision, v)
		if err != nil {
			return Session{}, err
		}
		if err := s.Store.Commit(ctx, change); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return Session{}, err
		}
		return v, nil
	}
	return Session{}, ErrUnauthenticated
}

func (s *Service) Validate(ctx context.Context, id string) (Session, error) {
	record, err := s.Store.Get(ctx, "sessions", id)
	if errors.Is(err, storage.ErrNotFound) {
		return Session{}, ErrUnauthenticated
	}
	if err != nil {
		return Session{}, err
	}
	v, err := storage.Decode[Session](record)
	if err != nil {
		return Session{}, err
	}
	v.ID = record.ID
	now := time.Now().UTC()
	idle := s.IdleTimeout
	if idle == 0 {
		idle = time.Hour
	}
	if !now.Before(v.ExpiresAt) || !now.Before(v.AuthenticationUntil) || now.Sub(v.LastSeen) > idle {
		return Session{}, ErrUnauthenticated
	}
	u, err := s.Directory.User(ctx, v.UserID)
	if err != nil {
		return Session{}, err
	}
	if u.Disabled || u.SessionEpoch != v.UserEpoch {
		return Session{}, ErrUnauthenticated
	}
	if err := s.checkCredential(ctx, v); err != nil {
		return Session{}, err
	}
	return v, nil
}

func (s *Service) checkCredential(ctx context.Context, v Session) error {
	if v.AuthenticationMethod != "password" {
		return nil
	}
	if !s.LocalAccounts {
		return ErrUnauthenticated
	}
	mode, err := s.Store.Get(ctx, "local-auth-mode", "installation")
	if errors.Is(err, storage.ErrNotFound) {
		return ErrUnauthenticated
	}
	if err != nil {
		return err
	}
	state, err := storage.Decode[identity.LocalAuthMode](mode)
	if err != nil {
		return err
	}
	if !state.Enabled || v.LocalModeRevision < 1 || mode.Revision != v.LocalModeRevision {
		return ErrUnauthenticated
	}
	row, err := s.Store.Get(ctx, "local-credentials", v.UserID)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrUnauthenticated
	}
	if err != nil {
		return err
	}
	c, err := storage.Decode[identity.LocalCredential](row)
	if err != nil {
		return err
	}
	if c.Disabled || !passwords.ValidHash(c.Hash) || c.UserID != v.UserID || v.CredentialRevision < 1 || row.Revision != v.CredentialRevision || c.MustChange != v.MustChangePassword {
		return ErrUnauthenticated
	}
	return nil
}

func (s *Service) CheckMutation(r *http.Request, v Session) error {
	if r.Header.Get("Origin") != s.Origin {
		return errors.New("request origin is not allowed")
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(v.CSRF)) != 1 || v.CSRF == "" {
		return errors.New("CSRF token is required")
	}
	return nil
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	r, err := s.Store.Get(ctx, "sessions", id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, storage.Change{Record: r, Delete: true})
}

func (s *Service) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "teststudio_session", Value: "", Path: "/", HttpOnly: true, Secure: s.Secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
