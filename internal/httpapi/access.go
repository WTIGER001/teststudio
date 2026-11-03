package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/sessions"
)

func (s *Server) accessRoutes(m *http.ServeMux) {
	a := s.accessService()
	m.HandleFunc("GET /api/v1/me", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		profile, err := a.CurrentProfile(r.Context(), actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, profile)
	}))
	m.HandleFunc("GET /api/v1/users", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := a.ListUsers(r.Context(), actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET /api/v1/users/{user}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := a.ReadUser(r.Context(), actor(r), r.PathValue("user"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	for _, method := range []string{"POST", "PUT"} {
		path := "/api/v1/users"
		if method == "PUT" {
			path += "/{user}"
		}
		m.HandleFunc(method+" "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var d access.UserDefinition
			if err := body(r, &d); err != nil {
				problem(w, err)
				return
			}
			var rev int64
			if method == "PUT" {
				var err error
				rev, err = revision(r)
				if err != nil || rev < 1 {
					problem(w, errors.New("existing users require a positive If-Match revision"))
					return
				}
				if d.ID != r.PathValue("user") {
					problem(w, errors.New("user identifier is permanent"))
					return
				}
			}
			if err := a.SaveUser(r.Context(), actor(r), d, rev); err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, "users", d.ID)
		}))
	}
	for _, action := range []string{"disable", "restore"} {
		m.HandleFunc("POST /api/v1/users/{user}/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
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
			if err := a.UserState(r.Context(), actor(r), r.PathValue("user"), d.Reason, rev, action == "disable"); err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, "users", r.PathValue("user"))
		}))
	}
	for _, kind := range []string{"identities", "sessions", "memberships"} {
		m.HandleFunc("GET /api/v1/users/{user}/"+kind, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			limit, err := pageLimit(r, 30)
			if err != nil {
				problem(w, err)
				return
			}
			var page access.Page
			if kind == "memberships" {
				page, err = a.UserMemberships(r.Context(), actor(r), r.PathValue("user"), r.URL.Query().Get("cursor"), limit)
			} else if kind == "identities" {
				page, err = a.UserIdentities(r.Context(), actor(r), r.PathValue("user"), r.URL.Query().Get("cursor"), limit)
			} else {
				page, err = a.UserSessions(r.Context(), actor(r), r.PathValue("user"), r.Context().Value(sessionKey{}).(sessions.Session).ID, r.URL.Query().Get("cursor"), limit)
			}
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, page)
		}))
	}
	m.HandleFunc("DELETE /api/v1/users/{user}/sessions/{session}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
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
		if err := a.RevokeSession(r.Context(), actor(r), r.PathValue("user"), r.PathValue("session"), d.Reason, rev); err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]bool{"revoked": true})
	}))
	m.HandleFunc("GET /api/v1/groups", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := a.ListGroups(r.Context(), actor(r), r.URL.Query().Get("project"), r.URL.Query().Get("cursor"), limit, r.URL.Query().Get("archived") == "true")
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET /api/v1/groups/{group}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := a.ReadGroup(r.Context(), actor(r), r.PathValue("group"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	for _, method := range []string{"POST", "PUT"} {
		path := "/api/v1/groups"
		if method == "PUT" {
			path += "/{group}"
		}
		m.HandleFunc(method+" "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var d access.GroupDefinition
			if err := body(r, &d); err != nil {
				problem(w, err)
				return
			}
			var rev int64
			if method == "PUT" {
				var err error
				rev, err = revision(r)
				if err != nil || rev < 1 {
					problem(w, errors.New("existing groups require a positive If-Match revision"))
					return
				}
				if d.ID != r.PathValue("group") {
					problem(w, errors.New("group identifier is permanent"))
					return
				}
			}
			if err := a.SaveGroup(r.Context(), actor(r), d, rev); err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, "groups", d.ID)
		}))
	}
	for _, action := range []string{"archive", "restore"} {
		m.HandleFunc("POST /api/v1/groups/{group}/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
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
			if err := a.GroupState(r.Context(), actor(r), r.PathValue("group"), d.Reason, rev, action == "archive"); err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, "groups", r.PathValue("group"))
		}))
	}
	m.HandleFunc("GET /api/v1/groups/{group}/members", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := a.ListMembers(r.Context(), actor(r), r.PathValue("group"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	for _, method := range []string{"PUT", "DELETE"} {
		m.HandleFunc(method+" /api/v1/groups/{group}/members/{user}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			if method == "PUT" {
				var d struct {
					ExpiresAt *time.Time `json:"expiresAt"`
				}
				if err := body(r, &d); err != nil {
					problem(w, err)
					return
				}
				err = a.SaveMember(r.Context(), actor(r), r.PathValue("group"), r.PathValue("user"), d.ExpiresAt, rev)
			} else {
				var d struct {
					Reason string `json:"reason"`
				}
				if err := body(r, &d); err != nil {
					problem(w, err)
					return
				}
				err = a.RemoveMember(r.Context(), actor(r), r.PathValue("group"), r.PathValue("user"), d.Reason, rev)
			}
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, map[string]bool{"saved": true})
		}))
	}
	m.HandleFunc("GET /api/v1/groups/{group}/members/{user}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := a.ReadMember(r.Context(), actor(r), r.PathValue("group"), r.PathValue("user"), r.URL.Query().Get("source"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
}

func (s *Server) accessService() *access.Service {
	service := &access.Service{Store: s.Store, Auth: s.Auth, LocalUser: s.LocalUser, LocalAccounts: s.LocalAccounts != nil, SessionIdleTimeout: s.Sessions.IdleTimeout, AuthOrigin: s.Sessions.Origin}
	if s.OIDC != nil {
		service.ProviderSecretRefs = s.OIDC.SecretRefs
	}
	return service
}
