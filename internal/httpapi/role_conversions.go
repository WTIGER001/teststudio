package httpapi

import (
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"net/http"
)

func (s *Server) roleConversionRoutes(m *http.ServeMux) {
	a := s.accessService()
	for _, base := range []string{"/api/v1/projects/{project}/role-conversions", "/api/v1/projects/{project}/applications/{application}/role-conversions"} {
		wrap := func(fn http.HandlerFunc) http.HandlerFunc {
			return s.authorized(func(w http.ResponseWriter, r *http.Request) {
				if app := r.PathValue("application"); app != "" {
					if err := s.Auth.CheckApplication(r.Context(), actor(r), r.PathValue("project"), app, "", "members.read"); err != nil {
						problem(w, err)
						return
					}
					r = r.WithContext(authorization.WithApplication(r.Context(), r.PathValue("project"), app))
				}
				fn(w, r)
			})
		}
		memberBase := base[:len(base)-len("role-conversions")]
		m.HandleFunc("GET "+memberBase+"member-candidates/{user}", wrap(func(w http.ResponseWriter, r *http.Request) {
			value, err := a.MemberCandidate(r.Context(), actor(r), r.PathValue("project"), r.PathValue("user"))
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, value)
		}))
		m.HandleFunc("GET "+memberBase+"ownership", wrap(func(w http.ResponseWriter, r *http.Request) {
			value, err := a.Ownership(r.Context(), actor(r), r.PathValue("project"))
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, value)
		}))
		m.HandleFunc("POST "+base+"/preview", wrap(func(w http.ResponseWriter, r *http.Request) {
			var d access.AssignmentDefinition
			if err := body(r, &d); err != nil {
				problem(w, err)
				return
			}
			row, err := a.PreviewRoleConversion(r.Context(), actor(r), r.PathValue("project"), d)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
		m.HandleFunc("GET "+base+"/{conversion}", wrap(func(w http.ResponseWriter, r *http.Request) {
			row, err := a.ReadRoleConversion(r.Context(), actor(r), r.PathValue("project"), r.PathValue("conversion"))
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
		m.HandleFunc("POST "+base+"/{conversion}/apply", wrap(func(w http.ResponseWriter, r *http.Request) {
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			row, err := a.ApplyRoleConversion(r.Context(), actor(r), r.PathValue("project"), r.PathValue("conversion"), rev)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
		m.HandleFunc("POST "+base+"/{conversion}/rollback-preview", wrap(func(w http.ResponseWriter, r *http.Request) {
			var d struct {
				Reason string `json:"reason"`
			}
			if err := body(r, &d); err != nil {
				problem(w, err)
				return
			}
			row, err := a.PreviewRoleConversionRollback(r.Context(), actor(r), r.PathValue("project"), r.PathValue("conversion"), d.Reason)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
	}
}
