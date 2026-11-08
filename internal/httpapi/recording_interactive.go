package httpapi

import (
	"net/http"

	"github.com/wtiger001/teststudio/internal/recording"
)

func (s *Server) interactiveRecordingRoutes(m *http.ServeMux, service *recording.Service) {
	base := "/api/v1/projects/{project}/suites/{suite}/browser-recordings/{id}"
	for _, action := range []string{"view", "input"} {
		m.HandleFunc("POST "+base+"/"+action, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var input *recording.Action
			if action == "input" {
				input = &recording.Action{}
				if err := body(r, input); err != nil {
					problem(w, err)
					return
				}
			}
			view, err := service.Interactive(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), input)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, view)
		}))
	}
}
