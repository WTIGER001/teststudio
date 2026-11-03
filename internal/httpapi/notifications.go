package httpapi

import (
	"errors"
	"github.com/wtiger001/teststudio/internal/notifications"
	"net/http"
)

func (s *Server) notificationRoutes(m *http.ServeMux) {
	if s.Notifications == nil {
		return
	}
	base := "/api/v1/projects/{project}/notifications"
	m.HandleFunc("GET "+base+"/references", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		refs, err := s.Notifications.References(r.Context(), actor(r), r.PathValue("project"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": refs})
	}))
	m.HandleFunc("GET "+base+"/channels", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Notifications.ListChannels(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("cursor"), limit, r.URL.Query().Get("archived") == "true")
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("POST "+base+"/channels", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var d notifications.Definition
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		project := r.PathValue("project")
		if err := s.Notifications.SaveChannel(r.Context(), actor(r), project, d, 0); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, notifications.Channels(project), d.ID)
	}))
	m.HandleFunc("GET "+base+"/channels/{channel}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		_, _, err := s.Notifications.ReadChannel(r.Context(), actor(r), r.PathValue("project"), r.PathValue("channel"))
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, notifications.Channels(r.PathValue("project")), r.PathValue("channel"))
	}))
	m.HandleFunc("PUT "+base+"/channels/{channel}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var d notifications.Definition
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		if d.ID != r.PathValue("channel") {
			problem(w, errors.New("notification channel identifier is permanent"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if rev < 1 {
			problem(w, errors.New("existing channels require a positive revision"))
			return
		}
		project := r.PathValue("project")
		if err = s.Notifications.SaveChannel(r.Context(), actor(r), project, d, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, notifications.Channels(project), d.ID)
	}))
	for _, action := range []string{"archive", "restore"} {
		m.HandleFunc("POST "+base+"/channels/{channel}/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			project, id := r.PathValue("project"), r.PathValue("channel")
			if err = s.Notifications.Archive(r.Context(), actor(r), project, id, rev, action == "archive"); err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, notifications.Channels(project), id)
		}))
	}
	m.HandleFunc("GET "+base+"/deliveries", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Notifications.ListDeliveries(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("run"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET "+base+"/deliveries/{delivery}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		_, _, err := s.Notifications.ReadDelivery(r.Context(), actor(r), r.PathValue("project"), r.PathValue("delivery"))
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, notifications.Deliveries(r.PathValue("project")), r.PathValue("delivery"))
	}))
	m.HandleFunc("GET "+base+"/deliveries/{delivery}/attempts", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Notifications.History(r.Context(), actor(r), r.PathValue("project"), r.PathValue("delivery"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	for _, action := range []string{"retry", "cancel"} {
		m.HandleFunc("POST "+base+"/deliveries/{delivery}/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			project, id := r.PathValue("project"), r.PathValue("delivery")
			if action == "retry" {
				err = s.Notifications.Retry(r.Context(), actor(r), project, id, rev)
			} else {
				err = s.Notifications.Cancel(r.Context(), actor(r), project, id, rev)
			}
			if err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, notifications.Deliveries(project), id)
		}))
	}
}
