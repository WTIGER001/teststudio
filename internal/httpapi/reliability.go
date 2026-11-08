package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/wtiger001/teststudio/internal/reliability"
)

func (s *Server) reliabilityRoutes(m *http.ServeMux) {
	s.reliabilityCheckRoutes(m)
	m.HandleFunc("GET /api/v1/projects/{project}/suites/{suite}/tests/{test}/reliability", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		q := reliability.Query{Scope: reliability.Scope{Project: r.PathValue("project"), Suite: r.PathValue("suite"), Test: r.PathValue("test"), Environment: params.Get("environment"), Location: params.Get("location")}, Resolution: params.Get("resolution")}
		var err error
		if params.Has("maxAgeSeconds") {
			q.MaxAge, err = strconv.Atoi(params.Get("maxAgeSeconds"))
			if err != nil {
				problem(w, err)
				return
			}
		}
		if params.Has("since") != params.Has("until") {
			JSON(w, 422, map[string]string{"message": "supply both since and until"})
			return
		}
		if params.Has("since") {
			q.Since, err = time.Parse(time.RFC3339Nano, params.Get("since"))
			if err == nil {
				q.Until, err = time.Parse(time.RFC3339Nano, params.Get("until"))
			}
			if err != nil {
				problem(w, err)
				return
			}
		}
		report, err := s.Runs.ReliabilityReport(r.Context(), actor(r), q, params.Get("window"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, report)
	}))
}
