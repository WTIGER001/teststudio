package httpapi

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) projectCoverRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/v1/project-covers", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.Projects.ProjectCovers(r.Context(), actor(r))
		if err != nil {
			problem(w, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		JSON(w, 200, map[string]any{"items": rows})
	}))
	base := "/api/v1/projects/{project}/cover"
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Projects.ProjectCover(r.Context(), actor(r), r.PathValue("project"))
		if err != nil {
			problem(w, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		JSON(w, 200, value)
	}))
	m.HandleFunc("PUT "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input projects.Cover
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := s.Projects.SaveProjectCover(r.Context(), actor(r), r.PathValue("project"), input, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/content", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Projects.ProjectCover(r.Context(), actor(r), r.PathValue("project"))
		if err != nil {
			problem(w, err)
			return
		}
		if !value.HasImage {
			problem(w, storage.ErrNotFound)
			return
		}
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value.Cover.Image, "data:image/jpeg;base64,"))
		if err != nil {
			problem(w, err)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(data)
	}))
}
