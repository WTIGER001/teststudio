package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/access"
)

func (s *Server) rolePolicyRoutes(m *http.ServeMux) {
	a := s.accessService()
	m.HandleFunc("GET /api/v1/permissions", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		items, err := a.PermissionCatalog(r.Context(), actor(r), r.URL.Query().Get("project"), r.URL.Query().Get("object"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": items})
	}))
	m.HandleFunc("GET /api/v1/roles/custom", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := a.ListRolePolicies(r.Context(), actor(r), r.URL.Query().Get("project"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET /api/v1/roles/{role}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := a.ReadRolePolicy(r.Context(), actor(r), r.PathValue("role"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	for _, method := range []string{"POST", "PUT"} {
		path := "/api/v1/roles"
		if method == "PUT" {
			path += "/{role}"
		}
		m.HandleFunc(method+" "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var input access.RolePolicyMutation
			if err := body(r, &input); err != nil {
				problem(w, err)
				return
			}
			var rev int64
			if method == "PUT" {
				var err error
				rev, err = revision(r)
				if err != nil || rev < 1 {
					problem(w, errors.New("existing roles require a positive If-Match revision"))
					return
				}
				if input.ID != r.PathValue("role") {
					problem(w, errors.New("role identifier is permanent"))
					return
				}
			}
			row, err := a.SaveRolePolicy(r.Context(), actor(r), input, rev)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
	}
}
