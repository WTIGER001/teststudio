package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/sources"
)

func (s *Server) bundleRoutes(m *http.ServeMux, service *sources.Service) {
	base := "/api/v1/projects/{project}/suites/{suite}"
	m.HandleFunc("POST "+base+"/source-bundles", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || contentType != "application/zip" {
			problem(w, errors.New("source bundle body must use application/zip"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, sourcebundle.CompressedLimit)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			problem(w, errors.New("source bundle exceeds its 32 MiB limit or cannot be read"))
			return
		}
		row, err := service.EnqueueBundle(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.Header.Get("Idempotency-Key"), r.URL.Query().Get("name"), r.URL.Query().Get("manifest"), data, r.URL.Query().Get("save") == "true")
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, row)
	}))
	m.HandleFunc("GET "+base+"/source-preparations/{id}/bundle", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		bundle, err := service.ReadBundle(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": bundle.Name}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("ETag", `"`+bundle.Digest+`"`)
		w.WriteHeader(200)
		_, _ = w.Write(bundle.Content)
	}))
}
