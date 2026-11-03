package httpapi

import (
	"errors"
	"net"
	"net/http"

	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/sessions"
)

func peer(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || net.ParseIP(host) == nil {
		return "unknown"
	}
	return net.ParseIP(host).String()
}

func passwordProblem(w http.ResponseWriter, err error) {
	if errors.Is(err, localauth.ErrInvalid) || errors.Is(err, sessions.ErrUnauthenticated) {
		JSON(w, 401, map[string]string{"message": localauth.ErrInvalid.Error()})
		return
	}
	if errors.Is(err, localauth.ErrLimited) || errors.Is(err, passwords.ErrBusy) {
		wait := "900"
		if errors.Is(err, passwords.ErrBusy) {
			wait = "1"
		}
		w.Header().Set("Retry-After", wait)
		JSON(w, 429, map[string]string{"message": "Sign-in attempts are temporarily limited. Try again later."})
		return
	}
	JSON(w, 503, map[string]string{"message": "Local sign-in is unavailable. Try again shortly."})
}

func (s *Server) localAccountRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /auth/password", func(w http.ResponseWriter, r *http.Request) {
		if s.LocalAccounts == nil || r.Header.Get("Origin") != s.Sessions.Origin {
			JSON(w, 403, map[string]string{"message": "local password sign-in is unavailable"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		var d struct {
			User     string `json:"userId"`
			Password string `json:"password"`
		}
		if err := body(r, &d); err != nil {
			JSON(w, 401, map[string]string{"message": localauth.ErrInvalid.Error()})
			return
		}
		previous := ""
		if cookie, err := r.Cookie("teststudio_session"); err == nil && len(cookie.Value) == 43 {
			previous = sessions.Digest(cookie.Value)
		}
		session, cookie, err := s.LocalAccounts.Login(r.Context(), d.User, d.Password, peer(r), previous)
		if err != nil {
			passwordProblem(w, err)
			return
		}
		http.SetCookie(w, cookie)
		JSON(w, 200, map[string]any{"signedIn": true, "mustChangePassword": session.MustChangePassword})
	})
	m.HandleFunc("POST /auth/password/change", s.authenticated(func(w http.ResponseWriter, r *http.Request) {
		if s.LocalAccounts == nil {
			JSON(w, 403, map[string]string{"message": "local password sign-in is unavailable"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		var d struct {
			Current string `json:"currentPassword"`
			New     string `json:"newPassword"`
		}
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		if err := passwords.Validate(d.New); err != nil {
			problem(w, err)
			return
		}
		current := r.Context().Value(sessionKey{}).(sessions.Session)
		_, cookie, err := s.LocalAccounts.Change(r.Context(), current, d.Current, d.New, peer(r))
		if err != nil {
			if errors.Is(err, localauth.ErrInvalid) {
				JSON(w, 422, map[string]string{"message": "Current password is incorrect."})
				return
			}
			if errors.Is(err, localauth.ErrLimited) || errors.Is(err, passwords.ErrBusy) {
				passwordProblem(w, err)
			} else {
				problem(w, err)
			}
			return
		}
		http.SetCookie(w, cookie)
		JSON(w, 200, map[string]bool{"changed": true})
	}, true))
	a := s.accessService()
	m.HandleFunc("GET /api/v1/me/local-credential", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		result, err := a.OwnLocalCredential(r.Context(), actor(r))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("GET /api/v1/users/{user}/local-credential", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		result, err := a.ReadLocalCredential(r.Context(), actor(r), r.PathValue("user"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("PUT /api/v1/users/{user}/local-credential", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var d struct {
			Password string `json:"password"`
			Reason   string `json:"reason"`
		}
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		result, err := a.SetLocalPassword(r.Context(), actor(r), r.PathValue("user"), d.Password, d.Reason, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	for _, action := range []string{"disable", "enable"} {
		m.HandleFunc("POST /api/v1/users/{user}/local-credential/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var d struct {
				Reason string `json:"reason"`
			}
			if err := body(r, &d); err != nil {
				problem(w, err)
				return
			}
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			result, err := a.LocalCredentialState(r.Context(), actor(r), r.PathValue("user"), d.Reason, rev, action == "disable")
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, result)
		}))
	}
}
