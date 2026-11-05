package httpapi

import (
	"mime"
	"net/http"
	"strings"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/recording"
)

func companionRequest(r *http.Request) bool {
	content, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && content == "application/json" && r.Header.Get("Origin") == "" && r.Header.Get("Cookie") == "" && r.Header.Get("Sec-Fetch-Site") == ""
}

func (s *Server) companionRoutes(m *http.ServeMux, service *recording.Service) {
	base := "/api/v1/recording-companion/sessions/{id}"
	m.HandleFunc("POST "+base+"/pair", func(w http.ResponseWriter, r *http.Request) {
		if !companionRequest(r) || r.Header.Get("Authorization") != "" {
			problem(w, authorization.ErrForbidden)
			return
		}
		var input recording.PairRequest
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		packet, err := service.Pair(r.Context(), r.PathValue("id"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, packet)
	})
	for _, action := range []string{"control", "snapshots"} {
		m.HandleFunc("POST "+base+"/"+action, func(w http.ResponseWriter, r *http.Request) {
			value := r.Header.Get("Authorization")
			if !companionRequest(r) || !strings.HasPrefix(value, "Bearer ") || strings.Count(value, " ") != 1 {
				problem(w, authorization.ErrForbidden)
				return
			}
			var input struct {
				Instance string                       `json:"instance"`
				Snapshot *recording.CompanionSnapshot `json:"snapshot,omitempty"`
			}
			if err := body(r, &input); err != nil {
				problem(w, err)
				return
			}
			if (action == "snapshots") != (input.Snapshot != nil) {
				problem(w, authorization.ErrForbidden)
				return
			}
			var control recording.CompanionControl
			var err error
			if action == "snapshots" {
				control, err = service.SubmitCompanion(r.Context(), r.PathValue("id"), input.Instance, strings.TrimPrefix(value, "Bearer "), *input.Snapshot)
			} else {
				control, err = service.CompanionControl(r.Context(), r.PathValue("id"), input.Instance, strings.TrimPrefix(value, "Bearer "))
			}
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, control)
		})
	}
}
