package httpapi

import (
	"net/http"

	"github.com/wtiger001/teststudio/internal/access"
)

func (s *Server) pendingAccessRoutes(m *http.ServeMux) {
	service := s.accessService()
	if s.OIDC != nil {
		s.OIDC.ProjectAccess = service
	}
	base := "/api/v1/projects/{project}/pending-access"
	m.HandleFunc("GET "+base+"/providers", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		choices, err := service.PendingProviderChoices(r.Context(), actor(r), r.PathValue("project"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": choices})
	}))
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		result, err := service.ListPendingAccess(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("POST "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input access.PendingAccessInput
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		result, err := service.CreatePendingAccess(r.Context(), actor(r), r.PathValue("project"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("GET "+base+"/{access}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ReadPendingAccess(r.Context(), actor(r), r.PathValue("project"), r.PathValue("access"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("POST "+base+"/{access}/cancel", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Reason string `json:"reason"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		result, err := service.CancelPendingAccess(r.Context(), actor(r), r.PathValue("project"), r.PathValue("access"), input.Reason, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
}
