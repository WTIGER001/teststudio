package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/sources"
)

func (s *Server) playwrightDraftRoutes(m *http.ServeMux, service *sources.Service) {
	base := "/api/v1/projects/{project}/suites/{suite}/playwright-drafts"
	m.HandleFunc("GET "+base+"/starter", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		if _, err := service.Drafts(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), "", 1); err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, sources.StarterDraft())
	}))
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := service.Drafts(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET "+base+"/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := service.Draft(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("PUT "+base+"/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input sources.PlaywrightDraft
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		if input.ID != r.PathValue("id") {
			problem(w, errors.New("draft identifier is permanent"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		row, err := service.SaveDraft(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), input, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("POST "+base+"/{id}/prepare", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		row, err := service.PrepareDraft(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), r.Header.Get("Idempotency-Key"), rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
}
