package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/access"
)

func (s *Server) builtinRoleRoutes(m *http.ServeMux) {
	a := s.accessService()
	m.HandleFunc("GET /api/v1/predefined-roles", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rows, err := a.BuiltinRoles(r.Context(), actor(r))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": rows})
	}))
	m.HandleFunc("GET /api/v1/predefined-roles/{role}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := a.ReadBuiltinRole(r.Context(), actor(r), r.PathValue("role"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("PUT /api/v1/predefined-roles/{role}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Disabled *bool  `json:"disabled"`
			Reason   string `json:"reason"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		if input.Disabled == nil {
			problem(w, errors.New("an explicit disabled state is required"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		row, err := a.UpdateBuiltinRole(r.Context(), actor(r), r.PathValue("role"), access.BuiltinRoleUpdate{Disabled: *input.Disabled, Reason: input.Reason}, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
}
