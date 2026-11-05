package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) recordingRoutes(m *http.ServeMux) {
	service := s.Recordings
	if service == nil {
		service = &recording.Service{Store: s.Store, Auth: s.Auth}
	}
	base := "/api/v1/projects/{project}/suites/{suite}/browser-recordings"
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := service.List(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET "+base+"/options", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		enabled, err := service.Available(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"enabled": enabled, "mode": service.Mode(), "durationLimitSeconds": int(recording.Lifetime.Seconds())})
	}))
	m.HandleFunc("POST "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input recording.Input
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		row, err := service.Start(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("GET "+base+"/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := service.Read(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("POST "+base+"/{id}/pairing", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.Pairing(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST "+base+"/{id}/actions/{action}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		action := r.PathValue("action")
		if action != "stop" && action != "cancel" {
			problem(w, errors.New("choose stop or cancel recording"))
			return
		}
		row, err := service.Stop(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), action == "cancel")
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("GET "+base+"/{id}/source", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		row, err := service.Read(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		value, err := storage.Decode[recording.Session](row)
		if err != nil {
			problem(w, err)
			return
		}
		draft, err := sources.RecordedDraft(value)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, draft)
	}))
	s.companionRoutes(m, service)
}
