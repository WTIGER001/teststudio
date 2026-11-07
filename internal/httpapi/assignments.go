package httpapi

import (
	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"net/http"
)

func (s *Server) assignmentRoutes(m *http.ServeMux) {
	s.roleConversionRoutes(m)
	a := s.accessService()
	m.HandleFunc("GET /api/v1/roles", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		if app := r.URL.Query().Get("application"); app != "" {
			if err := s.Auth.CheckApplication(r.Context(), actor(r), r.URL.Query().Get("project"), app, "", "roles.read"); err != nil {
				problem(w, err)
				return
			}
			r = r.WithContext(authorization.WithApplication(r.Context(), r.URL.Query().Get("project"), app))
		}
		roles, err := a.ListRoles(r.Context(), actor(r), r.URL.Query().Get("project"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": roles})
	}))
	for _, path := range []string{"/api/v1/role-assignments", "/api/v1/projects/{project}/role-assignments", "/api/v1/projects/{project}/applications/{application}/role-assignments"} {
		wrap := func(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
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
		m.HandleFunc("GET "+path, wrap(func(w http.ResponseWriter, r *http.Request) {
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
		m.HandleFunc("POST "+path, wrap(func(w http.ResponseWriter, r *http.Request) {
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
			m.HandleFunc(method+" "+path+"/{kind}/{principal}/{role}", wrap(func(w http.ResponseWriter, r *http.Request) {
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
		if path != "/api/v1/role-assignments" {
			m.HandleFunc("PUT "+path+"/{kind}/{principal}", wrap(func(w http.ResponseWriter, r *http.Request) {
				var d struct {
					PreviousRole string `json:"previousRoleId,omitempty"`
					Role         string `json:"roleId"`
					Reason       string `json:"reason"`
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
				row, err := a.ReplaceStandardAssignment(r.Context(), actor(r), r.PathValue("project"), access.AssignmentDefinition{PreviousRole: d.PreviousRole, PrincipalKind: permissions.PrincipalKind(r.PathValue("kind")), PrincipalID: r.PathValue("principal"), RoleID: d.Role, Reason: d.Reason}, rev)
				if err != nil {
					problem(w, err)
					return
				}
				JSON(w, 200, row)
			}))
		}
	}
}
