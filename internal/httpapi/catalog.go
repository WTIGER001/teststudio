package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) catalogRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/suites/{suite}"
	m.HandleFunc("GET /api/v1/projects/{project}/tests", s.authorized(s.projectTests))
	m.HandleFunc("GET "+base+"/tests", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.Catalog.Tests(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": rows})
	}))
	m.HandleFunc("POST "+base+"/tests", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var v catalog.Test
		if err := body(r, &v); err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		if err := s.Catalog.SaveTest(r.Context(), actor(r), p, suite, v, 0); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("tests", p, suite), v.ID)
	}))
	m.HandleFunc("GET "+base+"/tests/{test}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		p, suite, id := r.PathValue("project"), r.PathValue("suite"), r.PathValue("test")
		if err := s.Auth.Check(r.Context(), actor(r), p, "test/"+suite+"/"+id, "catalog.read"); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("tests", p, suite), id)
	}))
	m.HandleFunc("PUT "+base+"/tests/{test}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var v catalog.Test
		if err := body(r, &v); err != nil {
			problem(w, err)
			return
		}
		p, suite, id := r.PathValue("project"), r.PathValue("suite"), r.PathValue("test")
		if v.ID != id {
			problem(w, errors.New("test identifier is permanent"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		if err := s.Catalog.SaveTest(r.Context(), actor(r), p, suite, v, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("tests", p, suite), id)
	}))
	m.HandleFunc("GET "+base+"/tests/{test}/revisions", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		p, suite, id := r.PathValue("project"), r.PathValue("suite"), r.PathValue("test")
		if err := s.Auth.Check(r.Context(), actor(r), p, "test/"+suite+"/"+id, "catalog.read"); err != nil {
			problem(w, err)
			return
		}
		rows, err := s.Store.All(r.Context(), catalog.Bucket("test-revisions", p, suite)+"/"+id, p)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"items": rows})
	}))
	m.HandleFunc("POST "+base+"/tests/{test}/actions/{action}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		p, suite, id := r.PathValue("project"), r.PathValue("suite"), r.PathValue("test")
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		state := map[string]string{"enable": "enabled", "disable": "disabled", "archive": "archived", "restore": "enabled"}[r.PathValue("action")]
		if err := s.Catalog.ChangeState(r.Context(), actor(r), p, suite, id, state, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("tests", p, suite), id)
	}))
	m.HandleFunc("GET "+base+"/release-registry", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Catalog.ReleaseRegistry(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	for _, kind := range []string{"drafts", "releases"} {
		kind := kind
		route := map[string]string{"drafts": "catalog-drafts", "releases": "releases"}[kind]
		m.HandleFunc("GET "+base+"/"+route, s.authorized(func(w http.ResponseWriter, r *http.Request) {
			p, suite := r.PathValue("project"), r.PathValue("suite")
			if err := s.Auth.Check(r.Context(), actor(r), p, "suite/"+suite, "catalog.read"); err != nil {
				problem(w, err)
				return
			}
			rows, err := s.Store.All(r.Context(), catalog.Bucket(kind, p, suite), p)
			if err != nil {
				problem(w, err)
				return
			}
			JSON(w, 200, map[string]any{"items": rows})
		}))
		m.HandleFunc("GET "+base+"/"+route+"/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
			p, suite := r.PathValue("project"), r.PathValue("suite")
			if err := s.Auth.Check(r.Context(), actor(r), p, "suite/"+suite, "catalog.read"); err != nil {
				problem(w, err)
				return
			}
			s.record(w, r, catalog.Bucket(kind, p, suite), r.PathValue("id"))
		}))
	}
	m.HandleFunc("POST "+base+"/catalog-drafts", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var artifact catalog.Artifact
		if err := body(r, &artifact); err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		d, err := s.Catalog.CreateDraft(r.Context(), actor(r), p, suite, artifact)
		if err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("drafts", p, suite), d.ID)
	}))
	m.HandleFunc("PUT "+base+"/catalog-drafts/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var d catalog.Draft
		if err := body(r, &d); err != nil {
			problem(w, err)
			return
		}
		if d.ID != r.PathValue("id") {
			problem(w, errors.New("draft identifier is permanent"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		if err := s.Catalog.SaveDraft(r.Context(), actor(r), p, suite, d, rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, catalog.Bucket("drafts", p, suite), d.ID)
	}))
	m.HandleFunc("POST "+base+"/catalog-drafts/{id}/validate", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		issues, err := s.Catalog.ValidateDraft(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, map[string]any{"issues": issues, "valid": len(issues) == 0})
	}))
	m.HandleFunc("POST "+base+"/catalog-drafts/{id}/publish", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Activate bool `json:"activate"`
		}
		if err := body(r, &input); err != nil {
			problem(w, err)
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		v, err := s.Catalog.Publish(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), r.Header.Get("Idempotency-Key"), rev, input.Activate)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, v)
	}))
	m.HandleFunc("POST "+base+"/releases/{id}/activate", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		p, suite := r.PathValue("project"), r.PathValue("suite")
		if err := s.Catalog.Activate(r.Context(), actor(r), p, suite, r.PathValue("id"), rev); err != nil {
			problem(w, err)
			return
		}
		s.record(w, r, projects.Bucket("suites", p), suite)
	}))
}

func (s *Server) projectTests(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("project")
	if err := s.Auth.Check(r.Context(), actor(r), p, "", "catalog.read"); err != nil {
		problem(w, err)
		return
	}
	suites, err := s.Store.All(r.Context(), projects.Bucket("suites", p), p)
	if err != nil {
		problem(w, err)
		return
	}
	items := []map[string]any{}
	for _, row := range suites {
		v, err := storage.Decode[projects.Suite](row)
		if err != nil {
			problem(w, err)
			return
		}
		if v.Archived {
			continue
		}
		rows, err := s.Catalog.Tests(r.Context(), actor(r), p, row.ID)
		if err != nil {
			problem(w, err)
			return
		}
		for _, test := range rows {
			items = append(items, map[string]any{"id": test.ID, "suiteId": row.ID, "suiteName": v.Name, "revision": test.Revision, "data": test.Data, "updatedAt": test.Updated})
		}
	}
	JSON(w, 200, map[string]any{"items": items})
}
