package httpapi

import (
	"net/http"

	"github.com/wtiger001/teststudio/internal/cliauth"
	"github.com/wtiger001/teststudio/internal/sessions"
)

func (s *Server) cliAuthenticationRoutes(m *http.ServeMux) {
	service := &cliauth.Service{Sessions: s.Sessions}
	m.HandleFunc("GET /auth/cli/options", func(w http.ResponseWriter, r *http.Request) { JSON(w, 200, service.Options()) })
	m.HandleFunc("GET /auth/cli/requests/{request}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		current := r.Context().Value(sessionKey{}).(sessions.Session)
		value, err := service.Read(r.Context(), current.ID, r.PathValue("request"), r.URL.Query().Get("clientName"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST /auth/cli/requests/{request}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input cliauth.Confirmation
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		current := r.Context().Value(sessionKey{}).(sessions.Session)
		value, err := service.Confirm(r.Context(), current.ID, r.PathValue("request"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	for _, operation := range []string{"exchange", "cancel"} {
		m.HandleFunc("POST /auth/cli/"+operation, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Origin") != s.Sessions.Origin {
				JSON(w, 403, map[string]string{"message": "request origin is not allowed"})
				return
			}
			var input cliauth.Proof
			if err := body(r, &input); err != nil {
				problem(w, err)
				return
			}
			if operation == "cancel" {
				if err := service.Cancel(r.Context(), input); err != nil {
					problem(w, err)
					return
				}
				JSON(w, 200, map[string]string{"state": "cancelled"})
				return
			}
			value, err := service.Exchange(r.Context(), input)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, value)
		})
	}
}
