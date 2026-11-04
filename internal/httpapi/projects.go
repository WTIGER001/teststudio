package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) projectRoutes(m *http.ServeMux) {
	s.suiteLifecycleRoutes(m)
	m.HandleFunc("GET /api/v1/projects/{project}/api-credential-references", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		refs, err := s.Projects.CredentialReferences(r.Context(), actor(r), r.PathValue("project"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": refs})
	}))
	m.HandleFunc("GET /api/v1/projects", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.Store.All(r.Context(), "projects", "")
		if err != nil {
			problem(w, err)
			return
		}
		allowed := []storage.Record{}
		for _, row := range rows {
			err := s.Auth.Check(r.Context(), actor(r), row.ID, "", "projects.read")
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			if err != nil {
				problem(w, err)
				return
			}
			allowed = append(allowed, row)
		}
		JSON(w, 200, map[string]any{"items": allowed})
	}))
	m.HandleFunc("POST /api/v1/projects", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var p projects.Project
		if err := body(r, &p); err != nil {
			problem(w, err)
			return
		}
		if err := s.Projects.SaveProject(r.Context(), actor(r), p, 0); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, "projects", p.ID)
	}))
	m.HandleFunc("GET /api/v1/projects/{project}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		p := r.PathValue("project")
		if err := s.Auth.Check(r.Context(), actor(r), p, "", "projects.read"); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, "projects", p)
	}))
	m.HandleFunc("PUT /api/v1/projects/{project}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var p projects.Project
		if err := body(r, &p); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if p.ID != r.PathValue("project") {
			problem(w, errors.New("project identifier is permanent"))
			return
		}
		if err := s.Projects.SaveProject(r.Context(), actor(r), p, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, "projects", p.ID)
	}))
	for _, kind := range []string{"applications", "environments", "suites"} {
		kind := kind
		permission := "projects.read"
		if kind == "environments" {
			permission = "environments.read"
		}
		if kind == "suites" {
			permission = "suites.read"
		}
		m.HandleFunc("GET /api/v1/projects/{project}/"+kind, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			p := r.PathValue("project")
			if err := s.Auth.Check(r.Context(), actor(r), p, "", permission); err != nil {
				problem(w, err)
				return
			}
			rows, err := s.Store.All(r.Context(), projects.Bucket(kind, p), p)
			if err != nil {
				problem(w, err)
				return
			}
			if kind == "suites" {
				allowed := []storage.Record{}
				for _, row := range rows {
					if err := s.Auth.Check(r.Context(), actor(r), p, "suite/"+row.ID, permission); err == nil {
						allowed = append(allowed, row)
					} else if !errors.Is(err, authorization.ErrForbidden) {
						problem(w, err)
						return
					}
				}
				rows = allowed
			}
			JSON(w, 200, map[string]any{"items": rows})
		}))
		m.HandleFunc("PUT /api/v1/projects/{project}/"+kind+"/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
			p, id := r.PathValue("project"), r.PathValue("id")
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			switch kind {
			case "applications":
				var v projects.Application
				if err = body(r, &v); err == nil {
					if v.ID != id {
						err = errors.New("application identifier is permanent")
					} else {
						err = s.Projects.SaveApplication(r.Context(), actor(r), p, v, rev)
					}
				}
			case "environments":
				var v projects.Environment
				if err = body(r, &v); err == nil {
					if v.ID != id {
						err = errors.New("environment identifier is permanent")
					} else {
						err = s.Projects.SaveEnvironment(r.Context(), actor(r), p, v, rev)
					}
				}
			case "suites":
				var v projects.Suite
				if err = body(r, &v); err == nil {
					if v.ID != id {
						err = errors.New("suite identifier is permanent")
					} else {
						err = s.Projects.SaveSuite(r.Context(), actor(r), p, v, rev)
					}
				}
			}
			if err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, projects.Bucket(kind, p), id)
		}))
	}
}
