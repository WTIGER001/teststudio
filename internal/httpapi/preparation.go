package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/wtiger001/teststudio/internal/sources"
)

func (s *Server) preparationRoutes(m *http.ServeMux, service *sources.Service) {
	base := "/api/v1/projects/{project}/suites/{suite}"
	m.HandleFunc("GET "+base+"/source-preparation-options", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		options, err := service.Options(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, options)
	}))
	m.HandleFunc("GET "+base+"/source-preparations", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit := 30
		if input := r.URL.Query().Get("limit"); input != "" {
			value, err := strconv.Atoi(input)
			if err != nil {
				problem(w, errors.New("invalid preparation page limit"))
				return
			}
			limit = value
		}
		page, err := service.ListJobs(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("POST "+base+"/source-preparations", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input sources.PrebuiltInput
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		row, err := service.EnqueuePrebuilt(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.Header.Get("Idempotency-Key"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("GET "+base+"/source-preparations/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := service.ReadJob(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("POST "+base+"/source-preparations/{id}/cancel", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		revision, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		p, suite, id := r.PathValue("project"), r.PathValue("suite"), r.PathValue("id")
		if err := service.CancelJob(r.Context(), actor(r), p, suite, id, revision); err != nil {
			problem(w, err)
			return
		}
		row, err := service.ReadJob(r.Context(), actor(r), p, suite, id)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
}
