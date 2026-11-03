package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/access"
)

func (s *Server) grantRoutes(m *http.ServeMux) {
	a := s.accessService()
	for _, path := range []string{"/api/v1/permissions/grants", "/api/v1/projects/{project}/permissions"} {
		m.HandleFunc("GET "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			limit, err := pageLimit(r, 30)
			if err != nil {
				problem(w, err)
				return
			}
			page, err := a.ListGrants(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("cursor"), limit)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, page)
		}))
		m.HandleFunc("POST "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var input access.GrantDefinition
			if err := body(r, &input); err != nil {
				problem(w, err)
				return
			}
			row, err := a.CreateGrant(r.Context(), actor(r), r.PathValue("project"), input)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
		m.HandleFunc("GET "+path+"/{grant}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
			row, err := a.ReadGrant(r.Context(), actor(r), r.PathValue("project"), r.PathValue("grant"))
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
		for _, method := range []string{"PUT", "DELETE"} {
			m.HandleFunc(method+" "+path+"/{grant}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
				rev, err := revision(r)
				if err != nil || rev < 1 {
					problem(w, errors.New("existing grants require a positive If-Match revision"))
					return
				}
				if method == "DELETE" {
					var input struct {
						Reason string `json:"reason"`
					}
					if err := body(r, &input); err != nil {
						problem(w, err)
						return
					}
					if err := a.RemoveGrant(r.Context(), actor(r), r.PathValue("project"), r.PathValue("grant"), input.Reason, rev); err != nil {
						problem(w, err)
						return
					}
					JSON(w, 200, map[string]bool{"removed": true})
					return
				}
				var input access.GrantUpdate
				if err := body(r, &input); err != nil {
					problem(w, err)
					return
				}
				row, err := a.UpdateGrant(r.Context(), actor(r), r.PathValue("project"), r.PathValue("grant"), input, rev)
				if err != nil {
					problem(w, err)
					return
				}
				JSON(w, 200, row)
			}))
		}
	}
}
