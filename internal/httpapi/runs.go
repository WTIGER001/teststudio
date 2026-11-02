package httpapi

import (
	"encoding/json"
	"github.com/wtiger001/teststudio/internal/workerstate"
	"net/http"
	"strconv"
	"time"

	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func pageLimit(r *http.Request, defaultLimit int) (int, error) {
	if r.URL.Query().Get("limit") == "" {
		return defaultLimit, nil
	}
	return strconv.Atoi(r.URL.Query().Get("limit"))
}

func (s *Server) authorizedRun(w http.ResponseWriter, r *http.Request, permission string) (runs.Run, bool) {
	v, _, err := s.Runs.Get(r.Context(), r.PathValue("project"), r.PathValue("run"))
	if err != nil {
		problem(w, err)
		return v, false
	}
	if err := s.Auth.Check(r.Context(), actor(r), v.Project, "run/"+v.ID, permission); err != nil {
		problem(w, err)
		return v, false
	}
	return v, true
}

func (s *Server) runRoutes(m *http.ServeMux) {
	if s.Runs == nil {
		return
	}
	base := "/api/v1/projects/{project}/runs"
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		q := r.URL.Query()
		f := runs.Filter{Environment: q.Get("environment"), Suite: q.Get("suite"), Application: q.Get("application"), State: q.Get("state"), Stage: q.Get("stage"), Selection: q.Get("selection")}
		if value := q.Get("since"); value != "" {
			f.Since, err = time.Parse(time.RFC3339, value)
			if err != nil {
				problem(w, err)
				return
			}
		}
		if value := q.Get("until"); value != "" {
			f.Until, err = time.Parse(time.RFC3339, value)
			if err != nil {
				problem(w, err)
				return
			}
		}
		page, err := s.Runs.List(r.Context(), actor(r), r.PathValue("project"), f, q.Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("POST "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input runs.Request
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		v, err := s.Runs.Create(r.Context(), actor(r), r.PathValue("project"), input, r.Header.Get("Idempotency-Key"), "manual")
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, "runs", v.ID)
	}))
	m.HandleFunc("GET "+base+"/{run}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedRun(w, r, "runs.read")
		if !ok {
			return
		}
		s.record(w, r, "runs", v.ID)
	}))
	m.HandleFunc("POST "+base+"/{run}/cancel", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedRun(w, r, "runs.cancel")
		if !ok {
			return
		}
		if _, err := s.Runs.Cancel(r.Context(), actor(r), v.Project, v.ID); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, "runs", v.ID)
	}))
	m.HandleFunc("POST "+base+"/{run}/rerun-failures", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedRun(w, r, "runs.read")
		if !ok {
			return
		}
		child, err := s.Runs.RerunFailures(r.Context(), actor(r), v.Project, v.ID, r.Header.Get("Idempotency-Key"))
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, "runs", child.ID)
	}))
	for _, kind := range []string{"attempts", "events", "artifacts", "executions"} {
		kind := kind
		m.HandleFunc("GET "+base+"/{run}/"+kind, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			permission := "runs.read"
			if kind == "events" || kind == "artifacts" {
				permission = "evidence.read"
			}
			v, ok := s.authorizedRun(w, r, permission)
			if !ok {
				return
			}
			limit, err := pageLimit(r, 100)
			if err != nil {
				problem(w, err)
				return
			}
			page, err := s.Store.List(r.Context(), kind+"/"+v.ID, v.Project, r.URL.Query().Get("cursor"), limit)
			if err != nil {
				problem(w, err)
				return
			}
			if kind == "executions" {
				for i, row := range page.Items {
					value, err := storage.Decode[workerstate.Record](row)
					if err != nil {
						problem(w, err)
						return
					}
					value.Claim = ""
					page.Items[i].Data, _ = json.Marshal(value)
				}
			}
			JSON(w, 200, page)
		}))
	}
	m.HandleFunc("GET "+base+"/{run}/logs", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.authorizedRun(w, r, "evidence.read")
		if !ok {
			return
		}
		limit, err := pageLimit(r, 50)
		if err != nil {
			problem(w, err)
			return
		}
		if s.Evidence == nil {
			JSON(w, 503, map[string]string{"message": "Evidence storage is unavailable."})
			return
		}
		q := r.URL.Query()
		page, err := s.Evidence.Logs(r.Context(), v.Project, v.ID, q.Get("stream"), q.Get("test"), q.Get("cursor"), q.Get("tail") == "true", limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("POST "+base+"/{run}/artifacts/{artifact}/access", s.authorized(s.artifactAccess))
}
