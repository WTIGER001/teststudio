package httpapi

import (
	"errors"
	"net/http"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
)

func (s *Server) catalogRoutes(m *http.ServeMux) {
	base := "/api/v1/projects/{project}/suites/{suite}"
	m.HandleFunc("POST "+base+"/tests/{test}/save", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var value catalog.Test
		if err := body(r, &value); err != nil {
			problem(w, err)
			return
		}
		if value.ID != r.PathValue("test") {
			problem(w, errors.New("the test identifier is permanent"))
			return
		}
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		project, suite := r.PathValue("project"), r.PathValue("suite")
		saved, err := s.Catalog.SaveCurrentTestRecord(r.Context(), actor(r), project, suite, value, rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, saved)
	}))

	m.HandleFunc("GET /api/v1/projects/{project}/tests", s.authorized(s.projectTests))
	m.HandleFunc("GET "+base+"/tests", s.authorized(s.catalogList))
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
		saved, err := s.Catalog.ReadSavedTest(r.Context(), actor(r), p, suite, id)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, saved)
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
		var stateErr error
		if r.URL.Query().Get("save") == "true" {
			stateErr = s.Catalog.SaveLifecycle(r.Context(), actor(r), p, suite, id, state, rev)
		} else {
			stateErr = s.Catalog.ChangeState(r.Context(), actor(r), p, suite, id, state, rev)
		}
		if err := stateErr; err != nil {
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
	m.HandleFunc("POST "+base+"/catalog-drafts/{id}/publication-reviews", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		rev, err := revision(r)
		if err != nil {
			problem(w, err)
			return
		}
		value, err := s.Catalog.ReviewPublication(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), rev)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("GET "+base+"/publication-reviews/{id}", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Catalog.ReadPublicationReview(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, value)
	}))
	m.HandleFunc("POST "+base+"/catalog-drafts/{id}/publish", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Activate bool   `json:"activate"`
			Review   string `json:"reviewId"`
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
		var v catalog.Release
		if input.Review != "" {
			v, err = s.Catalog.PublishReviewed(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), input.Review, r.Header.Get("Idempotency-Key"), rev, input.Activate)
		} else {
			v, err = s.Catalog.Publish(r.Context(), actor(r), r.PathValue("project"), r.PathValue("suite"), r.PathValue("id"), r.Header.Get("Idempotency-Key"), rev, input.Activate)
		}
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

func (s *Server) projectTests(w http.ResponseWriter, r *http.Request) { s.catalogList(w, r) }
