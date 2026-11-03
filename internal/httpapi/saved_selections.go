package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/catalog"
)

func (s *Server) savedSelectionRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/suites/{suite}/saved-selections"
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Catalog.SavedSelections(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.URL.Query().Get("cursor"), limit, r.URL.Query().Get("archived") == "true")
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET "+base+"/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		_, row, err := s.Catalog.SavedSelection(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	save := s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input catalog.SavedSelection
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev := int64(0)
		if r.Method == "PUT" {
			var err error
			rev, err = revision(r)
			if err == nil && rev < 1 {
				err = errors.New("saved selection updates require a positive If-Match revision")
			}
			if err != nil {
				problem(w, err)
				return
			}
			if input.ID != r.PathValue("id") {
				problem(w, errors.New("saved selection identifier is permanent"))
				return
			}
		}
		project, suite := r.PathValue("project"), r.PathValue("suite")
		if err := s.Catalog.SaveSelection(r.Context(), actor(r), project, suite, input, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("saved-selections", project, suite), input.ID)
	})
	m.HandleFunc("POST "+base, save)
	m.HandleFunc("PUT "+base+"/{id}", save)
	m.HandleFunc("POST "+base+"/{id}/actions/{action}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		project, suite, id := r.PathValue("project"), r.PathValue("suite"), r.PathValue("id")
		if err := s.Catalog.ChangeSelectionState(r.Context(), actor(r), project, suite, id, r.PathValue("action"), rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("saved-selections", project, suite), id)
	}))
	m.HandleFunc("POST "+base+"/{id}/previews", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Release string `json:"releaseId,omitempty"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		preview, err := s.Catalog.PreviewSavedSelection(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), input.Release)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, preview)
	}))
}
