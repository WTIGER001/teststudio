package httpapi

import "net/http"

func (s *Server) healthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, 200, map[string]string{"status": "up", "service": "teststudio"})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Store.DB.PingContext(r.Context()); err != nil {
			JSON(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		JSON(w, 200, map[string]string{"status": "up", "service": "teststudio"})
	})
}
