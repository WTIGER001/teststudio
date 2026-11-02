package httpapi

import (
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/timing"
	"net/http"
)

func (s *Server) timingRoutes(m *http.ServeMux) {
	if s.Runs == nil {
		return
	}
	m.HandleFunc("POST /api/v1/projects/{project}/runs/estimate", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input runs.Request
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		estimate, err := s.Runs.Estimate(r.Context(), actor(r), r.PathValue("project"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, estimate)
	}))
	base := "/api/v1/projects/{project}/timing-baselines/{baseline}"
	authorize := func(w http.ResponseWriter, r *http.Request, permission string) (storage.Record, bool) {
		row, err := s.Store.Get(r.Context(), "timing-windows", r.PathValue("baseline"))
		if err != nil {
			problem(w, err)
			return row, false
		}
		v, err := storage.Decode[timing.Window](row)
		if err != nil {
			problem(w, err)
			return row, false
		}
		if v.Context.Project != r.PathValue("project") {
			problem(w, storage.ErrNotFound)
			return row, false
		}
		if err := s.Auth.Check(r.Context(), actor(r), v.Context.Project, "suite/"+v.Context.Suite, permission); err != nil {
			problem(w, err)
			return row, false
		}
		return row, true
	}
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := authorize(w, r, "runs.read"); !ok {
			return
		}
		s.record(w, r, "timing-windows", r.PathValue("baseline"))
	}))
	m.HandleFunc("POST "+base+"/reset", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := authorize(w, r, "catalog.publish"); !ok {
			return
		}
		var input struct {
			Reason string `json:"reason"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		service := timing.Service{Store: s.Store}
		if err := service.Reset(r.Context(), actor(r), r.PathValue("project"), r.PathValue("baseline"), input.Reason, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, "timing-windows", r.PathValue("baseline"))
	}))
}
