package httpapi

import (
	"errors"
	"github.com/wtiger001/teststudio/internal/probes"
	"net/http"
	"strconv"
)

func (s *Server) probeService() *probes.Service {
	return &probes.Service{Store: s.Store, Auth: s.Auth, Catalog: s.Catalog, Runs: s.Runs, Schedules: s.Schedules}
}
func (s *Server) probeRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/probes"
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		hours := 24
		if raw := r.URL.Query().Get("hours"); raw != "" {
			hours, err = strconv.Atoi(raw)
			if err != nil {
				problem(w, errors.New("invalid observation window"))
				return
			}
		}
		q := r.URL.Query()
		page, err := s.probeService().List(r.Context(), actor(r), r.PathValue("project"), q.Get("cursor"), probes.Query{Search: q.Get("search"), Application: q.Get("application"), Stage: q.Get("stage"), State: q.Get("state"), Freshness: q.Get("freshness"), Schedule: q.Get("schedule"), Archived: q.Get("archived") == "true", Limit: limit, Hours: hours})
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("POST "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var v probes.Binding
		if err := body(r, &v); err != nil {
			problem(w, err)
			return
		}
		row, err := s.probeService().Save(r.Context(), actor(r), r.PathValue("project"), v, 0)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("GET "+base+"/{probe}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		_, row, err := s.probeService().Read(r.Context(), actor(r), r.PathValue("project"), r.PathValue("probe"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("PUT "+base+"/{probe}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var v probes.Binding
		if err := body(r, &v); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if rev < 1 || v.ID != r.PathValue("probe") {
			problem(w, errors.New("existing probe identifier and positive revision are required"))
			return
		}
		row, err := s.probeService().Save(r.Context(), actor(r), r.PathValue("project"), v, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
}
