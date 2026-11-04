package httpapi

import "net/http"

func (s *Server) testHistoryRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/suites/{suite}/tests/{test}"
	m.HandleFunc("GET "+base+"/retained", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Catalog.RetainedTest(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("test"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/run-history", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 20)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Runs.TestHistory(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("test"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
}
