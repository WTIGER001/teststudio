package httpapi

import (
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"io"
	"net/http"

	"github.com/wtiger001/teststudio/internal/sources"
)

func (s *Server) repositoryRoutes(m *http.ServeMux, service *sources.Service) {
	base := "/api/v1/projects/{project}/repository-sources"
	m.HandleFunc("POST "+base+"/{source}/webhooks/github", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			JSON(w, 413, map[string]string{"message": "Source webhook exceeds its body limit."})
			return
		}
		receipt, err := service.ReceiveRepositoryWebhook(r.Context(), r.PathValue("project"), r.PathValue("source"), sourceprovider.WebhookRequest{Delivery: r.Header.Get("X-GitHub-Delivery"), Event: r.Header.Get("X-GitHub-Event"), Signature: r.Header.Get("X-Hub-Signature-256"), Body: data})
		if err != nil {
			JSON(w, 403, map[string]string{"message": "Source webhook authentication or current configuration does not permit this delivery."})
			return
		}
		JSON(w, 202, receipt)
	})
	m.HandleFunc("GET /api/v1/projects/{project}/repository-source-options", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.RepositoryOptions(r.Context(), actor(r), r.PathValue("project"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base, s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := service.ListRepositories(r.Context(), actor(r), r.PathValue("project"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	m.HandleFunc("GET "+base+"/{source}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.ReadRepository(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("PUT "+base+"/{source}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input sources.RepositoryConfig
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		revision, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if err := service.SaveRepository(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), input, revision); err != nil {
			problem(w, err)
			return
		}
		row, err := service.ReadRepository(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("POST "+base+"/{source}/syncs", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input sources.SyncInput
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		value, err := service.EnqueueRepositorySync(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.Header.Get("Idempotency-Key"), input)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/{source}/syncs", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		limit, err := pageLimit(r, 30)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := service.ListRepositorySyncs(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.URL.Query().Get("cursor"), limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/{source}/syncs/{sync}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := service.ReadRepositorySync(r.Context(), actor(r), r.PathValue("project"), r.PathValue("source"), r.PathValue("sync"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
}
