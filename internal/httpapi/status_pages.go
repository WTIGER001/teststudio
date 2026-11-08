package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/statuspages"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) statusPageService() *statuspages.Service {
	return &statuspages.Service{Store: s.Store, Auth: s.Auth, Probes: s.probeService()}
}

func (s *Server) statusPageRoutes(m *http.ServeMux) {
	slots := make(chan struct{}, 4)
	base := "/api/v1/projects/{project}/status-pages"
	m.HandleFunc("GET /api/v1/projects/{project}/status-page-slugs/{slug}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		result, err := s.statusPageService().SlugAvailable(r.Context(), actor(r), r.PathValue("project"), r.PathValue("slug"), r.URL.Query().Get("pageId"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, result)
	}))
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		page, err := s.statusPageService().List(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("cursor"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET "+base+"/{page}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		_, row, err := s.statusPageService().Read(r.Context(), actor(r), r.PathValue("project"), r.PathValue("page"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("PUT "+base+"/{page}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var value statuspages.Definition
		if err := body(r, &value); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if value.ID != r.PathValue("page") {
			problem(w, errors.New("the status page identifier is permanent"))
			return
		}
		row, err := s.statusPageService().Save(r.Context(), actor(r), r.PathValue("project"), value, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	for _, action := range []string{"publish", "unpublish"} {
		m.HandleFunc("POST "+base+"/{page}/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			rev, err := revision(r)
			if err != nil {
				problem(w, err)
				return
			}
			row, err := s.statusPageService().Publish(r.Context(), actor(r), r.PathValue("project"), r.PathValue("page"), rev, action == "publish")
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, row)
		}))
	}
	m.HandleFunc("DELETE "+base+"/{page}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if err = s.statusPageService().Delete(r.Context(), actor(r), r.PathValue("project"), r.PathValue("page"), rev); err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]bool{"deleted": true})
	}))
	m.HandleFunc("POST "+base+"/{page}/preview", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		page, _, err := s.statusPageService().Read(r.Context(), actor(r), r.PathValue("project"), r.PathValue("page"))
		if err != nil {
			problem(w, err)
			return
		}
		report, err := s.statusPageService().Report(r.Context(), actor(r), r.PathValue("project"), page.Definition, r.URL.Query().Get("window"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, report)
	}))
	m.HandleFunc("GET /api/v1/public/status-pages/{token}", func(w http.ResponseWriter, r *http.Request) {
		if address := r.PathValue("token"); len(address) < 1 || len(address) > 63 || strings.ContainsAny(address, "/\\") {
			JSON(w, 404, map[string]string{"message": "This status page is unavailable."})
			return
		}
		if value := r.URL.Query().Get("window"); value != "" {
			if _, err := statuspages.WindowStart(value, time.Now()); err != nil {
				problem(w, err)
				return
			}
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			w.Header().Set("Retry-After", "5")
			JSON(w, 503, map[string]string{"message": "Status information is busy. Retry shortly."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		report, err := s.statusPageService().Public(ctx, r.PathValue("token"), r.URL.Query().Get("window"))
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				JSON(w, 503, map[string]string{"message": "Status history is temporarily unavailable. Retry shortly."})
				return
			}
			if errors.Is(err, storage.ErrNotFound) {
				JSON(w, 404, map[string]string{"message": "This status page is unavailable."})
				return
			}
			JSON(w, 503, map[string]string{"message": "Status information is temporarily unavailable."})
			return
		}
		JSON(w, 200, report)
	})
}
