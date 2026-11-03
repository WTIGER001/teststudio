package httpapi

import (
	"net/http"

	"github.com/wtiger001/teststudio/internal/projects"
)

func (s *Server) suiteLifecycleRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/suites/{suite}"
	m.HandleFunc("GET "+base+"/lifecycle", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Projects.SuiteLifecycle(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("PATCH "+base+"/settings", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input projects.SuiteSettings
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := s.Projects.UpdateSuiteSettings(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), input, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST "+base+"/actions/{action}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := s.Projects.ChangeSuiteState(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("action"), rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
}
