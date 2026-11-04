package httpapi

import (
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/trends"
)

func (s *Server) trendRoutes(m *http.ServeMux) {
	service := trends.Service{Store: s.Store, Auth: s.Auth}
	m.HandleFunc("GET /api/v1/projects/{project}/trends", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := trends.Filter{Application: q.Get("application"), Environment: q.Get("environment"), Suite: q.Get("suite"), Category: q.Get("category"), Test: q.Get("test"), Location: q.Get("location")}
		var err error
		filter.Since, err = time.Parse(time.RFC3339, q.Get("since"))
		if err == nil {
			filter.Until, err = time.Parse(time.RFC3339, q.Get("until"))
		}
		if err != nil {
			problem(w, err)
			return
		}
		limit, err := pageLimit(r, 200)
		if err != nil {
			problem(w, err)
			return
		}
		report, err := service.Read(r.Context(), actor(r), r.PathValue("project"), filter, q.Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, report)
	}))
}
