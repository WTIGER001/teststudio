package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/accountlinks"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/sessions"
)

func (s *Server) identityLinkRoutes(m *http.ServeMux) {
	links := &accountlinks.Service{Store: s.Store, Sessions: s.Sessions, LocalAccounts: s.LocalAccounts, LocalUser: s.LocalUser}
	if s.OIDC != nil {
		s.OIDC.Links = links
	}
	m.HandleFunc("GET /api/v1/me/identity-linking", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		active := r.Context().Value(sessionKey{}).(sessions.Session)
		result, err := links.Options(r.Context(), active.ID, peer(r))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("POST /api/v1/me/identity-links", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		var input accountlinks.StartInput
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		active := r.Context().Value(sessionKey{}).(sessions.Session)
		result, err := links.Start(r.Context(), active.ID, input, peer(r))
		if err != nil {
			if errors.Is(err, localauth.ErrInvalid) {
				JSON(w, 422, map[string]string{"message": "Current password is incorrect."})
				return
			}
			if errors.Is(err, localauth.ErrLimited) || errors.Is(err, passwords.ErrBusy) {
				passwordProblem(w, err)
				return
			}
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("GET /api/v1/me/identity-links/{operation}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		active := r.Context().Value(sessionKey{}).(sessions.Session)
		result, err := links.Read(r.Context(), active.ID, r.PathValue("operation"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("POST /api/v1/me/identity-links/{operation}/cancel", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		active := r.Context().Value(sessionKey{}).(sessions.Session)
		result, err := links.Cancel(r.Context(), active.ID, r.PathValue("operation"), rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("POST /api/v1/me/identity-links/{operation}/confirm", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		active := r.Context().Value(sessionKey{}).(sessions.Session)
		result, cookie, err := links.Confirm(r.Context(), active.ID, r.PathValue("operation"), rev)
		if err != nil {
			problem(w, err)
			return
		}
		http.SetCookie(w, cookie)
		JSON(w, 200, result)
	}))
	m.HandleFunc("GET /auth/identity-links/{operation}/authorize", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		if s.OIDC == nil {
			problem(w, accountlinks.ErrInvalid)
			return
		}
		if err := s.OIDC.AuthorizeIdentityLink(w, r, r.PathValue("operation")); err != nil {
			problem(w, err)
		}
	}))
}
