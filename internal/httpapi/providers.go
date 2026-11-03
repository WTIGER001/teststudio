package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/access"
)

func (s *Server) providerRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/v1/identity-providers", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.accessService().Providers(r.Context(), actor(r), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET /api/v1/identity-providers/{provider}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.accessService().ReadProvider(r.Context(), actor(r), r.PathValue("provider"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	for _, method := range []string{"POST", "PUT"} {
		path := "/api/v1/identity-providers"
		if method == "PUT" {
			path += "/{provider}"
		}
		m.HandleFunc(method+" "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			var input access.ProviderInput
			if err := body(r, &input); err != nil {
				problem(w, err)
				return
			}
			var rev int64
			if method == "PUT" {
				var err error
				rev, err = revision(r)
				if err != nil || rev < 1 {
					problem(w, errors.New("existing providers require a positive If-Match revision"))
					return
				}
				if input.ID != r.PathValue("provider") {
					problem(w, errors.New("provider identifier is permanent"))
					return
				}
			}
			value, err := s.accessService().SaveProvider(r.Context(), actor(r), input, rev)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, value)
		}))
	}
}
