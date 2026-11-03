package httpapi

import (
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"net/http"
)

func (s *Server) assignmentRoutes(m *http.ServeMux) {
	a := s.accessService()
	m.HandleFunc("GET /api/v1/roles", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		roles, err := a.ListRoles(r.Context(), actor(r), r.URL.Query().Get("project"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": roles})
	}))
	for _, path := range []string{"/api/v1/role-assignments", "/api/v1/projects/{project}/role-assignments"} {
		m.HandleFunc("GET "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			limit, err := pageLimit(r, 30)
			if err != nil {
				problem(w, err)
				return
			}
			page, err := a.ListAssignments(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("cursor"), limit)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, page)
		}))
		m.HandleFunc("POST "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var definition access.AssignmentDefinition
			if err := body(r, &definition); err != nil {
				problem(w, err)
				return
			}
			row, err := a.CreateAssignment(r.Context(), actor(r), r.PathValue("project"), definition)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
		for _, method := range []string{"GET", "DELETE"} {
			m.HandleFunc(method+" "+path+"/{kind}/{principal}/{role}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
				definition := access.AssignmentDefinition{PrincipalKind: permissions.PrincipalKind(r.PathValue("kind")), PrincipalID: r.PathValue("principal"), RoleID: r.PathValue("role")}
				if method == "GET" {
					row, err := a.ReadAssignment(r.Context(), actor(r), r.PathValue("project"), definition)
					if err != nil {
						problem(w, err)
						return
					}
					JSON(w, 200, row)
					return
				}
				var value struct {
					Reason string `json:"reason"`
				}
				if err := body(r, &value); err != nil {
					problem(w, err)
					return
				}
				revision, err := revision(r)
				if err != nil {
					problem(w, err)
					return
				}
				definition.Reason = value.Reason
				if err := a.RemoveAssignment(r.Context(), actor(r), r.PathValue("project"), definition, revision); err != nil {
					problem(w, err)
					return
				}
				JSON(w, 200, map[string]bool{"revoked": true})
			}))
		}
	}
}
