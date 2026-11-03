package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/wtiger001/teststudio/internal/sources"
)

func (s *Server) sourceRoutes(m *http.ServeMux) {
	service := s.Sources
	if service == nil {
		service = &sources.Service{Store: s.Store, Auth: s.Auth}
	}
	s.preparationRoutes(m, service)
	base := "/api/v1/projects/{project}/suites/{suite}/source-uploads"
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit := 30
		if value := r.URL.Query().Get("limit"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				problem(w, errors.New("invalid page limit"))
				return
			}
			limit = parsed
		}
		page, err := service.List(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("POST "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input sources.UploadInput
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		preview, err := service.Stage(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, preview)
	}))
	m.HandleFunc("GET "+base+"/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		preview, err := service.Read(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, preview)
	}))
	m.HandleFunc("GET "+base+"/{id}/content", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		content, err := service.Content(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, content)
	}))
	m.HandleFunc("POST "+base+"/{id}/publish", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Activate *bool `json:"activate"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		if input.Activate == nil {
			problem(w, errors.New("an explicit activate choice is required"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		release, err := service.Publish(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), rev, *input.Activate)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, release)
	}))
}
