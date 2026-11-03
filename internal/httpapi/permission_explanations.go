package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/access"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) permissionExplanationRoutes(m *http.ServeMux) {
	a := s.accessService()
	for _, path := range []string{"/api/v1/me/effective-permissions", "/api/v1/users/{user}/effective-permissions"} {
		m.HandleFunc("GET "+path, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			limit, err := pageLimit(r, 30)
			if err != nil {
				problem(w, err)
				return
			}
			q := r.URL.Query()
			user := r.PathValue("user")
			if user == "" {
				user = actor(r)
			}
			result, err := a.ExplainPermission(r.Context(), actor(r), user, access.ExplanationQuery{Project: q.Get("project"), Object: q.Get("object"), Permission: q.Get("permission"), Cursor: q.Get("cursor"), Limit: limit})
			if err != nil {
				permissionExplanationProblem(w, err)
				return
			}
			JSON(w, 200, result)
		}))
	}
}

func permissionExplanationProblem(w http.ResponseWriter, err error) {
	for _, known := range []error{authorization.ErrForbidden, storage.ErrNotFound, storage.ErrConflict} {
		if errors.Is(err, known) {
			problem(w, known)
			return
		}
	}
	problem(w, errors.New("The permission explanation could not be completed. Check the inputs and refresh current access."))
}
