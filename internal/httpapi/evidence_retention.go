package httpapi

import (
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/evidence"
)

func (s *Server) evidenceRetentionRoutes(m *http.ServeMux) {
	if s.Runs == nil || s.Evidence == nil {
		return
	}
	base := "/api/v1/projects/{project}/runs/{run}"
	m.HandleFunc("GET "+base+"/retention", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedRun(w, r, "evidence.read")
		if !ok {
			return
		}
		result, err := s.Evidence.Retention(r.Context(), v.Project, v.ID)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("PUT "+base+"/retention/pin", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedRun(w, r, "evidence.retain")
		if !ok {
			return
		}
		var input struct {
			Pinned bool       `json:"pinned"`
			Until  *time.Time `json:"until"`
			Reason string     `json:"reason"`
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
		if err := s.Evidence.SavePin(r.Context(), actor(r), v.Project, v.ID, input.Pinned, input.Until, input.Reason, rev); err != nil {
			problem(w, err)
			return
		}
		result, err := s.Evidence.Retention(r.Context(), v.Project, v.ID)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("DELETE "+base+"/artifacts/{artifact}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedRun(w, r, "evidence.delete")
		if !ok {
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
		if err := s.Evidence.RequestDelete(r.Context(), actor(r), v.Project, v.ID, r.PathValue("artifact"), input.Reason, rev); err != nil {
			problem(w, err)
			return
		}
		row, err := s.Store.Get(r.Context(), evidence.Bucket(v.ID), r.PathValue("artifact"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 202, row)
	}))
}
