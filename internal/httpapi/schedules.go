package httpapi

import (
	"errors"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/storage"
	"net/http"
	"strconv"
)

func (s *Server) scheduleRoutes(m *http.ServeMux) {
	if s.Schedules == nil {
		return
	}
	base := "/api/v1/projects/{project}/schedules"
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Schedules.List(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("cursor"), limit, r.URL.Query().Get("archived") == "true")
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("POST "+base+"/preview", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var d schedules.Definition
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		preview, err := s.Schedules.Preview(r.Context(), actor(r), r.PathValue("project"), d)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, preview)
	}))
	m.HandleFunc("POST "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var d schedules.Definition
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		project := r.PathValue("project")
		if err := s.Schedules.Save(r.Context(), actor(r), project, d, 0); err != nil {
			problem(w, err)
			return
		}
		s.scheduleRecord(w, r, project, d.ID)
	}))
	m.HandleFunc("GET "+base+"/{schedule}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		_, row, err := s.Schedules.Read(r.Context(), actor(r), r.PathValue("project"), r.PathValue("schedule"))
		if err != nil {
			problem(w, err)
			return
		}
		w.Header().Set("ETag", `"`+strconv.FormatInt(row.Revision, 10)+`"`)
		JSON(w, 200, row)
	}))
	m.HandleFunc("PUT "+base+"/{schedule}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var d schedules.Definition
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		if d.ID != r.PathValue("schedule") {
			problem(w, errors.New("schedule identifier is permanent"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if rev < 1 {
			problem(w, errors.New("existing schedules require a positive revision"))
			return
		}
		project := r.PathValue("project")
		if err = s.Schedules.Save(r.Context(), actor(r), project, d, rev); err != nil {
			problem(w, err)
			return
		}
		s.scheduleRecord(w, r, project, d.ID)
	}))
	m.HandleFunc("GET "+base+"/{schedule}/firings", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Schedules.History(r.Context(), actor(r), r.PathValue("project"), r.PathValue("schedule"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	for _, action := range []string{"archive", "restore"} {
		m.HandleFunc("POST "+base+"/{schedule}/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			project, id := r.PathValue("project"), r.PathValue("schedule")
			if err = s.Schedules.Archive(r.Context(), actor(r), project, id, rev, action == "archive"); err != nil {
				problem(w, err)
				return
			}
			s.scheduleRecord(w, r, project, id)
		}))
	}
}

func (s *Server) scheduleRecord(w http.ResponseWriter, r *http.Request, project, id string) {
	row, err := s.Store.Get(r.Context(), schedules.Bucket(project), id)
	if err != nil {
		problem(w, err)
		return
	}
	v, err := storage.Decode[schedules.Schedule](row)
	if err != nil {
		problem(w, err)
		return
	}
	row.Revision = v.Generation
	w.Header().Set("ETag", `"`+strconv.FormatInt(row.Revision, 10)+`"`)
	JSON(w, 200, row)
}
