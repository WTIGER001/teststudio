package httpapi

import (
	"net/http"

	"github.com/wtiger001/teststudio/internal/sources"
)

func (s *Server) repositoryMetadataRoutes(m *http.ServeMux, service *sources.Service) {
	base := "/api/v1/projects/{project}/repository-sources/{source}"
	m.HandleFunc("POST "+base+"/preview", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input sources.RepositoryInspectionRequest
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		value, err := service.PreviewRepository(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))

	m.HandleFunc("GET "+base+"/metadata/{suite}/history", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := service.RepositoryMetadataHistory(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.PathValue("suite"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/access-status", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.RepositoryAccessStatus(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/metadata/{suite}/remote", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.RemoteRepositoryMetadata(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST "+base+"/inspect", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.InspectRepository(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST "+base+"/access-check", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Credential string `json:"credentialReference,omitempty"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		value, err := service.CheckRepositoryAccess(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), input.Credential)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST "+base+"/rotate-credential", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.RotateRepositoryCredential(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.Header.Get("Idempotency-Key"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/metadata/{suite}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.ReadRepositoryMetadata(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("PUT "+base+"/metadata/{suite}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input sources.RepositoryMetadata
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		err = service.SaveRepositoryMetadata(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.PathValue("suite"), input, rev)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := service.ReadRepositoryMetadata(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST "+base+"/metadata/{suite}/commit", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := service.CommitRepositoryMetadata(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.PathValue("suite"), r.Header.Get("Idempotency-Key"), rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
}
