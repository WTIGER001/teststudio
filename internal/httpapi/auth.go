package httpapi

import (
	"errors"
	"net"
	"net/http"
	"time"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) authRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /auth/providers", func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.Store.All(r.Context(), "providers", "")
		if err != nil {
			problem(w, err)
			return
		}
		list := []map[string]string{}
		for _, row := range rows {
			p, err := storage.Decode[authoidc.Provider](row)
			if err != nil {
				problem(w, err)
				return
			}
			if !p.Disabled {
				list = append(list, map[string]string{"id": p.ID, "name": p.Name})
			}
		}
		enabled := false
		if s.LocalAccounts != nil {
			enabled, err = s.Auth.Directory.LocalAccountsEnabled(r.Context())
			if err != nil {
				problem(w, err)
				return
			}
		}
		JSON(w, 200, map[string]any{"providers": list, "localEnabled": s.LocalUser != "", "localAccountsEnabled": enabled})
	})
	m.HandleFunc("POST /auth/local", func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || s.LocalUser == "" || r.Header.Get("Origin") != s.Sessions.Origin {
			JSON(w, 403, map[string]string{"message": "local sign-in is unavailable"})
			return
		}
		if _, err := s.Sessions.Create(r.Context(), w, s.LocalUser, "local", time.Time{}); err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]bool{"signedIn": true})
	})
	m.HandleFunc("GET /auth/session", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Sessions.Read(r.Context(), r)
		if err != nil {
			if errors.Is(err, sessions.ErrUnauthenticated) {
				JSON(w, 200, map[string]any{"user": nil, "session": nil})
				return
			}
			problem(w, err)
			return
		}
		u, err := s.Auth.Directory.User(r.Context(), v.UserID)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"user": u, "session": v})
	})
	m.HandleFunc("POST /auth/logout", s.authenticated(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(sessionKey{}).(sessions.Session)
		if err := s.Sessions.Revoke(r.Context(), v.ID); err != nil {
			problem(w, err)
			return
		}
		s.Sessions.Clear(w)
		JSON(w, 200, map[string]bool{"signedOut": true})
	}, true))
	m.HandleFunc("GET /auth/login/{provider}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.OIDC.Login(w, r, r.PathValue("provider")); err != nil {
			problem(w, errors.New("sign-in provider is unavailable"))
		}
	})
	m.HandleFunc("GET /auth/callback/{provider}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.OIDC.Callback(w, r, r.PathValue("provider")); err != nil {
			problem(w, err)
		}
	})
}
