package httpapi

import (
	"errors"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"net/http"
)

func (s *Server) applicationConfigurationRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/applications/{application}"
	m.HandleFunc("GET "+base+"/catalog", s.authorized(s.catalogList))
	m.HandleFunc("POST "+base+"/default-suite", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		revision, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if revision < 1 {
			problem(w, errors.New("default suite creation requires the current application revision"))
			return
		}
		if err := s.Projects.EnsureDefaultSuite(r.Context(), actor(r), r.PathValue("project"), r.PathValue("application"), revision); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, projects.Bucket("applications", r.PathValue("project")), r.PathValue("application"))
	}))
	m.HandleFunc("GET "+base+"/configuration", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Projects.Configuration(r.Context(), actor(r), r.PathValue("project"), r.PathValue("application"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("PUT "+base+"/configuration", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var value struct {
			Requirements []projects.Requirement `json:"requirements"`
		}
		if err := body(r, &value); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if rev < 1 {
			problem(w, errors.New("configuration updates require the current application revision"))
			return
		}
		if err := s.Projects.SaveRequirements(r.Context(), actor(r), r.PathValue("project"), r.PathValue("application"), value.Requirements, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, projects.Bucket("applications", r.PathValue("project")), r.PathValue("application"))
	}))
	m.HandleFunc("GET "+base+"/credential-references", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		refs, err := s.Projects.ApplicationCredentialReferences(r.Context(), actor(r), r.PathValue("project"), r.PathValue("application"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": refs})
	}))
	m.HandleFunc("PUT "+base+"/environments/{environment}/configuration", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		project, application, environment := r.PathValue("project"), r.PathValue("application"), r.PathValue("environment")
		var value struct {
			Endpoints   map[string]string `json:"endpoints"`
			Parameters  map[string]string `json:"parameters"`
			Credentials map[string]string `json:"credentialBindings"`
		}
		if err := body(r, &value); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if rev < 1 {
			problem(w, errors.New("configuration updates require the current environment revision"))
			return
		}
		if err := s.Auth.CheckApplication(r.Context(), actor(r), project, application, "environment/"+environment, "environments.manage"); err != nil {
			problem(w, err)
			return
		}
		row, err := s.Store.Get(r.Context(), projects.Bucket("environments", project), environment)
		if err != nil {
			problem(w, err)
			return
		}
		e, err := storage.Decode[projects.Environment](row)
		if err != nil {
			problem(w, err)
			return
		}
		if e.Application != application {
			problem(w, storage.ErrNotFound)
			return
		}
		e.Endpoints, e.Parameters, e.Credentials = value.Endpoints, value.Parameters, value.Credentials
		if err := s.Projects.SaveEnvironment(r.Context(), actor(r), project, e, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, row.Bucket, row.ID)
	}))
}
