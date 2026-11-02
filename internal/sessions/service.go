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
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrUnauthenticated = errors.New("sign-in is required")

type Session struct {
	ID                  string    `json:"-"`
	UserID              string    `json:"userId"`
	Provider            string    `json:"provider"`
	CSRF                string    `json:"csrfToken"`
	CreatedAt           time.Time `json:"createdAt"`
	LastSeen            time.Time `json:"lastSeen"`
	ExpiresAt           time.Time `json:"expiresAt"`
	AuthenticationUntil time.Time `json:"authenticationUntil"`
}

type Service struct {
	Store       *storage.Store
	Directory   *identity.Directory
	Origin      string
	Secure      bool
	IdleTimeout time.Duration
	Lifetime    time.Duration
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
	u, err := s.Directory.User(ctx, user)
	if err != nil {
		return Session{}, err
	}
	if u.Disabled {
		return Session{}, ErrUnauthenticated
	}
	lifetime := s.Lifetime
	if lifetime == 0 {
		lifetime = 8 * time.Hour
	}
	now := time.Now().UTC()
	if authUntil.IsZero() {
		authUntil = now.Add(lifetime)
	}
	token := Random()
	v := Session{ID: Digest(token), UserID: user, Provider: provider, CSRF: Random(), CreatedAt: now, LastSeen: now, ExpiresAt: now.Add(lifetime), AuthenticationUntil: authUntil}
	c, err := storage.Put("sessions", v.ID, "", 0, v)
	if err != nil {
		return Session{}, err
	}
	if err := s.Store.Commit(ctx, c); err != nil {
		return Session{}, err
	}
	http.SetCookie(w, &http.Cookie{Name: "teststudio_session", Value: token, Path: "/", HttpOnly: true, Secure: s.Secure, SameSite: http.SameSiteLaxMode, Expires: v.ExpiresAt})
	return v, nil
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
		if u.Disabled {
			return Session{}, ErrUnauthenticated
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
